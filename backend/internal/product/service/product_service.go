package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
	"payment-platform/pkg/eventbus"
)

type ProductService struct {
	repo         port.ProductRepository
	reservations port.ReservationRepository
	attributes   port.AttributeRepository
	variants     port.VariantRepository
	globalAttrs  port.GlobalAttributeRepository
	categories   port.CategoryRepository
	images       port.ImageRepository
	publisher    port.EventPublisher
	stores       port.StoreClient
	log          *slog.Logger
}

func New(repo port.ProductRepository, reservations port.ReservationRepository, attrs port.AttributeRepository, variants port.VariantRepository, globalAttrs port.GlobalAttributeRepository, categories port.CategoryRepository, images port.ImageRepository, pub port.EventPublisher, stores port.StoreClient, log *slog.Logger) *ProductService {
	return &ProductService{
		repo:         repo,
		reservations: reservations,
		attributes:   attrs,
		variants:     variants,
		globalAttrs:  globalAttrs,
		categories:   categories,
		images:       images,
		publisher:    pub,
		stores:       stores,
		log:          log,
	}
}

type AttributeInput struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type CreateRequest struct {
	Name          string
	Description   string
	SKU           string
	Price         int64 // cents
	Currency      string
	CategoryID    string
	Category      string
	SubcategoryID *string
	Stock         int
	StoreID       *string  // nil = platform product
	Images        []string // ordered image URLs; first becomes the thumbnail
	Attributes    []AttributeInput
	CallerID      string // user ID of the requester (for seller ownership check)
	CallerRole    string // "admin" | "seller"
}

type CreateVariantRequest struct {
	SKU             string
	Price           *int64
	Stock           int
	AttributeValues map[string]string
	CallerID        string
	CallerRole      string
}

type UpdateVariantRequest struct {
	SKU             *string
	Price           *int64 // explicit nil = clear override
	Stock           *int
	AttributeValues map[string]string
	CallerID        string
	CallerRole      string
}

type UpdateRequest struct {
	Name          *string
	Description   *string
	Price         *int64
	Stock         *int
	CategoryID    *string
	Category      *string
	SubcategoryID *string
	Images        *[]string        // nil = don't change; &[]string{} = clear; &[]string{"url"} = replace
	Attributes    []AttributeInput // nil = don't change, empty = clear all
	CallerID      string           // for seller ownership check
	CallerRole    string
}

type StockItem struct {
	ProductID string
	VariantID *string
	Quantity  int
}

type FacetValue struct {
	Value string
	Count int
}

type AttributeFacet struct {
	Name   string
	Values []FacetValue
}

type CategoryFacet struct {
	ID    string
	Name  string
	Count int
}

type ListResult struct {
	Products   []*domain.Product
	Total      int
	Categories []CategoryFacet
	Facets     []AttributeFacet
}

var ErrNotStoreOwner = errors.New("you can only add products to your own store")
var ErrSellerMustProvideStore = errors.New("sellers must provide a store_id")
var ErrSellerOnly = errors.New("only sellers can create/edit products")

func (s *ProductService) Create(ctx context.Context, req CreateRequest) (*domain.Product, error) {
	if req.CallerRole != "seller" {
		return nil, ErrSellerOnly
	}
	if req.StoreID == nil || *req.StoreID == "" {
		return nil, ErrSellerMustProvideStore
	}
	if err := s.checkStoreOwnership(ctx, *req.StoreID, req.CallerID); err != nil {
		return nil, err
	}

	imageURL := ""
	if len(req.Images) > 0 {
		imageURL = req.Images[0]
	}

	category, err := s.resolveCategory(ctx, req.CategoryID, req.Category)
	if err != nil {
		return nil, err
	}
	subcategory, err := s.resolveSubcategory(ctx, category.ID, req.SubcategoryID)
	if err != nil {
		return nil, err
	}

	p, err := domain.NewProduct(req.Name, req.Description, req.SKU, req.Price, req.Currency, category.Name, imageURL, req.Stock)
	if err != nil {
		return nil, err
	}
	p.CategoryID = category.ID
	p.Category = category.Name
	if subcategory != nil {
		p.SubcategoryID = &subcategory.ID
		p.Subcategory = subcategory.Name
	}
	p.StoreID = req.StoreID

	if err := s.repo.Create(ctx, p); err != nil {
		if errors.Is(err, domain.ErrSKUConflict) {
			return nil, domain.ErrSKUConflict
		}
		return nil, fmt.Errorf("create product: %w", err)
	}

	if len(req.Images) > 0 {
		imgs, err := s.images.SetImages(ctx, p.ID, req.Images)
		if err != nil {
			s.log.Error("failed to save product images", "product_id", p.ID, "err", err)
		} else {
			p.Images = imgs
		}
	}

	var subcatID, subcatName string
	if subcategory != nil {
		subcatID = subcategory.ID
		subcatName = subcategory.Name
	}
	attrs, err := s.buildSubcategoryAttributes(ctx, p.ID, subcatID, subcatName, req.Attributes)
	if err != nil {
		return nil, err
	}
	if len(attrs) > 0 {
		if err := s.attributes.SaveBatch(ctx, attrs); err != nil {
			return nil, fmt.Errorf("save attributes: %w", err)
		}
		p.Attributes = attrs
	}

	s.publishProductCreated(ctx, p)
	return p, nil
}

