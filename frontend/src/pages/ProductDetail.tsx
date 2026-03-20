import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Check, ChevronLeft, ChevronRight, Minus, Package, Plus, ShoppingBag } from 'lucide-react';
import { useNavigate, useParams } from 'react-router-dom';
import { toast } from 'sonner';
import { cartApi, productsApi } from '../lib/api';
import { getPurchasableVariants } from '../lib/productVariants';
import { Button } from '../components/ui/Button';
import { useUser } from '../lib/store';
import { cn, formatMoney } from '../lib/utils';
import type { Product, ProductAttribute, ProductVariant } from '../lib/types';

export function ProductDetail() {
  const { id } = useParams<{ id: string }>();

  if (!id) {
    return null;
  }

  return <ProductDetailContent key={id} id={id} />;
}

function ProductDetailContent({ id }: { id: string }) {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const user = useUser();

  const [activeImg, setActiveImg] = useState(0);
  const [selectedValues, setSelectedValues] = useState<Record<string, string>>({});
  const [added, setAdded] = useState(false);
  const [quantity, setQuantity] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ['product', id],
    queryFn: () => productsApi.get(id),
  });

  const product: Product | undefined = data?.data?.product ?? data?.data;
  const attributes = product?.attributes ?? [];
  const variants = product?.variants ?? [];
  const activeVariants = useMemo(() => (product ? getPurchasableVariants(product) : []), [product]);
  const allAttrsSelected = attributes.length > 0 && attributes.every((attribute) => selectedValues[attribute.name]);
  const matchingVariant = useMemo(
    () =>
      allAttrsSelected
        ? activeVariants.find((variant) =>
            attributes.every((attribute) => variant.attribute_values[attribute.name] === selectedValues[attribute.name]),
          )
        : undefined,
    [activeVariants, allAttrsSelected, attributes, selectedValues],
  );

  const addToCartMutation = useMutation({
    mutationFn: ({ productId, variantId, qty }: { productId: string; variantId?: string; qty: number }) =>
      cartApi.addItem({ product_id: productId, variant_id: variantId, quantity: qty }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['cart', user?.id] });
      toast.success('Added to cart');
      setAdded(true);
      setTimeout(() => setAdded(false), 1500);
    },
    onError: () => toast.error('Failed to add to cart'),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-500 border-t-transparent" />
      </div>
    );
  }

  if (!product) {
    return (
      <div className="flex flex-col items-center justify-center py-24">
        <Package size={40} className="mb-4 text-zinc-600" />
        <p className="font-medium text-zinc-400">Product not found</p>
        <button type="button" onClick={() => navigate('/shop')} className="mt-4 text-sm text-indigo-400 hover:text-indigo-300">
          Back to shop
        </button>
      </div>
    );
  }

  const currentProduct = product;

  const images =
    currentProduct.images && currentProduct.images.length > 0
      ? currentProduct.images.map((image) => image.url)
      : currentProduct.image_url
        ? [currentProduct.image_url]
        : [];

  const displayPrice = matchingVariant?.price ?? getLowestVariantPrice(activeVariants, currentProduct.price) ?? currentProduct.price;
  const variantRequired = attributes.length > 0 || variants.length > 0;
  const inStock = variantRequired
    ? Boolean(matchingVariant)
    : currentProduct.status === 'active' && currentProduct.stock > 0;
  const displayStock = matchingVariant?.stock ?? (variantRequired ? 0 : currentProduct.stock);
  const safeQuantity = displayStock > 0 ? Math.min(quantity, displayStock) : 1;
  const canPurchase = variantRequired ? Boolean(matchingVariant) : inStock;

  function handleAttrSelect(attributeName: string, value: string) {
    const next = pruneSelections(attributes, activeVariants, {
      ...selectedValues,
      [attributeName]: value,
    });
    setSelectedValues(next);
    setQuantity(1);
  }

  function handleAddToCart() {
    addToCartMutation.mutate({
      productId: currentProduct.id,
      variantId: matchingVariant?.id,
      qty: safeQuantity,
    });
  }

  return (
    <div className="max-w-5xl animate-fade-in">
      <button
        type="button"
        onClick={() => navigate('/shop')}
        className="mb-6 flex items-center gap-1.5 text-sm text-zinc-500 transition-colors hover:text-zinc-300"
      >
        <ArrowLeft size={15} /> Back to Shop
      </button>

      <div className="grid grid-cols-1 gap-10 md:grid-cols-2">
        <div className="space-y-3">
          {images.length > 0 ? (
            <>
              <div className="relative aspect-square overflow-hidden rounded-2xl bg-zinc-800 ring-1 ring-white/[0.06]">
                <img src={images[activeImg]} alt={product.name} className="h-full w-full object-cover" />
                {images.length > 1 && (
                  <>
                    <button
                      type="button"
                      onClick={() => setActiveImg((index) => (index - 1 + images.length) % images.length)}
                      className="absolute left-2 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-black/50 text-white transition-colors hover:bg-black/70"
                    >
                      <ChevronLeft size={16} />
                    </button>
                    <button
                      type="button"
                      onClick={() => setActiveImg((index) => (index + 1) % images.length)}
                      className="absolute right-2 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-black/50 text-white transition-colors hover:bg-black/70"
                    >
                      <ChevronRight size={16} />
                    </button>
                  </>
                )}
              </div>

              {images.length > 1 && (
                <div className="flex gap-2 overflow-x-auto pb-1">
                  {images.map((url, index) => (
                    <button
                      key={index}
                      type="button"
                      onClick={() => setActiveImg(index)}
                      className={cn(
                        'h-16 w-16 flex-shrink-0 overflow-hidden rounded-lg ring-2 transition-all',
                        index === activeImg ? 'ring-indigo-500' : 'ring-white/[0.06] hover:ring-white/20',
                      )}
                    >
                      <img src={url} alt="" className="h-full w-full object-cover" />
                    </button>
                  ))}
                </div>
              )}
            </>
          ) : (
            <div className="flex aspect-square items-center justify-center rounded-2xl bg-zinc-800 ring-1 ring-white/[0.06]">
              <Package size={48} className="text-zinc-600" />
            </div>
          )}
        </div>

        <div className="space-y-6">
          <div>
            {currentProduct.category && (
              <p className="mb-1.5 text-xs font-semibold uppercase tracking-wider text-indigo-400">
                {currentProduct.category}
                {currentProduct.subcategory ? ` / ${currentProduct.subcategory}` : ''}
              </p>
            )}
            <h1 className="text-2xl font-bold leading-tight tracking-tight text-zinc-100">{currentProduct.name}</h1>
            {currentProduct.description && <p className="mt-3 text-sm leading-relaxed text-zinc-400">{currentProduct.description}</p>}
          </div>

          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-bold tabular-nums text-zinc-100">{formatMoney(displayPrice, currentProduct.currency)}</span>
            {matchingVariant?.price != null && matchingVariant.price !== currentProduct.price && (
              <span className="text-sm text-zinc-500 line-through">{formatMoney(currentProduct.price, currentProduct.currency)}</span>
            )}
          </div>

          <div className="space-y-4 rounded-2xl bg-white/[0.03] p-4 ring-1 ring-white/[0.06]">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wider text-zinc-500">Quantity</p>
                <p className="mt-1 text-sm text-zinc-400">Choose how many units to add to your cart.</p>
              </div>
              <div className="inline-flex items-center rounded-xl bg-white/5 ring-1 ring-white/[0.08]">
                <button
                  type="button"
                  onClick={() => setQuantity(Math.max(1, safeQuantity - 1))}
                  className="p-2 text-zinc-400 transition-colors hover:text-zinc-200"
                >
                  <Minus size={15} />
                </button>
                <span className="w-10 text-center text-sm font-semibold tabular-nums text-zinc-100">{safeQuantity}</span>
                <button
                  type="button"
                  onClick={() => setQuantity(Math.min(Math.max(1, displayStock), safeQuantity + 1))}
                  disabled={!canPurchase}
                  className="p-2 text-zinc-400 transition-colors hover:text-zinc-200 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <Plus size={15} />
                </button>
              </div>
            </div>

            {attributes.length > 0 && (
              <div className="rounded-xl bg-zinc-950/40 px-3.5 py-3 ring-1 ring-white/[0.04]">
                <p className="mb-1.5 text-xs font-semibold uppercase tracking-wider text-zinc-500">Selected Options</p>
                <p className="text-sm text-zinc-300">
                  {matchingVariant
                    ? attributes.map((attribute) => `${attribute.name}: ${selectedValues[attribute.name]}`).join(' · ')
                    : 'Choose a valid exact combination before purchasing.'}
                </p>
              </div>
            )}
          </div>

          {attributes.length > 0 && (
            <div className="space-y-4">
              {attributes.map((attribute) => (
                <div key={attribute.id}>
                  <p className="mb-2 text-sm font-medium text-zinc-300">
                    {attribute.name}
                    {selectedValues[attribute.name] ? (
                      <span className="ml-1.5 font-normal text-zinc-500">- {selectedValues[attribute.name]}</span>
                    ) : null}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {attribute.values.map((value) => {
                      const available = isValueAvailable(attribute.name, value, attributes, activeVariants, selectedValues);
                      return (
                        <button
                          key={value}
                          type="button"
                          onClick={() => handleAttrSelect(attribute.name, value)}
                          disabled={!available}
                          className={cn(
                            'rounded-lg px-3.5 py-2 text-sm font-medium transition-all',
                            selectedValues[attribute.name] === value
                              ? 'bg-indigo-600 text-white ring-2 ring-indigo-500 ring-offset-1 ring-offset-zinc-900'
                              : available
                                ? 'bg-white/[0.06] text-zinc-300 ring-1 ring-white/[0.08] hover:bg-white/[0.10]'
                                : 'cursor-not-allowed bg-white/[0.02] text-zinc-600 line-through ring-1 ring-white/[0.04]',
                          )}
                        >
                          {value}
                        </button>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          )}

          {variantRequired ? (
            <div className="flex items-center gap-2 text-sm">
              <div className={cn('h-2 w-2 rounded-full', matchingVariant ? 'bg-emerald-400' : 'bg-red-400')} />
              <span className={matchingVariant ? 'text-emerald-400' : 'text-red-400'}>
                {matchingVariant ? `${matchingVariant.stock} in stock` : 'Select a valid variant combination'}
              </span>
              {matchingVariant?.sku ? <span className="text-zinc-600">· SKU: {matchingVariant.sku}</span> : null}
            </div>
          ) : (
            <div className="flex items-center gap-2 text-sm">
              <div className={cn('h-2 w-2 rounded-full', inStock ? 'bg-emerald-400' : 'bg-red-400')} />
              <span className={inStock ? 'text-emerald-400' : 'text-red-400'}>
                {inStock ? `${displayStock} in stock` : 'Out of stock'}
              </span>
            </div>
          )}

          {attributes.length > 0 && !matchingVariant && (
            <p className="text-xs text-zinc-500">
              Buyers can only continue after selecting an existing exact combination of {attributes.map((attribute) => attribute.name).join(', ')}.
            </p>
          )}

          <Button
            size="lg"
            className="w-full"
            disabled={!canPurchase}
            loading={addToCartMutation.isPending}
            onClick={handleAddToCart}
          >
            {added ? (
              <>
                <Check size={16} /> Added to Cart
              </>
            ) : !canPurchase ? (
              'Select Valid Variant'
            ) : (
              <>
                <ShoppingBag size={16} /> Add {safeQuantity} to Cart
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}

function getLowestVariantPrice(variants: ProductVariant[], basePrice: number) {
  if (variants.length === 0) {
    return null;
  }
  return variants.reduce((lowest, variant) => Math.min(lowest, variant.price ?? basePrice), Number.MAX_SAFE_INTEGER);
}

function isValueAvailable(
  attributeName: string,
  value: string,
  attributes: ProductAttribute[],
  variants: ProductVariant[],
  selectedValues: Record<string, string>,
) {
  if (variants.length === 0) {
    return true;
  }
  return variants.some((variant) =>
    attributes.every((attribute) => {
      if (attribute.name === attributeName) {
        return variant.attribute_values[attribute.name] === value;
      }
      const selected = selectedValues[attribute.name];
      return !selected || variant.attribute_values[attribute.name] === selected;
    }),
  );
}

function pruneSelections(
  attributes: ProductAttribute[],
  variants: ProductVariant[],
  nextSelections: Record<string, string>,
) {
  if (variants.length === 0) {
    return nextSelections;
  }

  const pruned = { ...nextSelections };
  let changed = true;
  while (changed) {
    changed = false;
    for (const attribute of attributes) {
      const selected = pruned[attribute.name];
      if (!selected) {
        continue;
      }
      const stillValid = variants.some((variant) =>
        attributes.every((candidate) => {
          const targetValue = candidate.name === attribute.name ? selected : pruned[candidate.name];
          return !targetValue || variant.attribute_values[candidate.name] === targetValue;
        }),
      );
      if (!stillValid) {
        delete pruned[attribute.name];
        changed = true;
      }
    }
  }

  return pruned;
}
