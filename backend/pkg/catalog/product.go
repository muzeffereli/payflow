package catalog

// ProductInfo is the authoritative product snapshot used by both the cart
// and order services when they call the product service.
type ProductInfo struct {
	ID         string
	Name       string
	Price      int64   // authoritative unit price in cents
	Currency   string
	Stock      int
	Status     string  // "active" | "inactive"
	StoreID    *string // nil = platform product
	Attributes []AttributeInfo
	Variants   []VariantInfo
}

type AttributeInfo struct {
	Name   string
	Values []string
}

type VariantInfo struct {
	ID              string
	SKU             string
	Price           *int64
	Stock           int
	Status          string
	AttributeValues map[string]string
}