func (s *ProductService) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if imgs, err := s.images.GetByProductID(ctx, id); err == nil {
		p.Images = imgs
	}
	if attrs, err := s.attributes.ListByProduct(ctx, id); err == nil {
		p.Attributes = attrs
	}
	if variants, err := s.variants.ListByProduct(ctx, id); err == nil {
		p.Variants = variants
	}
	return p, nil
}

func (s *ProductService) GetByIDs(ctx context.Context, ids []string) ([]*domain.Product, error) {
	products, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, product := range products {
		if attrs, err := s.attributes.ListByProduct(ctx, product.ID); err == nil {
			product.Attributes = attrs
		}
		if variants, err := s.variants.ListByProduct(ctx, product.ID); err == nil {
			product.Variants = variants
		}
	}
	return products, nil
}

func (s *ProductService) List(ctx context.Context, f port.ListFilter) (*ListResult, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	products, _, err := s.repo.List(ctx, port.ListFilter{
		Status:  f.Status,
		StoreID: f.StoreID,
		Search:  f.Search,
	})
	if err != nil {
		return nil, err
	}

	if len(products) > 0 {
		ids := make([]string, len(products))
		for i, p := range products {
			ids[i] = p.ID
		}
		if imgMap, err := s.images.GetByProductIDs(ctx, ids); err == nil {
			for _, p := range products {
				p.Images = imgMap[p.ID]
			}
		}
		for _, product := range products {
			if attrs, err := s.attributes.ListByProduct(ctx, product.ID); err == nil {
				product.Attributes = attrs
			}
			if variants, err := s.variants.ListByProduct(ctx, product.ID); err == nil {
				product.Variants = variants
			}
		}
	}

	categoryIDFilter := strings.TrimSpace(f.CategoryID)
	if categoryIDFilter == "" && strings.TrimSpace(f.Category) != "" {
		if category, err := s.resolveCategory(ctx, "", f.Category); err == nil {
			categoryIDFilter = category.ID
		}
	}

	categories := s.buildCategoryFacets(products, f.AttributeValues)
	facets := buildAttributeFacets(products, categoryIDFilter, f.SubcategoryID, f.AttributeValues)
	filtered := filterCatalogProducts(products, categoryIDFilter, f.SubcategoryID, f.AttributeValues)
	total := len(filtered)

	if offset >= total {
		return &ListResult{
			Products:   []*domain.Product{},
			Total:      total,
			Categories: categories,
			Facets:     facets,
		}, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return &ListResult{
		Products:   filtered[offset:end],
		Total:      total,
		Categories: categories,
		Facets:     facets,
	}, nil
}

func (s *ProductService) Update(ctx context.Context, id string, req UpdateRequest) (*domain.Product, error) {
	if req.CallerRole != "seller" {
		return nil, ErrSellerOnly
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if p.StoreID == nil {
		return nil, ErrNotStoreOwner
	}
	if err := s.checkStoreOwnership(ctx, *p.StoreID, req.CallerID); err != nil {
		return nil, err
	}

	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Price != nil {
		if err := p.UpdatePrice(*req.Price); err != nil {
			return nil, err
		}
	}
	if req.Stock != nil {
		if *req.Stock < 0 {
			return nil, errors.New("stock cannot be negative")
		}
		p.Stock = *req.Stock
		if p.Stock == 0 {
			p.Status = domain.StatusOutOfStock
		} else if p.Status == domain.StatusOutOfStock {
			p.Status = domain.StatusActive
		}
	}
	categoryChanged := false
	if req.CategoryID != nil || req.Category != nil {
		categoryID := p.CategoryID
		categoryName := p.Category
		if req.CategoryID != nil {
			categoryID = *req.CategoryID
		}
		if req.Category != nil {
			categoryName = *req.Category
		}
		category, err := s.resolveCategory(ctx, categoryID, categoryName)
		if err != nil {
			return nil, err
		}
		categoryChanged = category.ID != p.CategoryID
		p.CategoryID = category.ID
		p.Category = category.Name
	}
	if req.SubcategoryID != nil || categoryChanged {
		subcategory, err := s.resolveSubcategory(ctx, p.CategoryID, req.SubcategoryID)
		if err != nil {
			return nil, err
		}
		p.SubcategoryID = nil
		p.Subcategory = ""
		if subcategory != nil {
			p.SubcategoryID = &subcategory.ID
			p.Subcategory = subcategory.Name
		}
	}
	if req.Images != nil {
		if len(*req.Images) > 0 {
			p.ImageURL = (*req.Images)[0]
		} else {
			p.ImageURL = ""
		}
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	if req.Images != nil {
		if imgs, err := s.images.SetImages(ctx, p.ID, *req.Images); err != nil {
			s.log.Error("failed to update product images", "product_id", p.ID, "err", err)
		} else {
			p.Images = imgs
		}
	}

	if req.Attributes != nil {
		var subcatID, subcatName string
		if p.SubcategoryID != nil {
			subcatID = *p.SubcategoryID
			subcatName = p.Subcategory
		}
		attrs, err := s.buildSubcategoryAttributes(ctx, p.ID, subcatID, subcatName, req.Attributes)
		if err != nil {
			return nil, err
		}
		if err := s.attributes.SaveBatch(ctx, attrs); err != nil {
			return nil, fmt.Errorf("save attributes: %w", err)
		}
		p.Attributes = attrs
	}

	return p, nil
}

func (s *ProductService) checkStoreOwnership(ctx context.Context, storeID, callerID string) error {
	ownerStoreID, err := s.stores.GetStoreIDByOwner(ctx, callerID)
	if err != nil {
		return fmt.Errorf("check store ownership: %w", err)
	}
	if ownerStoreID != storeID {
		return ErrNotStoreOwner
	}
	return nil
}

func (s *ProductService) Deactivate(ctx context.Context, id string) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	p.Deactivate()
	return s.repo.Update(ctx, p)
}

func (s *ProductService) ReserveStock(ctx context.Context, orderID string, items []StockItem) error {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ProductID
	}

	products, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("fetch products for reservation: %w", err)
	}

	byID := make(map[string]*domain.Product, len(products))
	for _, p := range products {
		byID[p.ID] = p
	}

	for _, item := range items {
		if item.VariantID != nil {
			variant, err := s.variants.GetByID(ctx, *item.VariantID)
			if err != nil {
				s.log.Warn("variant not found during stock release Ã¢â‚¬â€ skipping", "variant_id", *item.VariantID, "product_id", item.ProductID)
				continue
			}
			variant.ReleaseStock(item.Quantity)
			if err := s.variants.Update(ctx, variant); err != nil {
				s.log.Error("failed to persist variant stock release", "variant_id", variant.ID, "err", err)
			}
			continue
		}
		p, ok := byID[item.ProductID]
		if !ok {
			return fmt.Errorf("product %s not found", item.ProductID)
		}
		if p.Status == domain.StatusInactive {
			return fmt.Errorf("product %s (%s) is inactive", p.ID, p.Name)
		}
		if item.VariantID != nil {
			variant, err := s.variants.GetByID(ctx, *item.VariantID)
			if err != nil {
				return fmt.Errorf("get variant %s: %w", *item.VariantID, err)
			}
			if variant.ProductID != item.ProductID {
				return fmt.Errorf("variant %s does not belong to product %s", variant.ID, item.ProductID)
			}
			if variant.Status == domain.StatusInactive {
				return fmt.Errorf("variant %s is inactive", variant.ID)
			}
			if variant.Stock < item.Quantity {
				return fmt.Errorf("insufficient stock for variant %s: have %d, need %d",
					variant.ID, variant.Stock, item.Quantity)
			}
			continue
		}
		if p.Stock < item.Quantity {
			return fmt.Errorf("insufficient stock for product %s (%s): have %d, need %d",
				p.ID, p.Name, p.Stock, item.Quantity)
		}
	}

	for _, item := range items {
		p := byID[item.ProductID]
		if item.VariantID != nil {
			variant, err := s.variants.GetByID(ctx, *item.VariantID)
			if err != nil {
				return fmt.Errorf("get variant %s: %w", *item.VariantID, err)
			}
			if variant.ProductID != item.ProductID {
				return fmt.Errorf("variant %s does not belong to product %s", variant.ID, item.ProductID)
			}
			if err := variant.ReserveStock(item.Quantity); err != nil {
				return fmt.Errorf("reserve stock for variant %s: %w", variant.ID, err)
			}
			if err := s.variants.Update(ctx, variant); err != nil {
				return fmt.Errorf("persist stock for variant %s: %w", variant.ID, err)
			}
		} else {
			if err := p.ReserveStock(item.Quantity); err != nil {
				return fmt.Errorf("reserve stock for %s: %w", item.ProductID, err)
			}
			if err := s.repo.Update(ctx, p); err != nil {
				return fmt.Errorf("persist stock for %s: %w", item.ProductID, err)
			}
			if p.Status == domain.StatusOutOfStock {
				s.publishOutOfStock(ctx, p)
			}
		}
		res := domain.NewReservation(orderID, item.ProductID, item.VariantID, item.Quantity)
		if err := s.reservations.Save(ctx, res); err != nil {
			s.log.Error("failed to save reservation record", "order_id", orderID, "product_id", item.ProductID, "err", err)
		}
	}

	s.publishStockReserved(ctx, orderID, items)
	return nil
}

