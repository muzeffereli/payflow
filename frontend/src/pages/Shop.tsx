import { useCallback, useDeferredValue, useMemo, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Eye, Loader2, Package, PackageX, Search, ShoppingBag, SlidersHorizontal, X } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';
import { cartApi, productsApi } from '../lib/api';
import { getPurchasableVariants } from '../lib/productVariants';
import { Button } from '../components/ui/Button';
import { Card, CardBody } from '../components/ui/Card';
import { Input } from '../components/ui/Input';
import { useUser } from '../lib/store';
import { cn, formatMoney } from '../lib/utils';
import type { AttributeFacet, Product, ProductListResponse } from '../lib/types';

export function Shop() {
  const [search, setSearch] = useState('');
  const [activeCategory, setActiveCategory] = useState('all');
  const [selectedFilters, setSelectedFilters] = useState<Record<string, string[]>>({});
  const [addedIds, setAddedIds] = useState<Set<string>>(new Set());
  const timers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const user = useUser();
  const deferredSearch = useDeferredValue(search.trim());

  const { data, isLoading } = useQuery({
    queryKey: ['shop-products', deferredSearch, activeCategory, JSON.stringify(selectedFilters)],
    queryFn: () =>
      productsApi.list({
        limit: 100,
        offset: 0,
        status: 'active',
        search: deferredSearch || undefined,
        category_id: activeCategory === 'all' ? undefined : activeCategory,
        attribute_filters: selectedFilters,
      }),
  });

  const response: ProductListResponse | undefined = data?.data;
  const products: Product[] = response?.products ?? data?.data?.data ?? [];

  const categories = useMemo(
    () => [{ id: 'all', name: 'All' }, ...(response?.categories ?? [])],
    [response?.categories],
  );

  const filterDefinitions = useMemo(
    () => response?.facets ?? [],
    [response?.facets],
  );
  const filteredProducts = products;

  const hasActiveFilters = Object.values(selectedFilters).some((values) => values.length > 0);

  const addToCartMutation = useMutation({
    mutationFn: (productId: string) => cartApi.addItem({ product_id: productId, quantity: 1 }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cart', user?.id] });
      toast.success('Added to cart');
    },
    onError: () => {
      toast.error('Failed to add to cart');
    },
  });

  const handleAddToCart = useCallback(
    (product: Product) => {
      addToCartMutation.mutate(product.id);

      setAddedIds((current) => {
        const next = new Set(current);
        next.add(product.id);
        return next;
      });

      const existing = timers.current.get(product.id);
      if (existing) {
        clearTimeout(existing);
      }

      const timer = setTimeout(() => {
        setAddedIds((current) => {
          const next = new Set(current);
          next.delete(product.id);
          return next;
        });
        timers.current.delete(product.id);
      }, 1500);

      timers.current.set(product.id, timer);
    },
    [addToCartMutation],
  );

  function toggleFilter(attributeName: string, value: string) {
    setSelectedFilters((current) => {
      const existing = current[attributeName] ?? [];
      const nextValues = existing.includes(value)
        ? existing.filter((item) => item !== value)
        : [...existing, value];
      if (nextValues.length === 0) {
        const { [attributeName]: _, ...rest } = current;
        return rest;
      }
      return { ...current, [attributeName]: nextValues };
    });
  }

  function clearFilters() {
    setSelectedFilters({});
  }

  return (
    <div className="animate-fade-in">
      <div className="mb-8">
        <h1 className="text-2xl font-bold tracking-tight text-zinc-100">Shop</h1>
        <p className="mt-1 text-sm text-zinc-500">
          {response?.total ?? filteredProducts.length} {(response?.total ?? filteredProducts.length) === 1 ? 'product' : 'products'}
        </p>
      </div>

      <div className="mb-6 max-w-md">
        <Input
          placeholder="Search products..."
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          icon={<Search size={15} />}
        />
      </div>

      {categories.length > 1 && (
        <div className="mb-6 flex flex-wrap gap-2">
          {categories.map((category) => (
            <button
              key={category.id}
              type="button"
              onClick={() => {
                setActiveCategory(category.id);
                setSelectedFilters({});
              }}
              className={cn(
                'cursor-pointer rounded-full px-3.5 py-1.5 text-sm font-medium transition-colors',
                activeCategory === category.id
                  ? 'bg-indigo-600 text-white'
                  : 'bg-white/5 text-zinc-400 hover:text-zinc-200',
              )}
            >
              {category.name}
            </button>
          ))}
        </div>
      )}

      {filterDefinitions.length > 0 && (
        <div className="mb-8 rounded-2xl bg-white/[0.03] p-4 ring-1 ring-white/[0.06]">
          <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-2">
              <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-indigo-500/10 ring-1 ring-indigo-500/20">
                <SlidersHorizontal size={16} className="text-indigo-300" />
              </div>
              <div>
                <p className="font-medium text-zinc-100">Variant Filters</p>
                <p className="text-xs text-zinc-500">Default and seller-added values stay filterable here.</p>
              </div>
            </div>

            {hasActiveFilters && (
              <Button variant="ghost" size="sm" onClick={clearFilters}>
                <X size={13} /> Clear Filters
              </Button>
            )}
          </div>

          <div className="space-y-4">
          {filterDefinitions.map((filter: AttributeFacet) => (
              <div key={filter.name}>
                <p className="mb-2 text-sm font-medium text-zinc-300">{filter.name}</p>
                <div className="flex flex-wrap gap-2">
                  {filter.values.map((value) => {
                    const selected = (selectedFilters[filter.name] ?? []).includes(value.value);
                    return (
                      <button
                        key={value.value}
                        type="button"
                        onClick={() => toggleFilter(filter.name, value.value)}
                        className={cn(
                          'rounded-lg px-3 py-1.5 text-xs transition-colors',
                          selected
                            ? 'bg-indigo-500/20 text-indigo-300 ring-1 ring-indigo-500/30'
                            : 'bg-white/[0.04] text-zinc-400 ring-1 ring-white/[0.08] hover:text-zinc-200',
                        )}
                      >
                        {value.value} <span className="text-zinc-500">({value.count})</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-24">
          <Loader2 size={24} className="animate-spin text-zinc-500" />
        </div>
      ) : filteredProducts.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24">
          <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-white/5">
            <PackageX size={24} className="text-zinc-600" />
          </div>
          <p className="text-sm text-zinc-500">No products found</p>
          <p className="mt-1 text-xs text-zinc-600">Try adjusting your search or active filters.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-5 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {filteredProducts.map((product) => {
            const isAdded = addedIds.has(product.id);
            const thumbnail = product.images?.[0]?.url ?? product.image_url;
            const hasVariants = (product.variants?.length ?? 0) > 0 || (product.attributes?.length ?? 0) > 0;
            const inStock = productHasPurchasableInventory(product);
            const displayPrice = getProductCardPrice(product);

            return (
              <Card
                key={product.id}
                hover
                className="group flex cursor-pointer flex-col overflow-hidden"
                onClick={() => navigate(`/shop/${product.id}`)}
              >
                <div className="relative aspect-[4/3] overflow-hidden bg-zinc-800">
                  {thumbnail ? (
                    <img
                      src={thumbnail}
                      alt={product.name}
                      className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                    />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center">
                      <Package size={32} className="text-zinc-600" />
                    </div>
                  )}

                  {!inStock && (
                    <div className="absolute inset-0 flex items-center justify-center bg-black/50">
                      <span className="rounded-full bg-black/60 px-2.5 py-1 text-xs font-medium text-zinc-300">
                        Unavailable
                      </span>
                    </div>
                  )}

                  {hasVariants && (
                    <div className="absolute right-2 top-2 rounded-full bg-indigo-600/90 px-2 py-0.5 text-[10px] font-semibold text-white">
                      Options
                    </div>
                  )}
                </div>

                <CardBody className="flex flex-1 flex-col gap-2.5">
                  <div className="flex-1 space-y-1">
                    <p className="text-sm font-semibold leading-snug text-zinc-100">{product.name}</p>
                    {product.category && (
                      <p className="text-[11px] font-medium uppercase tracking-wider text-zinc-500">
                        {product.category}
                        {product.subcategory ? ` / ${product.subcategory}` : ''}
                      </p>
                    )}
                    {product.description && (
                      <p className="line-clamp-2 text-xs leading-relaxed text-zinc-500">{product.description}</p>
                    )}
                  </div>

                  <div className="flex items-center justify-between pt-1">
                    <div>
                      <span className="text-sm font-bold tabular-nums text-zinc-100">
                        {formatMoney(displayPrice, product.currency)}
                      </span>
                      {hasVariants && displayPrice !== product.price && (
                        <p className="text-[11px] text-zinc-500">Starting price</p>
                      )}
                    </div>

                    {hasVariants ? (
                      <Button size="sm" onClick={(event) => { event.stopPropagation(); navigate(`/shop/${product.id}`); }}>
                        <Eye size={13} /> View
                      </Button>
                    ) : inStock ? (
                      <Button
                        size="sm"
                        variant={isAdded ? 'success' : 'primary'}
                        onClick={(event) => {
                          event.stopPropagation();
                          handleAddToCart(product);
                        }}
                      >
                        {isAdded ? <><Check size={13} /> Added!</> : <><ShoppingBag size={13} /> Add to Cart</>}
                      </Button>
                    ) : (
                      <Button size="sm" variant="ghost" disabled onClick={(event) => event.stopPropagation()}>
                        Unavailable
                      </Button>
                    )}
                  </div>
                </CardBody>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}

function getActiveVariants(product: Product) {
  return getPurchasableVariants(product);
}

function productHasPurchasableInventory(product: Product) {
  const activeVariants = getActiveVariants(product);
  if (activeVariants.length > 0) {
    return true;
  }
  if ((product.attributes?.length ?? 0) > 0) {
    return false;
  }
  return product.status === 'active' && product.stock > 0;
}

function getProductCardPrice(product: Product) {
  const activeVariants = getActiveVariants(product);
  if (activeVariants.length === 0) {
    return product.price;
  }

  return activeVariants.reduce((lowest, variant) => {
    const currentPrice = variant.price ?? product.price;
    return Math.min(lowest, currentPrice);
  }, Number.MAX_SAFE_INTEGER);
}
