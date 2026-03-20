package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// FindVariant returns the variant with the given ID, or (zero, false) if not found.
func FindVariant(product ProductInfo, variantID *string) (VariantInfo, bool) {
	if variantID == nil {
		return VariantInfo{}, false
	}
	for _, v := range product.Variants {
		if v.ID == *variantID {
			return v, true
		}
	}
	return VariantInfo{}, false
}

// EffectivePrice returns the variant's price if set, otherwise the product base price.
func EffectivePrice(product ProductInfo, variant VariantInfo) int64 {
	if variant.Price != nil {
		return *variant.Price
	}
	return product.Price
}

// VariantMatchesAttributes checks that a variant's attribute values exactly match
// the product's declared attribute definitions (names and allowed values).
func VariantMatchesAttributes(product ProductInfo, variant VariantInfo) bool {
	if len(product.Attributes) == 0 {
		return len(variant.AttributeValues) == 0
	}
	if len(variant.AttributeValues) != len(product.Attributes) {
		return false
	}
	for _, attr := range product.Attributes {
		val, ok := variant.AttributeValues[attr.Name]
		if !ok {
			return false
		}
		if !containsValue(attr.Values, val) {
			return false
		}
	}
	return true
}

// FormatVariantLabel produces a stable, human-readable label from attribute values,
// e.g. "Color: Red / Size: M".
func FormatVariantLabel(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, values[k]))
	}
	return strings.Join(parts, " / ")
}

func containsValue(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