func (s *ProductService) CommitReservation(ctx context.Context, orderID string) error {
	if err := s.reservations.UpdateStatus(ctx, orderID, domain.ReservationCommitted); err != nil {
		s.log.Error("failed to commit reservation", "order_id", orderID, "err", err)
		return err
	}
	s.log.Info("reservation committed", "order_id", orderID)
	return nil
}

func (s *ProductService) ReleaseReservation(ctx context.Context, orderID, reason string) error {
	reservations, err := s.reservations.GetByOrderID(ctx, orderID)
	if err != nil {
		s.log.Error("no reservations found for order â€” cannot release stock",
			"order_id", orderID, "err", err)
		return nil
	}

	items := make([]StockItem, len(reservations))
	for i, r := range reservations {
		items[i] = StockItem{ProductID: r.ProductID, VariantID: r.VariantID, Quantity: r.Quantity}
	}

	if err := s.ReleaseStock(ctx, orderID, reason, items); err != nil {
		return err
	}

	return s.reservations.UpdateStatus(ctx, orderID, domain.ReservationReleased)
}

func (s *ProductService) ReleaseStock(ctx context.Context, orderID, reason string, items []StockItem) error {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ProductID
	}

	products, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("fetch products for release: %w", err)
	}

	byID := make(map[string]*domain.Product, len(products))
	for _, p := range products {
		byID[p.ID] = p
	}

	for _, item := range items {
		if item.VariantID != nil {
			variant, err := s.variants.GetByID(ctx, *item.VariantID)
			if err != nil {
				s.log.Warn("variant not found during stock release Ã¢â‚¬â€ skipping", "variant_id", *item.VariantID, "product_id", item.ProductID)
				continue
			}
			variant.ReleaseStock(item.Quantity)
			if err := s.variants.Update(ctx, variant); err != nil {
				s.log.Error("failed to persist variant stock release", "variant_id", variant.ID, "err", err)
			}
			continue
		}
		p, ok := byID[item.ProductID]
		if !ok {
			s.log.Warn("product not found during stock release â€” skipping", "product_id", item.ProductID)
			continue
		}
		p.ReleaseStock(item.Quantity)
		if err := s.repo.Update(ctx, p); err != nil {
			s.log.Error("failed to persist stock release", "product_id", item.ProductID, "err", err)
		}
	}

	s.publishStockReleased(ctx, orderID, reason, items)
	return nil
}

