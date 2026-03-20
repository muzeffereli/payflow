package service

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"payment-platform/internal/product/domain"
	"payment-platform/internal/product/port"
)

type fakeProductRepo struct {
	product  *domain.Product
	products []*domain.Product
}

func (r *fakeProductRepo) Create(_ context.Context, p *domain.Product) error {
	r.product = p
	r.products = append(r.products, p)
	return nil
}

func (r *fakeProductRepo) GetByID(_ context.Context, id string) (*domain.Product, error) {
	for _, product := range r.products {
		if product.ID == id {
			return product, nil
		}
	}
	if r.product != nil && r.product.ID == id {
		return r.product, nil
	}
	return nil, domain.ErrProductNotFound
}

func (r *fakeProductRepo) GetByIDs(_ context.Context, ids []string) ([]*domain.Product, error) {
	result := make([]*domain.Product, 0, len(ids))
	for _, id := range ids {
		if product, err := r.GetByID(context.Background(), id); err == nil {
			result = append(result, product)
		}
	}
	return result, nil
}

func (r *fakeProductRepo) GetBySKU(_ context.Context, sku string) (*domain.Product, error) {
	for _, product := range r.products {
		if product.SKU == sku {
			return product, nil
		}
	}
	if r.product != nil && r.product.SKU == sku {
		return r.product, nil
	}
	return nil, domain.ErrProductNotFound
}

func (r *fakeProductRepo) List(_ context.Context, filter port.ListFilter) ([]*domain.Product, int, error) {
	source := r.products
	if len(source) == 0 && r.product != nil {
		source = []*domain.Product{r.product}
	}

	result := make([]*domain.Product, 0, len(source))
	for _, product := range source {
		if filter.Category != "" && product.Category != filter.Category {
			continue
		}
		if filter.CategoryID != "" && product.CategoryID != filter.CategoryID {
			continue
		}
		if filter.SubcategoryID != "" {
			if product.SubcategoryID == nil || *product.SubcategoryID != filter.SubcategoryID {
				continue
			}
		}
		if filter.Status != "" && string(product.Status) != filter.Status {
			continue
		}
		if filter.StoreID != "" {
			if product.StoreID == nil || *product.StoreID != filter.StoreID {
				continue
			}
		}
		if strings.TrimSpace(filter.Search) != "" {
			query := strings.ToLower(strings.TrimSpace(filter.Search))
			if !strings.Contains(strings.ToLower(product.Name), query) && !strings.Contains(strings.ToLower(product.Description), query) {
				continue
			}
		}
		result = append(result, product)
	}
	return result, len(result), nil
}

func (r *fakeProductRepo) Update(_ context.Context, p *domain.Product) error {
	r.product = p
	for i, product := range r.products {
		if product.ID == p.ID {
			r.products[i] = p
			return nil
		}
	}
	r.products = append(r.products, p)
	return nil
}

type fakeReservationRepo struct{}

func (r *fakeReservationRepo) Save(_ context.Context, _ *domain.Reservation) error { return nil }
func (r *fakeReservationRepo) GetByOrderID(_ context.Context, _ string) ([]*domain.Reservation, error) {
	return nil, nil
}
func (r *fakeReservationRepo) UpdateStatus(_ context.Context, _ string, _ domain.ReservationStatus) error {
	return nil
}

type fakeAttributeRepo struct {
	attrs         []*domain.Attribute
	listByProduct func(context.Context, string) ([]*domain.Attribute, error)
}

func (r *fakeAttributeRepo) SaveBatch(_ context.Context, attrs []*domain.Attribute) error {
	r.attrs = attrs
	return nil
}

func (r *fakeAttributeRepo) ListByProduct(ctx context.Context, productID string) ([]*domain.Attribute, error) {
	if r.listByProduct != nil {
		return r.listByProduct(ctx, productID)
	}
	return r.attrs, nil
}

func (r *fakeAttributeRepo) DeleteByProduct(_ context.Context, _ string) error { return nil }

type fakeVariantRepo struct {
	variants      []*domain.Variant
	listByProduct func(context.Context, string) ([]*domain.Variant, error)
}

func (r *fakeVariantRepo) Create(_ context.Context, v *domain.Variant) error {
	r.variants = append(r.variants, v)
	return nil
}

