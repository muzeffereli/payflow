package domain

import (
	"errors"
	"time"
)

var ErrCartEmpty = errors.New("cart is empty")

var ErrItemNotFound = errors.New("item not found in cart")

type Cart struct {
	UserID    string     `json:"user_id"`
	Items     []CartItem `json:"items"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CartItem struct {
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity"`
}

func New(userID string) *Cart {
	return &Cart{
		UserID:    userID,
		Items:     []CartItem{},
		UpdatedAt: time.Now().UTC(),
	}
}

func (c *Cart) AddItem(productID string, variantID *string, qty int) {
	for i, item := range c.Items {
		if sameCartItem(item.ProductID, item.VariantID, productID, variantID) {
			c.Items[i].Quantity += qty
			c.UpdatedAt = time.Now().UTC()
			return
		}
	}
	c.Items = append(c.Items, CartItem{ProductID: productID, VariantID: variantID, Quantity: qty})
	c.UpdatedAt = time.Now().UTC()
}

func (c *Cart) RemoveItem(productID string, variantID *string) error {
	for i, item := range c.Items {
		if sameCartItem(item.ProductID, item.VariantID, productID, variantID) {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			c.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return ErrItemNotFound
}

func (c *Cart) SetQuantity(productID string, variantID *string, qty int) error {
	if qty <= 0 {
		return c.RemoveItem(productID, variantID)
	}
	for i, item := range c.Items {
		if sameCartItem(item.ProductID, item.VariantID, productID, variantID) {
			c.Items[i].Quantity = qty
			c.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return ErrItemNotFound
}

func (c *Cart) IsEmpty() bool {
	return len(c.Items) == 0
}

func sameCartItem(productID string, variantID *string, otherProductID string, otherVariantID *string) bool {
	if productID != otherProductID {
		return false
	}
	return sameOptionalString(variantID, otherVariantID)
}

func sameOptionalString(a, b *string) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}