func (s *ProductService) publishProductCreated(ctx context.Context, p *domain.Product) {
	data, _ := json.Marshal(eventbus.ProductCreatedData{
		ProductID: p.ID,
		Name:      p.Name,
		SKU:       p.SKU,
		Price:     p.Price,
		Currency:  p.Currency,
		Stock:     p.Stock,
	})
	event := eventbus.NewEvent("product.created", p.ID, "product", data, eventbus.Metadata{})
	if err := s.publisher.Publish(ctx, eventbus.SubjectProductCreated, event); err != nil {
		s.log.Error("failed to publish product.created", "product_id", p.ID, "err", err)
	}
}

func (s *ProductService) publishStockReserved(ctx context.Context, orderID string, items []StockItem) {
	busItems := make([]eventbus.StockItem, len(items))
	for i, it := range items {
		busItems[i] = eventbus.StockItem{ProductID: it.ProductID, VariantID: it.VariantID, Quantity: it.Quantity}
	}
	data, _ := json.Marshal(eventbus.StockReservedData{OrderID: orderID, Items: busItems})
	event := eventbus.NewEvent("products.stock_reserved", orderID, "order", data, eventbus.Metadata{CorrelationID: orderID})
	if err := s.publisher.Publish(ctx, eventbus.SubjectStockReserved, event); err != nil {
		s.log.Error("failed to publish products.stock_reserved", "order_id", orderID, "err", err)
	}
}