func (r *fakeVariantRepo) Update(_ context.Context, v *domain.Variant) error {
	for i, existing := range r.variants {
		if existing.ID == v.ID {
			r.variants[i] = v
			return nil
		}
	}
	r.variants = append(r.variants, v)
	return nil
}

func (r *fakeVariantRepo) Delete(_ context.Context, id string) error {
	filtered := r.variants[:0]
	for _, variant := range r.variants {
		if variant.ID != id {
			filtered = append(filtered, variant)
		}
	}
	r.variants = filtered
	return nil
}

func (r *fakeVariantRepo) GetByID(_ context.Context, id string) (*domain.Variant, error) {
	for _, variant := range r.variants {
		if variant.ID == id {
			return variant, nil
		}
	}
	return nil, domain.ErrVariantNotFound
}

func (r *fakeVariantRepo) ListByProduct(ctx context.Context, productID string) ([]*domain.Variant, error) {
	if r.listByProduct != nil {
		return r.listByProduct(ctx, productID)
	}
	return r.variants, nil
}

func (r *fakeVariantRepo) GetBySKU(_ context.Context, sku string) (*domain.Variant, error) {
	for _, variant := range r.variants {
		if variant.SKU == sku {
			return variant, nil
		}
	}
	return nil, domain.ErrVariantNotFound
}

type fakeGlobalAttributeRepo struct {
	attrs []*domain.GlobalAttribute
}

func (r *fakeGlobalAttributeRepo) Create(_ context.Context, a *domain.GlobalAttribute) error {
	r.attrs = append(r.attrs, a)
	return nil
}

func (r *fakeGlobalAttributeRepo) GetByID(_ context.Context, id string) (*domain.GlobalAttribute, error) {
	for _, attr := range r.attrs {
		if attr.ID == id {
			return attr, nil
		}
	}
	return nil, domain.ErrGlobalAttributeNotFound
}

func (r *fakeGlobalAttributeRepo) List(_ context.Context, filter port.GlobalAttributeFilter) ([]*domain.GlobalAttribute, error) {
	var result []*domain.GlobalAttribute
	for _, attr := range r.attrs {
		if filter.CategoryID != "" && attr.CategoryID == filter.CategoryID {
			result = append(result, attr)
			continue
		}
		if filter.CategoryID == "" && filter.Category != "" && attr.Category == filter.Category {
			result = append(result, attr)
		}
	}
	return result, nil
}

func (r *fakeGlobalAttributeRepo) ListCategories(_ context.Context) ([]string, error) {
	return nil, nil
}
func (r *fakeGlobalAttributeRepo) Update(_ context.Context, _ *domain.GlobalAttribute) error {
	return nil
}
func (r *fakeGlobalAttributeRepo) Delete(_ context.Context, _ string) error { return nil }

type fakeCategoryRepo struct {
	categories    []*domain.Category
	subcategories []*domain.Subcategory
}

func (r *fakeCategoryRepo) CreateCategory(_ context.Context, category *domain.Category) error {
	r.categories = append(r.categories, category)
	return nil
}

