import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  ShoppingBag,
  Check,
  ChevronLeft,
  ChevronRight,
  Package,
  Minus,
  Plus,
} from 'lucide-react';
import { toast } from 'sonner';
import { productsApi, cartApi } from '../lib/api';
import { Button } from '../components/ui/Button';
import { useUser } from '../lib/store';
import { cn, formatMoney } from '../lib/utils';
import type { Product, ProductVariant } from '../lib/types';

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
  const variants: ProductVariant[] = product?.variants ?? [];
  const allAttrsSelected =
    attributes.length === 0 ||
    attributes.every((attribute) => selectedValues[attribute.name]);
  const matchingVariant: ProductVariant | undefined =
    variants.length > 0 && allAttrsSelected
      ? variants.find((variant) =>
          Object.entries(selectedValues).every(
            ([attr, value]) => variant.attribute_values[attr] === value,
          ),
        )
      : undefined;
  const displayStock =
    variants.length > 0 ? matchingVariant?.stock ?? 0 : product?.stock ?? 0;
  const safeQuantity = displayStock > 0 ? Math.min(quantity, displayStock) : 1;

  const addToCartMutation = useMutation({
    mutationFn: ({
      productId,
      variantId,
      qty,
    }: {
      productId: string;
      variantId?: string;
      qty: number;
    }) => cartApi.addItem({ product_id: productId, variant_id: variantId, quantity: qty }),
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
        <button
          onClick={() => navigate('/shop')}
          className="mt-4 cursor-pointer text-sm text-indigo-400 hover:text-indigo-300"
        >
          Back to shop
        </button>
      </div>
    );
  }

  const images =
    product.images && product.images.length > 0
      ? product.images.map((image) => image.url)
      : product.image_url
        ? [product.image_url]
        : [];

  const displayPrice =
    matchingVariant?.price != null ? matchingVariant.price : product.price;

  const inStock =
    allAttrsSelected &&
    displayStock > 0 &&
    (matchingVariant
      ? matchingVariant.status === 'active'
      : product.status === 'active');

  const handleAttrSelect = (attrName: string, value: string) => {
    setSelectedValues((prev) => ({ ...prev, [attrName]: value }));
  };

  const handleAddToCart = () => {
    addToCartMutation.mutate({
      productId: product.id,
      variantId: matchingVariant?.id,
      qty: safeQuantity,
    });
  };

  return (
    <div className="max-w-5xl animate-fade-in">
      <button
        onClick={() => navigate('/shop')}
        className="mb-6 flex cursor-pointer items-center gap-1.5 text-sm text-zinc-500 transition-colors hover:text-zinc-300"
      >
        <ArrowLeft size={15} /> Back to Shop
      </button>

      <div className="grid grid-cols-1 gap-10 md:grid-cols-2">
        <div className="space-y-3">
          {images.length > 0 ? (
            <>
              <div className="relative aspect-square overflow-hidden rounded-2xl bg-zinc-800 ring-1 ring-white/[0.06]">
                <img
                  src={images[activeImg]}
                  alt={product.name}
                  className="h-full w-full object-cover"
                />
                {images.length > 1 && (
                  <>
                    <button
                      onClick={() =>
                        setActiveImg((index) => (index - 1 + images.length) % images.length)
                      }
                      className="absolute left-2 top-1/2 flex h-8 w-8 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full bg-black/50 text-white transition-colors hover:bg-black/70"
                    >
                      <ChevronLeft size={16} />
                    </button>
                    <button
                      onClick={() => setActiveImg((index) => (index + 1) % images.length)}
                      className="absolute right-2 top-1/2 flex h-8 w-8 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full bg-black/50 text-white transition-colors hover:bg-black/70"
                    >
                      <ChevronRight size={16} />
                    </button>
                    <div className="absolute bottom-2 left-1/2 flex -translate-x-1/2 gap-1">
                      {images.map((_, index) => (
                        <div
                          key={index}
                          className={cn(
                            'h-1.5 w-1.5 rounded-full transition-colors',
                            index === activeImg ? 'bg-white' : 'bg-white/30',
                          )}
                        />
                      ))}
                    </div>
                  </>
                )}
              </div>

              {images.length > 1 && (
                <div className="flex gap-2 overflow-x-auto pb-1">
                  {images.map((url, index) => (
                    <button
                      key={index}
                      onClick={() => setActiveImg(index)}
                      className={cn(
                        'h-16 w-16 flex-shrink-0 cursor-pointer overflow-hidden rounded-lg ring-2 transition-all',
                        index === activeImg
                          ? 'ring-indigo-500'
                          : 'ring-white/[0.06] hover:ring-white/20',
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
            {product.category && (
              <p className="mb-1.5 text-xs font-semibold uppercase tracking-wider text-indigo-400">
                {product.category}
              </p>
            )}
            <h1 className="text-2xl font-bold leading-tight tracking-tight text-zinc-100">
              {product.name}
            </h1>
            {product.description && (
              <p className="mt-3 text-sm leading-relaxed text-zinc-400">
                {product.description}
              </p>
            )}
          </div>

          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-bold tabular-nums text-zinc-100">
              {formatMoney(displayPrice, product.currency)}
            </span>
            {matchingVariant?.price != null && matchingVariant.price !== product.price && (
              <span className="text-sm text-zinc-500 line-through">
                {formatMoney(product.price, product.currency)}
              </span>
            )}
          </div>

          <div className="space-y-4 rounded-2xl bg-white/[0.03] p-4 ring-1 ring-white/[0.06]">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wider text-zinc-500">
                  Quantity
                </p>
                <p className="mt-1 text-sm text-zinc-400">
                  Choose how many units to add to your cart.
                </p>
              </div>
              <div className="inline-flex items-center rounded-xl bg-white/5 ring-1 ring-white/[0.08]">
                <button
                  type="button"
                  onClick={() => setQuantity(Math.max(1, safeQuantity - 1))}
                  className="cursor-pointer p-2 text-zinc-400 transition-colors hover:text-zinc-200"
                >
                  <Minus size={15} />
                </button>
                <span className="w-10 text-center text-sm font-semibold tabular-nums text-zinc-100">
                  {safeQuantity}
                </span>
                <button
                  type="button"
                  onClick={() =>
                    setQuantity(Math.min(Math.max(1, displayStock), safeQuantity + 1))
                  }
                  disabled={!inStock}
                  className="cursor-pointer p-2 text-zinc-400 transition-colors hover:text-zinc-200 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <Plus size={15} />
                </button>
              </div>
            </div>

            {attributes.length > 0 && (
              <div className="rounded-xl bg-zinc-950/40 px-3.5 py-3 ring-1 ring-white/[0.04]">
                <p className="mb-1.5 text-xs font-semibold uppercase tracking-wider text-zinc-500">
                  Selected Options
                </p>
                <p className="text-sm text-zinc-300">
                  {allAttrsSelected
                    ? attributes
                        .map((attribute) => `${attribute.name}: ${selectedValues[attribute.name]}`)
                        .join(' • ')
                    : 'Choose every option below before adding this item to your cart.'}
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
                    {selectedValues[attribute.name] && (
                      <span className="ml-1.5 font-normal text-zinc-500">
                        - {selectedValues[attribute.name]}
                      </span>
                    )}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {attribute.values.map((value) => {
                      const hasStock =
                        variants.length === 0 ||
                        variants.some(
                          (variant) =>
                            variant.attribute_values[attribute.name] === value &&
                            variant.stock > 0 &&
                            variant.status === 'active',
                        );

                      return (
                        <button
                          key={value}
                          onClick={() => handleAttrSelect(attribute.name, value)}
                          disabled={!hasStock}
                          className={cn(
                            'cursor-pointer rounded-lg px-3.5 py-2 text-sm font-medium transition-all',
                            selectedValues[attribute.name] === value
                              ? 'bg-indigo-600 text-white ring-2 ring-indigo-500 ring-offset-1 ring-offset-zinc-900'
                              : hasStock
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

          {variants.length > 0 && matchingVariant && (
            <div className="flex items-center gap-2 text-sm">
              <div
                className={cn(
                  'h-2 w-2 rounded-full',
                  inStock ? 'bg-emerald-400' : 'bg-red-400',
                )}
              />
              <span className={inStock ? 'text-emerald-400' : 'text-red-400'}>
                {inStock ? `${displayStock} in stock` : 'Out of stock'}
              </span>
              {matchingVariant.sku && (
                <span className="text-zinc-600">· SKU: {matchingVariant.sku}</span>
              )}
            </div>
          )}

          {variants.length === 0 && (
            <div className="flex items-center gap-2 text-sm">
              <div
                className={cn(
                  'h-2 w-2 rounded-full',
                  inStock ? 'bg-emerald-400' : 'bg-red-400',
                )}
              />
              <span className={inStock ? 'text-emerald-400' : 'text-red-400'}>
                {inStock ? `${displayStock} in stock` : 'Out of stock'}
              </span>
            </div>
          )}

          {attributes.length > 0 && !allAttrsSelected && (
            <p className="text-xs text-zinc-500">
              Please select{' '}
              {attributes
                .filter((attribute) => !selectedValues[attribute.name])
                .map((attribute) => attribute.name)
                .join(', ')}{' '}
              to continue
            </p>
          )}

          <Button
            size="lg"
            className="w-full"
            disabled={!inStock || (attributes.length > 0 && !allAttrsSelected)}
            loading={addToCartMutation.isPending}
            onClick={handleAddToCart}
          >
            {added ? (
              <>
                <Check size={16} /> Added to Cart
              </>
            ) : !inStock ? (
              'Out of Stock'
            ) : attributes.length > 0 && !allAttrsSelected ? (
              'Select Options'
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