func (s *ProductService) publishStockReleased(ctx context.Context, orderID, reason string, items []StockItem) {
	busItems := make([]eventbus.StockItem, len(items))
	for i, it := range items {
		busItems[i] = eventbus.StockItem{ProductID: it.ProductID, VariantID: it.VariantID, Quantity: it.Quantity}
	}
	data, _ := json.Marshal(eventbus.StockReleasedData{OrderID: orderID, Items: busItems, Reason: reason})
	event := eventbus.NewEvent("products.stock_released", orderID, "order", data, eventbus.Metadata{CorrelationID: orderID})
	if err := s.publisher.Publish(ctx, eventbus.SubjectStockReleased, event); err != nil {
		s.log.Error("failed to publish products.stock_released", "order_id", orderID, "err", err)
	}
}

func (s *ProductService) publishOutOfStock(ctx context.Context, p *domain.Product) {
	data, _ := json.Marshal(eventbus.ProductCreatedData{ProductID: p.ID, Name: p.Name, SKU: p.SKU})
	event := eventbus.NewEvent("products.out_of_stock", p.ID, "product", data, eventbus.Metadata{})
	if err := s.publisher.Publish(ctx, eventbus.SubjectProductOutOfStock, event); err != nil {
		s.log.Error("failed to publish products.out_of_stock", "product_id", p.ID, "err", err)
	}
}

