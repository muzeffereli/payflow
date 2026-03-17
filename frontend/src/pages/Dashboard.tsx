import { useQuery } from '@tanstack/react-query';
import {
  Package, Wallet, Clock, CheckCircle, ArrowRight,
  TrendingUp, Sparkles, DollarSign, ShoppingBag,
} from 'lucide-react';
import { Link } from 'react-router-dom';
import { ordersApi, walletApi, storesApi } from '../lib/api';
import { StatCard } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { PageSpinner } from '../components/ui/Spinner';
import { formatMoney, formatDate } from '../lib/utils';
import type { Order } from '../lib/types';
import { useUser, useIsSeller } from '../lib/store';

const WALLET_REFRESH_MS = 5000;

function getGreeting(): string {
  const h = new Date().getHours();
  if (h < 12) return 'Good morning';
  if (h < 18) return 'Good afternoon';
  return 'Good evening';
}

export function Dashboard() {
  const user = useUser();
  const isSeller = useIsSeller();
  const ordersQueryKey = ['orders', user?.id];
  const walletQueryKey = ['wallet', user?.id];

  const { data: ordersData, isLoading: ordersLoading } = useQuery({
    queryKey: ordersQueryKey,
    queryFn: () => ordersApi.list(10),
    enabled: !!user?.id,
  });

  const { data: walletData } = useQuery({
    queryKey: walletQueryKey,
    queryFn: () => walletApi.get(),
    enabled: !!user?.id,
    retry: false,
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
    refetchInterval: WALLET_REFRESH_MS,
    refetchIntervalInBackground: false,
  });

  const { data: storeData } = useQuery({
    queryKey: ['my-store'],
    queryFn: () => storesApi.getMe(),
    enabled: isSeller,
  });
  const myStore = storeData?.data;

  const { data: analyticsData } = useQuery({
    queryKey: ['seller-analytics', myStore?.id],
    queryFn: () => ordersApi.sellerAnalytics(myStore!.id),
    enabled: !!myStore?.id,
  });
  const analytics = analyticsData?.data;

  const orders: Order[] = ordersData?.data?.orders ?? [];
  const wallet = walletData?.data?.wallet ?? walletData?.data;

  const stats = {
    total: orders.length,
    pending: orders.filter((o) => o.status === 'pending').length,
    paid: orders.filter((o) => o.status === 'paid').length,
    cancelled: orders.filter((o) => o.status === 'cancelled').length,
  };

  return (
    <div className="animate-fade-in">
      <div className="relative rounded-2xl overflow-hidden mb-8 ring-1 ring-white/[0.06]">
        <div className="absolute inset-0 bg-gradient-to-br from-indigo-600/20 via-violet-600/10 to-transparent" />
        <div className="absolute inset-0 dot-grid opacity-30" />
        <div className="absolute top-0 right-0 w-80 h-80 bg-indigo-500/10 rounded-full blur-[80px]" />

        <div className="relative px-8 py-10">
          <div className="flex items-start justify-between">
            <div>
              <div className="flex items-center gap-2 mb-3">
                <div className="w-8 h-8 rounded-lg bg-indigo-500/10 flex items-center justify-center">
                  <Sparkles size={14} className="text-indigo-400" />
                </div>
                <span className="text-xs font-medium text-indigo-400 uppercase tracking-wider">Dashboard</span>
              </div>
              <h1 className="text-3xl font-bold text-zinc-100 tracking-tight">
                {getGreeting()}, {user?.name?.split(' ')[0]}
              </h1>
              <p className="text-zinc-500 mt-2 text-sm max-w-md">
                Here's an overview of your account activity and recent transactions.
              </p>
            </div>
            <Link
              to="/orders"
              className="hidden sm:inline-flex items-center gap-2 px-4 py-2.5 rounded-xl bg-white/5 ring-1 ring-white/10 text-sm text-zinc-300 hover:bg-white/10 hover:text-white transition-all duration-150"
            >
              View all orders
              <ArrowRight size={14} />
            </Link>
          </div>
        </div>
      </div>

      {isSeller && analytics && (
        <div className="mb-8">
          <div className="flex items-center gap-2.5 mb-4">
            <div className="w-7 h-7 rounded-lg bg-emerald-500/10 flex items-center justify-center">
              <ShoppingBag size={14} className="text-emerald-400" />
            </div>
            <h2 className="text-sm font-semibold text-zinc-200">Store Analytics</h2>
            {myStore && <span className="text-xs text-zinc-600">({myStore.name})</span>}
          </div>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 stagger-children">
            <StatCard
              title="Store Revenue"
              value={formatMoney(analytics.total_revenue)}
              icon={<DollarSign size={20} />}
              color="green"
            />
            <StatCard
              title="Store Orders"
              value={analytics.total_orders}
              icon={<Package size={20} />}
              color="indigo"
            />
            <StatCard
              title="Paid Orders"
              value={analytics.paid_orders}
              icon={<CheckCircle size={20} />}
              color="blue"
            />
            <StatCard
              title="Pending Orders"
              value={analytics.pending_orders}
              icon={<Clock size={20} />}
              color="amber"
            />
          </div>
        </div>
      )}

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8 stagger-children">
        <StatCard
          title="Total Orders"
          value={stats.total}
          icon={<Package size={20} />}
          color="indigo"
        />
        <StatCard
          title="Wallet Balance"
          value={wallet ? formatMoney(wallet.balance, wallet.currency) : '--'}
          icon={<Wallet size={20} />}
          color="green"
        />
        <StatCard
          title="Paid Orders"
          value={stats.paid}
          icon={<CheckCircle size={20} />}
          color="blue"
        />
        <StatCard
          title="Pending"
          value={stats.pending}
          icon={<Clock size={20} />}
          color="amber"
        />
      </div>

      <div className="bg-zinc-900/50 ring-1 ring-white/[0.06] rounded-2xl overflow-hidden">
        <div className="px-6 py-4 border-b border-white/[0.06] flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-lg bg-indigo-500/10 flex items-center justify-center">
              <TrendingUp size={14} className="text-indigo-400" />
            </div>
            <h2 className="text-sm font-semibold text-zinc-200">Recent Orders</h2>
          </div>
          <Link
            to="/orders"
            className="text-xs text-zinc-500 hover:text-indigo-400 transition-colors duration-150 flex items-center gap-1"
          >
            View all <ArrowRight size={12} />
          </Link>
        </div>

        {ordersLoading ? (
          <PageSpinner />
        ) : orders.length === 0 ? (
          <div className="py-16 text-center">
            <div className="w-14 h-14 rounded-2xl bg-white/5 flex items-center justify-center mx-auto mb-4">
              <Package size={24} className="text-zinc-600" />
            </div>
            <p className="text-zinc-500 text-sm mb-1">No orders yet</p>
            <p className="text-zinc-600 text-xs">Create your first order to get started</p>
          </div>
        ) : (
          <div className="divide-y divide-white/[0.04]">
            {orders.slice(0, 8).map((order) => (
              <Link
                key={order.id}
                to="/orders"
                className="px-6 py-4 flex items-center gap-4 hover:bg-white/[0.02] transition-colors duration-150 group"
              >
                <div className="w-10 h-10 rounded-xl bg-indigo-500/5 ring-1 ring-indigo-500/10 flex items-center justify-center shrink-0 group-hover:bg-indigo-500/10 transition-colors duration-150">
                  <Package size={16} className="text-indigo-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-zinc-200 truncate">
                    Order #{order.id.slice(0, 8)}
                  </p>
                  <p className="text-xs text-zinc-600 mt-0.5">{formatDate(order.created_at)}</p>
                </div>
                <Badge status={order.status} />
                <p className="text-sm font-semibold text-zinc-300 ml-2 shrink-0 tabular-nums">
                  {formatMoney(order.total_amount, order.currency)}
                </p>
                <ArrowRight size={14} className="text-zinc-700 group-hover:text-zinc-500 transition-colors duration-150 shrink-0" />
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
