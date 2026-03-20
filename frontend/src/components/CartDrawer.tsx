import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { X, ShoppingBag, Trash2, Minus, Plus, Loader2, CheckCircle, CreditCard, Wallet } from 'lucide-react';
import { cartApi, getApiErrorMessage, walletApi } from '../lib/api';
import { useUser } from '../lib/store';
import { cn, formatMoney } from '../lib/utils';
import { Button } from './ui/Button';
import { Badge } from './ui/Badge';

interface CartDrawerProps {
  open: boolean;
  onClose: () => void;
}

interface CartViewItem {
  product_id: string;
  variant_id?: string;
  variant_label?: string;
  variant_sku?: string;
  name: string;
  quantity: number;
  unit_price: number;
  line_total: number;
  currency: string;
}

interface CartView {
  user_id: string;
  items: CartViewItem[];
  total_cents: number;
  currency: string;
}

const WALLET_REFRESH_MS = 5000;

export function CartDrawer({ open, onClose }: CartDrawerProps) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const user = useUser();
  const [checkoutSuccess, setCheckoutSuccess] = useState(false);
  const [orderIds, setOrderIds] = useState<string[]>([]);
  const [paymentMethod, setPaymentMethod] = useState<'card' | 'wallet'>('card');
  const [checkoutError, setCheckoutError] = useState('');
  const cartQueryKey = ['cart', user?.id];
  const ordersQueryKey = ['orders', user?.id];
  const walletQueryKey = ['wallet', user?.id];

  const { data, isLoading } = useQuery({
    queryKey: cartQueryKey,
    queryFn: () => cartApi.get(),
    enabled: open && !!user?.id,
    refetchOnWindowFocus: false,
  });

  const { data: walletData } = useQuery({
    queryKey: walletQueryKey,
    queryFn: () => walletApi.get(),
    enabled: open && !!user?.id,
    retry: false,
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
    refetchInterval: open ? WALLET_REFRESH_MS : false,
    refetchIntervalInBackground: false,
  });

  const cart: CartView | null = data?.data ?? null;
  const items = cart?.items ?? [];
  const isEmpty = items.length === 0;
  const totalCents = cart?.total_cents ?? 0;
  const currency = cart?.currency || 'USD';
  const count = items.reduce((sum, it) => sum + it.quantity, 0);
  const wallet = walletData?.data?.wallet ?? walletData?.data ?? null;
  const walletBalance = wallet?.balance ?? 0;
  const walletCurrency = wallet?.currency ?? currency;
  const hasWallet = !!wallet;
  const walletHasFunds = hasWallet && walletCurrency === currency && walletBalance >= totalCents;

  const updateQtyMutation = useMutation({
    mutationFn: ({
      productId,
      variantId,
      qty,
    }: {
      productId: string;
      variantId?: string;
      qty: number;
    }) => cartApi.updateQuantity(productId, qty, variantId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: cartQueryKey }),
  });

  const removeMutation = useMutation({
    mutationFn: ({ productId, variantId }: { productId: string; variantId?: string }) =>
      cartApi.removeItem(productId, variantId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: cartQueryKey }),
  });

  const checkoutMutation = useMutation({
    mutationFn: () => cartApi.checkout(`checkout-${Date.now()}`, paymentMethod),
    onSuccess: (res) => {
      const ids = res.data?.order_ids ?? [];
      setOrderIds(ids);
      setCheckoutSuccess(true);
      setCheckoutError('');
      queryClient.invalidateQueries({ queryKey: cartQueryKey });
      queryClient.invalidateQueries({ queryKey: ordersQueryKey });
      queryClient.invalidateQueries({ queryKey: walletQueryKey });
      queryClient.invalidateQueries({ queryKey: ['wallet-transactions', user?.id] });
    },
    onError: (error) => {
      setCheckoutError(getApiErrorMessage(error, 'Checkout failed. Please try again.'));
    },
  });

  const handleClose = () => {
    if (checkoutSuccess) {
      setCheckoutSuccess(false);
      setOrderIds([]);
    }
    setCheckoutError('');
    onClose();
  };

  const walletLabel = hasWallet
    ? `${formatMoney(walletBalance, walletCurrency)} available`
    : 'Create and top up a wallet first';
  const walletHint = !hasWallet
    ? 'Wallet payments require an existing wallet.'
    : walletCurrency !== currency
      ? `Wallet currency ${walletCurrency} does not match cart currency ${currency}.`
      : walletHasFunds
        ? 'Your wallet balance covers this order.'
        : 'Not enough wallet balance for this order.';

  return (
    <div
      className={cn(
        'fixed inset-0 z-50 transition-opacity duration-300',
        open ? 'pointer-events-auto' : 'pointer-events-none',
      )}
    >
      <div
        className={cn(
          'absolute inset-0 bg-black/60 backdrop-blur-sm transition-opacity duration-300',
          open ? 'opacity-100' : 'opacity-0',
        )}
        onClick={handleClose}
      />

      <aside
        className={cn(
          'fixed right-0 top-0 h-full w-full max-w-md',
          'bg-[var(--color-bg)] border-l border-white/[0.06]',
          'flex flex-col',
          'transition-transform duration-300 ease-in-out',
          open ? 'translate-x-0' : 'translate-x-full',
        )}
      >
        <div className="flex items-center justify-between px-6 py-5 border-b border-white/[0.06]">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold text-zinc-100">Shopping Cart</h2>
            {count > 0 && (
              <span className="inline-flex items-center justify-center bg-indigo-600 text-white text-xs font-medium rounded-full px-2 py-0.5 min-w-[1.25rem]">
                {count}
              </span>
            )}
          </div>
          <button
            onClick={handleClose}
            className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-white/5 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {checkoutSuccess ? (
          <div className="flex-1 flex flex-col items-center justify-center gap-4 px-6">
            <div className="rounded-2xl bg-emerald-500/10 p-5">
              <CheckCircle className="h-10 w-10 text-emerald-400" />
            </div>
            <div className="text-center">
              <p className="text-zinc-200 font-medium">Order placed!</p>
              <p className="text-sm text-zinc-500 mt-1">
                {orderIds.length} order{orderIds.length !== 1 ? 's' : ''} created
              </p>
            </div>
            <Button
              variant="primary"
              size="md"
              onClick={() => {
                handleClose();
                navigate('/orders');
              }}
              className="mt-2"
            >
              View Orders
            </Button>
            <Button variant="secondary" size="md" onClick={handleClose}>
              Continue Shopping
            </Button>
          </div>
        ) : isLoading ? (
          <div className="flex-1 flex items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-zinc-500" />
          </div>
        ) : isEmpty ? (
          <div className="flex-1 flex flex-col items-center justify-center gap-4 px-6">
            <div className="rounded-2xl bg-zinc-800/50 p-5">
              <ShoppingBag className="h-10 w-10 text-zinc-600" />
            </div>
            <div className="text-center">
              <p className="text-zinc-200 font-medium">Your cart is empty</p>
              <p className="text-sm text-zinc-500 mt-1">Browse the shop to add items</p>
            </div>
            <Button variant="secondary" size="md" onClick={handleClose} className="mt-2">
              Continue Shopping
            </Button>
          </div>
        ) : (
          <>
            <ul className="flex-1 overflow-y-auto divide-y divide-white/[0.06] px-6">
              {items.map((item) => (
                <li key={`${item.product_id}:${item.variant_id ?? 'base'}`} className="py-4 flex gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-zinc-100 truncate">
                          {item.name}
                        </p>
                        {(item.variant_label || item.variant_sku) && (
                          <p className="mt-0.5 text-xs text-zinc-500">
                            {[item.variant_label, item.variant_sku].filter(Boolean).join(' · ')}
                          </p>
                        )}
                        <p className="text-xs text-zinc-500 mt-0.5">
                          {formatMoney(item.unit_price, item.currency)}
                        </p>
                      </div>
                      <p className="text-sm font-medium text-zinc-200 whitespace-nowrap flex-shrink-0">
                        {formatMoney(item.line_total, item.currency)}
                      </p>
                    </div>

                    <div className="flex items-center justify-between mt-2.5">
                      <div className="inline-flex items-center bg-white/5 rounded-lg">
                        <button
                          onClick={() => {
                            if (item.quantity <= 1) {
                              removeMutation.mutate({ productId: item.product_id, variantId: item.variant_id });
                            } else {
                              updateQtyMutation.mutate({
                                productId: item.product_id,
                                variantId: item.variant_id,
                                qty: item.quantity - 1,
                              });
                            }
                          }}
                          className="p-1.5 text-zinc-400 hover:text-zinc-200 transition-colors"
                        >
                          <Minus className="h-3.5 w-3.5" />
                        </button>
                        <span className="w-8 text-center text-sm font-medium text-zinc-200 tabular-nums">
                          {item.quantity}
                        </span>
                        <button
                          onClick={() =>
                            updateQtyMutation.mutate({
                              productId: item.product_id,
                              variantId: item.variant_id,
                              qty: item.quantity + 1,
                            })
                          }
                          className="p-1.5 text-zinc-400 hover:text-zinc-200 transition-colors"
                        >
                          <Plus className="h-3.5 w-3.5" />
                        </button>
                      </div>
                      <button
                        onClick={() => removeMutation.mutate({ productId: item.product_id, variantId: item.variant_id })}
                        className="p-1.5 text-zinc-600 hover:text-red-400 transition-colors"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </li>
              ))}
            </ul>

            <div className="border-t border-white/[0.06] px-6 py-5 space-y-4">
              <div className="space-y-2">
                <p className="text-xs font-medium uppercase tracking-[0.18em] text-zinc-500">Payment method</p>
                <div className="grid gap-2">
                  <button
                    type="button"
                    onClick={() => setPaymentMethod('card')}
                    className={cn(
                      'rounded-xl border px-4 py-3 text-left transition-colors',
                      paymentMethod === 'card'
                        ? 'border-indigo-500/50 bg-indigo-500/10'
                        : 'border-white/[0.08] bg-white/[0.03] hover:bg-white/[0.05]',
                    )}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-3">
                        <div className="rounded-lg bg-white/5 p-2 text-zinc-300">
                          <CreditCard className="h-4 w-4" />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-zinc-100">Card</p>
                          <p className="text-xs text-zinc-500">Pay through the standard payment flow</p>
                        </div>
                      </div>
                      {paymentMethod === 'card' && <Badge status="confirmed" />}
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={() => {
                      if (walletHasFunds) {
                        setPaymentMethod('wallet');
                      }
                    }}
                    disabled={!walletHasFunds}
                    className={cn(
                      'rounded-xl border px-4 py-3 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-60',
                      paymentMethod === 'wallet'
                        ? 'border-emerald-500/50 bg-emerald-500/10'
                        : 'border-white/[0.08] bg-white/[0.03] hover:bg-white/[0.05]',
                    )}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-3">
                        <div className="rounded-lg bg-white/5 p-2 text-zinc-300">
                          <Wallet className="h-4 w-4" />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-zinc-100">Wallet</p>
                          <p className="text-xs text-zinc-500">{walletLabel}</p>
                        </div>
                      </div>
                      {paymentMethod === 'wallet' && <Badge status="paid" />}
                    </div>
                    <p className="mt-2 text-xs text-zinc-500">{walletHint}</p>
                  </button>
                </div>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-zinc-200">Total</span>
                <span className="text-lg font-semibold text-indigo-400">
                  {formatMoney(totalCents, currency)}
                </span>
              </div>

              <div className="space-y-2">
                <Button
                  variant="primary"
                  size="lg"
                  className="w-full"
                  onClick={() => checkoutMutation.mutate()}
                  disabled={checkoutMutation.isPending}
                >
                  {checkoutMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    paymentMethod === 'wallet' ? 'Pay with Wallet' : 'Checkout'
                  )}
                </Button>
                {checkoutError && (
                  <p className="text-xs text-red-400 text-center">
                    {checkoutError}
                  </p>
                )}
                <Button variant="ghost" size="lg" className="w-full" onClick={handleClose}>
                  Continue Shopping
                </Button>
              </div>
            </div>
          </>
        )}
      </aside>
    </div>
  );
}
