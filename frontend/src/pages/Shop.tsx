import { useState, useCallback, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { Search, ShoppingBag, Check, PackageX, Loader2, Package, Eye } from 'lucide-react';
import { toast } from 'sonner';
import { productsApi, cartApi } from '../lib/api';
import { Button } from '../components/ui/Button';
import { Card, CardBody } from '../components/ui/Card';
import { Input } from '../components/ui/Input';
import { useUser } from '../lib/store';
import { cn, formatMoney } from '../lib/utils';
import type { Product } from '../lib/types';

export function Shop() {
  const [search, setSearch] = useState('');
  const [activeCategory, setActiveCategory] = useState('all');
  const [addedIds, setAddedIds] = useState<Set<string>>(new Set());
  const timers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const user = useUser();

  const { data, isLoading } = useQuery({
    queryKey: ['shop-products'],
    queryFn: () => productsApi.list({ limit: 100, offset: 0 }),
  });

  const products: Product[] = data?.data?.products ?? data?.data?.data ?? [];

  const categories = [
    { key: 'all', label: 'All' },
    ...Array.from(new Set(products.map((p) => p.category).filter(Boolean))).map(
      (c) => ({ key: c, label: c.charAt(0).toUpperCase() + c.slice(1) }),
    ),
  ];

  const filteredProducts = products.filter((p) => {
    if (activeCategory !== 'all' && p.category !== activeCategory) return false;
    if (search.trim()) {
      const q = search.toLowerCase().trim();
      return (
        p.name.toLowerCase().includes(q) ||
        (p.description ?? '').toLowerCase().includes(q)
      );
    }
    return true;
  });

  const addToCartMutation = useMutation({
    mutationFn: (productId: string) =>
      cartApi.addItem({ product_id: productId, quantity: 1 }),
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

      setAddedIds((prev) => {
        const next = new Set(prev);
        next.add(product.id);
        return next;
      });

      const existing = timers.current.get(product.id);
      if (existing) clearTimeout(existing);

      const timer = setTimeout(() => {
        setAddedIds((prev) => {
          const next = new Set(prev);
          next.delete(product.id);
          return next;
        });
        timers.current.delete(product.id);
      }, 1500);

      timers.current.set(product.id, timer);
    },
    [addToCartMutation],
  );

  return (
    <div className="animate-fade-in">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Shop</h1>
        <p className="text-zinc-500 text-sm mt-1">
          {filteredProducts.length}{' '}
          {filteredProducts.length === 1 ? 'product' : 'products'}
        </p>
      </div>

      <div className="mb-6 max-w-md">
        <Input
          placeholder="Search products..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          icon={<Search size={15} />}
        />
      </div>

      {categories.length > 1 && (
        <div className="flex flex-wrap gap-2 mb-8">
          {categories.map((cat) => (
            <button
              key={cat.key}
              onClick={() => setActiveCategory(cat.key)}
              className={cn(
                'px-3.5 py-1.5 rounded-full text-sm font-medium transition-all duration-150 cursor-pointer',
                activeCategory === cat.key
                  ? 'bg-indigo-600 text-white'
                  : 'bg-white/5 text-zinc-400 hover:text-zinc-200',
              )}
            >
              {cat.label}
            </button>
          ))}
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-24">
          <Loader2 size={24} className="animate-spin text-zinc-500" />
        </div>
      ) : filteredProducts.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24">
          <div className="w-14 h-14 rounded-2xl bg-white/5 flex items-center justify-center mb-4">
            <PackageX size={24} className="text-zinc-600" />
          </div>
          <p className="text-zinc-500 text-sm">No products found</p>
          <p className="text-zinc-600 text-xs mt-1">
            Try adjusting your search or filter
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
          {filteredProducts.map((product) => {
            const isAdded = addedIds.has(product.id);
            const inStock = product.stock > 0 && product.status === 'active';

            const thumbnail = product.images?.[0]?.url ?? product.image_url;
            const hasVariants = (product.variants?.length ?? 0) > 0 || (product.attributes?.length ?? 0) > 0;

            return (
              <Card key={product.id} hover className="overflow-hidden flex flex-col cursor-pointer group" onClick={() => navigate(`/shop/${product.id}`)}>
                <div className="relative aspect-[4/3] bg-zinc-800 overflow-hidden">
                  {thumbnail ? (
                    <img
                      src={thumbnail}
                      alt={product.name}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center">
                      <Package size={32} className="text-zinc-600" />
                    </div>
                  )}
                  {!inStock && (
                    <div className="absolute inset-0 bg-black/50 flex items-center justify-center">
                      <span className="text-xs font-medium text-zinc-300 bg-black/60 px-2.5 py-1 rounded-full">Out of Stock</span>
                    </div>
                  )}
                  {hasVariants && (
                    <div className="absolute top-2 right-2 bg-indigo-600/90 text-white text-[10px] font-semibold px-2 py-0.5 rounded-full">
                      Options
                    </div>
                  )}
                </div>

                <CardBody className="flex flex-col flex-1 gap-2.5">
                  <div className="flex-1 space-y-1">
                    <p className="text-sm font-semibold text-zinc-100 leading-snug">
                      {product.name}
                    </p>
                    {product.category && (
                      <p className="text-[11px] text-zinc-500 uppercase tracking-wider font-medium">
                        {product.category}
                      </p>
                    )}
                    {product.description && (
                      <p className="text-xs text-zinc-500 line-clamp-2 leading-relaxed">
                        {product.description}
                      </p>
                    )}
                  </div>

                  <div className="flex items-center justify-between pt-1">
                    <span className="text-sm font-bold text-zinc-100 tabular-nums">
                      {formatMoney(product.price, product.currency)}
                    </span>

                    {hasVariants ? (
                      <Button
                        size="sm"
                        onClick={(e) => { e.stopPropagation(); navigate(`/shop/${product.id}`); }}
                      >
                        <Eye size={13} /> View
                      </Button>
                    ) : inStock ? (
                      <Button
                        size="sm"
                        variant={isAdded ? 'success' : 'primary'}
                        onClick={(e) => { e.stopPropagation(); handleAddToCart(product); }}
                      >
                        {isAdded ? <><Check size={13} /> Added!</> : <><ShoppingBag size={13} /> Add to Cart</>}
                      </Button>
                    ) : (
                      <Button size="sm" variant="ghost" disabled onClick={(e) => e.stopPropagation()}>
                        Out of Stock
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
