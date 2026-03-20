import type { Product, ProductAttribute, ProductVariant } from './types';

export function isVariantCompatible(attributes: ProductAttribute[], variant: ProductVariant) {
  if (attributes.length === 0) {
    return Object.keys(variant.attribute_values ?? {}).length === 0;
  }

  const values = variant.attribute_values ?? {};
  if (Object.keys(values).length !== attributes.length) {
    return false;
  }

  return attributes.every((attribute) => {
    const selectedValue = values[attribute.name];
    return Boolean(selectedValue) && attribute.values.includes(selectedValue);
  });
}

export function getPurchasableVariants(product: Product) {
  const attributes = product.attributes ?? [];
  return (product.variants ?? []).filter(
    (variant) => variant.status === 'active' && variant.stock > 0 && isVariantCompatible(attributes, variant),
  );
}