func (s *ProductService) CreateVariant(ctx context.Context, productID string, req CreateVariantRequest) (*domain.Variant, error) {
	if req.CallerRole != "seller" {
		return nil, ErrSellerOnly
	}
	p, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.StoreID == nil {
		return nil, ErrNotStoreOwner
	}
	if err := s.checkStoreOwnership(ctx, *p.StoreID, req.CallerID); err != nil {
		return nil, err
	}
	if err := s.validateVariantSelection(ctx, p, req.AttributeValues, ""); err != nil {
		return nil, err
	}

	v, err := domain.NewVariant(productID, req.SKU, req.Price, req.Stock, req.AttributeValues)
	if err != nil {
		return nil, err
	}
	if err := s.variants.Create(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *ProductService) UpdateVariant(ctx context.Context, productID, variantID string, req UpdateVariantRequest) (*domain.Variant, error) {
	if req.CallerRole != "seller" {
		return nil, ErrSellerOnly
	}
	p, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.StoreID == nil {
		return nil, ErrNotStoreOwner
	}
	if err := s.checkStoreOwnership(ctx, *p.StoreID, req.CallerID); err != nil {
		return nil, err
	}

	v, err := s.variants.GetByID(ctx, variantID)
	if err != nil {
		return nil, err
	}
	if v.ProductID != productID {
		return nil, domain.ErrVariantNotFound
	}

	if req.SKU != nil {
		v.SKU = *req.SKU
	}
	if req.Price != nil {
		v.Price = req.Price
	}
	if req.Stock != nil {
		if *req.Stock < 0 {
			return nil, errors.New("stock cannot be negative")
		}
		v.Stock = *req.Stock
		if v.Stock == 0 {
			v.Status = domain.StatusOutOfStock
		} else if v.Status == domain.StatusOutOfStock {
			v.Status = domain.StatusActive
		}
	}
	if req.AttributeValues != nil {
		if err := s.validateVariantSelection(ctx, p, req.AttributeValues, v.ID); err != nil {
			return nil, err
		}
		v.AttributeValues = req.AttributeValues
	}
	v.UpdatedAt = v.UpdatedAt // trigger update timestamp via domain
	v.UpdatedAt = timeNow()

	if err := s.variants.Update(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *ProductService) DeleteVariant(ctx context.Context, productID, variantID string, callerID, callerRole string) error {
	if callerRole != "seller" {
		return ErrSellerOnly
	}
	p, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	if p.StoreID == nil {
		return ErrNotStoreOwner
	}
	if err := s.checkStoreOwnership(ctx, *p.StoreID, callerID); err != nil {
		return err
	}

	v, err := s.variants.GetByID(ctx, variantID)
	if err != nil {
		return err
	}
	if v.ProductID != productID {
		return domain.ErrVariantNotFound
	}
	return s.variants.Delete(ctx, variantID)
}

func (s *ProductService) ListVariants(ctx context.Context, productID string) ([]*domain.Variant, error) {
	return s.variants.ListByProduct(ctx, productID)
}

func timeNow() time.Time { return time.Now().UTC() }

func (s *ProductService) buildSubcategoryAttributes(ctx context.Context, productID, subcategoryID, subcategoryName string, inputs []AttributeInput) ([]*domain.Attribute, error) {
	// No subcategory → attributes not allowed
	if subcategoryID == "" {
		if len(inputs) > 0 {
			return nil, errors.New("attributes require a subcategory to be selected")
		}
		return nil, nil
	}

	definitions, err := s.globalAttrs.List(ctx, port.GlobalAttributeFilter{SubcategoryID: subcategoryID})
	if err != nil {
		return nil, fmt.Errorf("load subcategory attributes: %w", err)
	}

	// Subcategory has no defined attributes — skip validation
	if len(definitions) == 0 {
		if len(inputs) > 0 {
			return nil, fmt.Errorf("subcategory %q has no configured attributes", subcategoryName)
		}
		return nil, nil
	}

	byName := make(map[string]*domain.GlobalAttribute, len(definitions))
	for _, definition := range definitions {
		byName[strings.ToLower(definition.Name)] = definition
	}

	inputByName := make(map[string]AttributeInput, len(inputs))
	for _, input := range inputs {
		nameKey := strings.ToLower(strings.TrimSpace(input.Name))
		if nameKey == "" {
			return nil, errors.New("attribute name is required")
		}
		if _, ok := byName[nameKey]; !ok {
			return nil, fmt.Errorf("attribute %q is not configured for subcategory %q", input.Name, subcategoryName)
		}
		cleanValues := uniqueNonEmptyStrings(input.Values)
		if len(cleanValues) == 0 {
			return nil, fmt.Errorf("attribute %q must have at least one value", input.Name)
		}
		input.Name = strings.TrimSpace(input.Name)
		input.Values = cleanValues
		inputByName[nameKey] = input
	}

	attrs := make([]*domain.Attribute, 0, len(definitions))
	for index, definition := range definitions {
		input, ok := inputByName[strings.ToLower(definition.Name)]
		if !ok {
			return nil, fmt.Errorf("attribute %q is required for subcategory %q", definition.Name, subcategoryName)
		}
		attr, err := domain.NewAttribute(productID, &definition.ID, definition.Name, input.Values, index)
		if err != nil {
			return nil, fmt.Errorf("attribute %q: %w", definition.Name, err)
		}
		attrs = append(attrs, attr)
	}

	return attrs, nil
}

func (s *ProductService) validateVariantSelection(ctx context.Context, product *domain.Product, values map[string]string, currentVariantID string) error {
	attrs, err := s.attributes.ListByProduct(ctx, product.ID)
	if err != nil {
		return fmt.Errorf("load product attributes: %w", err)
	}
	if len(attrs) == 0 {
		if len(values) > 0 {
			return errors.New("attribute values are not allowed for a product without attributes")
		}
		return nil
	}
	if len(values) != len(attrs) {
		return errors.New("every product attribute must be selected for a variant")
	}

	allowed := make(map[string]map[string]struct{}, len(attrs))
	for _, attr := range attrs {
		valueSet := make(map[string]struct{}, len(attr.Values))
		for _, value := range attr.Values {
			valueSet[value] = struct{}{}
		}
		allowed[attr.Name] = valueSet
	}

	for name, value := range values {
		attrAllowed, ok := allowed[name]
		if !ok {
			return fmt.Errorf("attribute %q is not valid for this product", name)
		}
		if _, ok := attrAllowed[value]; !ok {
			return fmt.Errorf("value %q is not valid for attribute %q", value, name)
		}
	}

	existingVariants, err := s.variants.ListByProduct(ctx, product.ID)
	if err != nil {
		return fmt.Errorf("load product variants: %w", err)
	}
	for _, variant := range existingVariants {
		if currentVariantID != "" && variant.ID == currentVariantID {
			continue
		}
		if sameAttributeCombination(variant.AttributeValues, values) {
			return errors.New("that exact variant combination already exists")
		}
	}

	return nil
}

func sameAttributeCombination(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if right[key] != leftValue {
			return false
		}
	}
	return true
}

func filterCatalogProducts(products []*domain.Product, categoryID, subcategoryID string, filters map[string][]string) []*domain.Product {
	filtered := make([]*domain.Product, 0, len(products))
	for _, product := range products {
		if categoryID != "" && product.CategoryID != categoryID {
			continue
		}
		if subcategoryID != "" {
			if product.SubcategoryID == nil || *product.SubcategoryID != subcategoryID {
				continue
			}
		}
		if categoryID == "" && product.CategoryID == "" && strings.TrimSpace(product.Category) == "" {
			continue
		}
		if !productMatchesAttributeFilters(product, filters) {
			continue
		}
		filtered = append(filtered, product)
	}
	return filtered
}

func (s *ProductService) buildCategoryFacets(products []*domain.Product, filters map[string][]string) []CategoryFacet {
	type categoryCount struct {
		Name  string
		Count int
	}
	counts := make(map[string]categoryCount)
	for _, product := range products {
		if !productMatchesAttributeFilters(product, filters) {
			continue
		}
		if strings.TrimSpace(product.CategoryID) == "" || strings.TrimSpace(product.Category) == "" {
			continue
		}
		entry := counts[product.CategoryID]
		entry.Name = product.Category
		entry.Count++
		counts[product.CategoryID] = entry
	}

	categories := make([]CategoryFacet, 0, len(counts))
	for id, entry := range counts {
		categories = append(categories, CategoryFacet{ID: id, Name: entry.Name, Count: entry.Count})
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})
	return categories
}

