package port

import "context"

type ProductClient interface {
	GetProducts(ctx context.Context, ids []string) ([]ProductInfo, error)
}

type ProductInfo struct {
	ID       string
	Name     string
	Price    int64 // authoritative unit price in cents
	Currency string
	Stock    int
	Status   string
	Variants []VariantInfo
}

type VariantInfo struct {
	ID              string
	SKU             string
	Price           *int64
	Stock           int
	Status          string
	AttributeValues map[string]string
}
