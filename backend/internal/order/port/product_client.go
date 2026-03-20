package port

import (
	"context"

	"payment-platform/pkg/catalog"
)

type ProductClient interface {
	GetProducts(ctx context.Context, ids []string) ([]ProductInfo, error)
}

// Type aliases so existing code using port.ProductInfo compiles unchanged.
type ProductInfo = catalog.ProductInfo
type AttributeInfo = catalog.AttributeInfo
type VariantInfo = catalog.VariantInfo