func buildAttributeFacets(products []*domain.Product, categoryID, subcategoryID string, filters map[string][]string) []AttributeFacet {
	nameSet := make(map[string]struct{})
	for _, product := range products {
		if categoryID != "" && product.CategoryID != categoryID {
			continue
		}
		if subcategoryID != "" && (product.SubcategoryID == nil || *product.SubcategoryID != subcategoryID) {
			continue
		}
		for _, attribute := range product.Attributes {
			nameSet[attribute.Name] = struct{}{}
		}
	}

	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)

	facets := make([]AttributeFacet, 0, len(names))
	for _, name := range names {
		counts := make(map[string]int)
		filtersWithoutCurrent := make(map[string][]string, len(filters))
		for key, values := range filters {
			if key == name {
				continue
			}
			filtersWithoutCurrent[key] = values
		}

		for _, product := range products {
			if categoryID != "" && product.CategoryID != categoryID {
				continue
			}
			if subcategoryID != "" && (product.SubcategoryID == nil || *product.SubcategoryID != subcategoryID) {
				continue
			}
			if !productMatchesAttributeFilters(product, filtersWithoutCurrent) {
				continue
			}
			for _, value := range getFilterValuesForProduct(product, name) {
				counts[value]++
			}
		}

		values := make([]FacetValue, 0, len(counts))
		for value, count := range counts {
			values = append(values, FacetValue{Value: value, Count: count})
		}
		sort.Slice(values, func(i, j int) bool {
			return values[i].Value < values[j].Value
		})
		if len(values) == 0 {
			continue
		}
		facets = append(facets, AttributeFacet{Name: name, Values: values})
	}

	return facets
}