func (r *fakeCategoryRepo) GetCategoryByID(_ context.Context, id string) (*domain.Category, error) {
	for _, category := range r.categories {
		if category.ID == id {
			return category, nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}

func (r *fakeCategoryRepo) ListCategories(_ context.Context) ([]*domain.Category, error) {
	return r.categories, nil
}

func (r *fakeCategoryRepo) UpdateCategory(_ context.Context, category *domain.Category) error {
	for i, existing := range r.categories {
		if existing.ID == category.ID {
			r.categories[i] = category
			return nil
		}
	}
	return domain.ErrCategoryNotFound
}

func (r *fakeCategoryRepo) DeleteCategory(_ context.Context, id string) error {
	for i, category := range r.categories {
		if category.ID == id {
			r.categories = append(r.categories[:i], r.categories[i+1:]...)
			return nil
		}
	}
	return domain.ErrCategoryNotFound
}

func (r *fakeCategoryRepo) CreateSubcategory(_ context.Context, subcategory *domain.Subcategory) error {
	r.subcategories = append(r.subcategories, subcategory)
	return nil
}

func (r *fakeCategoryRepo) GetSubcategoryByID(_ context.Context, id string) (*domain.Subcategory, error) {
	for _, subcategory := range r.subcategories {
		if subcategory.ID == id {
			return subcategory, nil
		}
	}
	return nil, domain.ErrSubcategoryNotFound
}

func (r *fakeCategoryRepo) ListSubcategories(_ context.Context, categoryID string) ([]*domain.Subcategory, error) {
	if categoryID == "" {
		return r.subcategories, nil
	}
	var result []*domain.Subcategory
	for _, subcategory := range r.subcategories {
		if subcategory.CategoryID == categoryID {
			result = append(result, subcategory)
		}
	}
	return result, nil
}

func (r *fakeCategoryRepo) UpdateSubcategory(_ context.Context, subcategory *domain.Subcategory) error {
	for i, existing := range r.subcategories {
		if existing.ID == subcategory.ID {
			r.subcategories[i] = subcategory
			return nil
		}
	}
	return domain.ErrSubcategoryNotFound
}

func (r *fakeCategoryRepo) DeleteSubcategory(_ context.Context, id string) error {
	for i, subcategory := range r.subcategories {
		if subcategory.ID == id {
			r.subcategories = append(r.subcategories[:i], r.subcategories[i+1:]...)
			return nil
		}
	}
	return domain.ErrSubcategoryNotFound
}

type fakeImageRepo struct{}

func (r *fakeImageRepo) SetImages(_ context.Context, _ string, _ []string) ([]*domain.ProductImage, error) {
	return nil, nil
}
func (r *fakeImageRepo) GetByProductID(_ context.Context, _ string) ([]*domain.ProductImage, error) {
	return nil, nil
}
func (r *fakeImageRepo) GetByProductIDs(_ context.Context, _ []string) (map[string][]*domain.ProductImage, error) {
	return map[string][]*domain.ProductImage{}, nil
}

type fakePublisher struct{}

func (p *fakePublisher) Publish(_ context.Context, _ string, _ any) error { return nil }

type fakeStoreClient struct {
	storeID string
}

func (c *fakeStoreClient) GetStoreIDByOwner(_ context.Context, _ string) (string, error) {
	return c.storeID, nil
}

func TestUpdate_AllowsAttributeExpansionBeforeVariantMigration(t *testing.T) {
	storeID := "store-1"
	product, err := domain.NewProduct("Codex Mug", "Test", "CODEX-MUG", 1234, "USD", "general", "", 10)
	if err != nil {
		t.Fatalf("new product: %v", err)
	}
	product.CategoryID = "cat-general"
	product.StoreID = &storeID

	colorID := "ga-color"
	sizeID := "ga-size"
	currentColorAttr, _ := domain.NewAttribute(product.ID, &colorID, "Color", []string{"Red"}, 0)
	existingVariant, _ := domain.NewVariant(product.ID, "CODEX-MUG-V1", nil, 5, map[string]string{
		"Color": "Red",
	})

	productRepo := &fakeProductRepo{product: product, products: []*domain.Product{product}}
	attrRepo := &fakeAttributeRepo{attrs: []*domain.Attribute{currentColorAttr}}
	variantRepo := &fakeVariantRepo{variants: []*domain.Variant{existingVariant}}
	globalRepo := &fakeGlobalAttributeRepo{attrs: []*domain.GlobalAttribute{
		{ID: colorID, CategoryID: "cat-general", Category: "general", Name: "Color", Values: []string{"Red", "Grey"}},
		{ID: sizeID, CategoryID: "cat-general", Category: "general", Name: "Size", Values: []string{"M", "L"}},
	}}

	svc := New(
		productRepo,
		&fakeReservationRepo{},
		attrRepo,
		variantRepo,
		globalRepo,
		&fakeCategoryRepo{},
		&fakeImageRepo{},
		nil,
		&fakeStoreClient{storeID: storeID},
		slog.Default(),
	)

	updated, err := svc.Update(context.Background(), product.ID, UpdateRequest{
		Attributes: []AttributeInput{
			{Name: "Color", Values: []string{"Red", "Grey"}},
			{Name: "Size", Values: []string{"M", "L"}},
		},
		CallerID:   "seller-1",
		CallerRole: "seller",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if len(updated.Attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(updated.Attributes))
	}
	if got := updated.Attributes[0].Values; len(got) != 2 || got[1] != "Grey" {
		t.Fatalf("expected expanded values to be saved, got %+v", got)
	}
}

func TestList_ComputesServerSideFacetsAndFiltering(t *testing.T) {
	storeID := "store-1"
	productOne, _ := domain.NewProduct("Codex Mug", "Ceramic mug", "MUG-1", 1000, "USD", "general", "", 10)
	productOne.StoreID = &storeID
	productOne.CategoryID = "cat-general"
	productTwo, _ := domain.NewProduct("Codex Tee", "Cotton tee", "TEE-1", 2000, "USD", "apparel", "", 10)
	productTwo.StoreID = &storeID
	productTwo.CategoryID = "cat-apparel"

	colorID := "ga-color"
	sizeID := "ga-size"
	productOneColor, _ := domain.NewAttribute(productOne.ID, &colorID, "Color", []string{"Red", "Blue"}, 0)
	productOneSize, _ := domain.NewAttribute(productOne.ID, &sizeID, "Size", []string{"M"}, 1)
	productTwoColor, _ := domain.NewAttribute(productTwo.ID, &colorID, "Color", []string{"Red", "Black"}, 0)
	productTwoSize, _ := domain.NewAttribute(productTwo.ID, &sizeID, "Size", []string{"L"}, 1)

	productOneRed, _ := domain.NewVariant(productOne.ID, "MUG-1-RED-M", nil, 5, map[string]string{"Color": "Red", "Size": "M"})
	productOneBlue, _ := domain.NewVariant(productOne.ID, "MUG-1-BLUE-M", nil, 5, map[string]string{"Color": "Blue", "Size": "M"})
	productTwoRed, _ := domain.NewVariant(productTwo.ID, "TEE-1-RED-L", nil, 5, map[string]string{"Color": "Red", "Size": "L"})

	productRepo := &fakeProductRepo{products: []*domain.Product{productOne, productTwo}}
	attrRepo := &fakeAttributeRepo{
		listByProduct: func(_ context.Context, productID string) ([]*domain.Attribute, error) {
			switch productID {
			case productOne.ID:
				return []*domain.Attribute{productOneColor, productOneSize}, nil
			case productTwo.ID:
				return []*domain.Attribute{productTwoColor, productTwoSize}, nil
			default:
				return nil, nil
			}
		},
	}
	variantRepo := &fakeVariantRepo{
		listByProduct: func(_ context.Context, productID string) ([]*domain.Variant, error) {
			switch productID {
			case productOne.ID:
				return []*domain.Variant{productOneRed, productOneBlue}, nil
			case productTwo.ID:
				return []*domain.Variant{productTwoRed}, nil
			default:
				return nil, nil
			}
		},
	}

	svc := New(
		productRepo,
		&fakeReservationRepo{},
		attrRepo,
		variantRepo,
		&fakeGlobalAttributeRepo{},
		&fakeCategoryRepo{categories: []*domain.Category{
			{ID: "cat-general", Name: "general"},
			{ID: "cat-apparel", Name: "apparel"},
		}},
		&fakeImageRepo{},
		nil,
		&fakeStoreClient{storeID: storeID},
		slog.Default(),
	)

	result, err := svc.List(context.Background(), port.ListFilter{
		Status:          "active",
		CategoryID:      "cat-general",
		AttributeValues: map[string][]string{"Color": []string{"Red"}},
		Limit:           20,
		Offset:          0,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("expected total 1, got %d", result.Total)
	}
	if len(result.Products) != 1 || result.Products[0].ID != productOne.ID {
		t.Fatalf("expected only general red mug, got %+v", result.Products)
	}
	if len(result.Categories) != 2 {
		t.Fatalf("expected 2 category facets, got %+v", result.Categories)
	}
	if result.Categories[0].ID == "" || result.Categories[0].Name == "" {
		t.Fatalf("expected category id and name, got %+v", result.Categories[0])
	}
	if len(result.Facets) != 2 {
		t.Fatalf("expected 2 attribute facets, got %+v", result.Facets)
	}
	if result.Facets[0].Name != "Color" || len(result.Facets[0].Values) != 2 {
		t.Fatalf("expected color facet counts, got %+v", result.Facets[0])
	}
}