func productMatchesAttributeFilters(product *domain.Product, filters map[string][]string) bool {
	activeFilters := make(map[string][]string, len(filters))
	for name, values := range filters {
		if len(values) > 0 {
			activeFilters[name] = values
		}
	}
	if len(activeFilters) == 0 {
		return true
	}

	activeVariants := getPurchasableVariants(product)
	if len(activeVariants) > 0 {
		for _, variant := range activeVariants {
			matches := true
			for attributeName, values := range activeFilters {
				if !containsString(values, variant.AttributeValues[attributeName]) {
					matches = false
					break
				}
			}
			if matches {
				return true
			}
		}
		return false
	}

	attributesByName := make(map[string]map[string]struct{}, len(product.Attributes))
	for _, attribute := range product.Attributes {
		valueSet := make(map[string]struct{}, len(attribute.Values))
		for _, value := range attribute.Values {
			valueSet[value] = struct{}{}
		}
		attributesByName[attribute.Name] = valueSet
	}

	for attributeName, values := range activeFilters {
		available := attributesByName[attributeName]
		matched := false
		for _, value := range values {
			if _, ok := available[value]; ok {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func getFilterValuesForProduct(product *domain.Product, attributeName string) []string {
	activeVariants := getPurchasableVariants(product)
	if len(activeVariants) > 0 {
		seen := make(map[string]struct{})
		values := make([]string, 0, len(activeVariants))
		for _, variant := range activeVariants {
			value := variant.AttributeValues[attributeName]
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			values = append(values, value)
		}
		return values
	}

	for _, attribute := range product.Attributes {
		if attribute.Name == attributeName {
			return attribute.Values
		}
	}
	return nil
}

func getPurchasableVariants(product *domain.Product) []*domain.Variant {
	variants := make([]*domain.Variant, 0, len(product.Variants))
	for _, variant := range product.Variants {
		if variant.Status != domain.StatusActive || variant.Stock <= 0 {
			continue
		}
		if !variantMatchesProductAttributes(product.Attributes, variant) {
			continue
		}
		variants = append(variants, variant)
	}
	return variants
}

func variantMatchesProductAttributes(attributes []*domain.Attribute, variant *domain.Variant) bool {
	if len(attributes) == 0 {
		return len(variant.AttributeValues) == 0
	}
	if len(variant.AttributeValues) != len(attributes) {
		return false
	}
	for _, attribute := range attributes {
		value, ok := variant.AttributeValues[attribute.Name]
		if !ok {
			return false
		}
		if !containsString(attribute.Values, value) {
			return false
		}
	}
	return true
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *ProductService) resolveCategory(ctx context.Context, categoryID, categoryName string) (*domain.Category, error) {
	if s.categories == nil {
		return nil, errors.New("category repository is not configured")
	}

	categoryID = strings.TrimSpace(categoryID)
	categoryName = strings.TrimSpace(categoryName)

	if categoryID != "" {
		category, err := s.categories.GetCategoryByID(ctx, categoryID)
		if err != nil {
			if errors.Is(err, domain.ErrCategoryNotFound) {
				return nil, errors.New("category not found")
			}
			return nil, fmt.Errorf("load category: %w", err)
		}
		return category, nil
	}

	if categoryName == "" {
		return nil, errors.New("category_id is required")
	}

	categories, err := s.categories.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	for _, category := range categories {
		if strings.EqualFold(category.Name, categoryName) {
			return category, nil
		}
	}
	return nil, errors.New("category not found")
}

func (s *ProductService) resolveSubcategory(ctx context.Context, categoryID string, subcategoryID *string) (*domain.Subcategory, error) {
	if subcategoryID == nil {
		return nil, nil
	}

	trimmedID := strings.TrimSpace(*subcategoryID)
	if trimmedID == "" {
		return nil, nil
	}

	subcategory, err := s.categories.GetSubcategoryByID(ctx, trimmedID)
	if err != nil {
		if errors.Is(err, domain.ErrSubcategoryNotFound) {
			return nil, errors.New("subcategory not found")
		}
		return nil, fmt.Errorf("load subcategory: %w", err)
	}
	if subcategory.CategoryID != categoryID {
		return nil, errors.New("subcategory does not belong to the selected category")
	}
	return subcategory, nil
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}
