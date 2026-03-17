import { useQuery } from '@tanstack/react-query';
import { LayoutDashboard, Users, Package, ShieldAlert, ArrowDownToLine, Wallet } from 'lucide-react';
import { adminApi, ordersApi, fraudApi, withdrawalsApi, walletApi } from '../lib/api';
import { PageSpinner } from '../components/ui/Spinner';
import { formatMoney } from '../lib/utils';

const WALLET_REFRESH_MS = 5000;

export function AdminDashboard() {
  const { data: usersData, isLoading: usersLoading } = useQuery({
    queryKey: ['admin-users', 0],
    queryFn: () => adminApi.listUsers(1, 0),
  });

  const { data: ordersData, isLoading: ordersLoading } = useQuery({
    queryKey: ['admin-orders'],
    queryFn: () => ordersApi.list(1, 0),
  });

  const { data: fraudData, isLoading: fraudLoading } = useQuery({
    queryKey: ['admin-fraud-stats'],
    queryFn: () => fraudApi.list({ limit: 1 }),
  });

  const { data: withdrawalsData, isLoading: withdrawalsLoading } = useQuery({
    queryKey: ['admin-withdrawals-stats'],
    queryFn: () => withdrawalsApi.pendingList(1, 0),
  });

  const { data: walletData, isLoading: walletLoading } = useQuery({
    queryKey: ['admin-platform-wallet'],
    queryFn: () => walletApi.get(),
    retry: false,
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
    refetchInterval: WALLET_REFRESH_MS,
    refetchIntervalInBackground: false,
  });

  const isLoading = usersLoading || ordersLoading || fraudLoading || withdrawalsLoading || walletLoading;

  const totalUsers = usersData?.data?.total ?? 0;
  const totalOrders = ordersData?.data?.total ?? 0;
  const totalFraud = fraudData?.data?.total ?? 0;
  const pendingWithdrawals = (withdrawalsData?.data?.withdrawals ?? []).length;
  const wallet = walletData?.data?.wallet ?? walletData?.data;
  const platformBalance = wallet ? formatMoney(wallet.balance, wallet.currency) : '--';

  if (isLoading) return <PageSpinner />;

  const stats = [
    { label: 'Platform Wallet', value: platformBalance, icon: Wallet, color: 'emerald' },
    { label: 'Total Users', value: totalUsers, icon: Users, color: 'blue' },
    { label: 'Total Orders', value: totalOrders, icon: Package, color: 'indigo' },
    { label: 'Fraud Checks', value: totalFraud, icon: ShieldAlert, color: 'orange' },
    { label: 'Pending Withdrawals', value: pendingWithdrawals, icon: ArrowDownToLine, color: 'amber' },
  ];

  const colorMap: Record<string, { bg: string; ring: string; text: string; label: string }> = {
    emerald: { bg: 'bg-emerald-500/5', ring: 'ring-emerald-500/10', text: 'text-emerald-400', label: 'text-emerald-500' },
    blue: { bg: 'bg-blue-500/5', ring: 'ring-blue-500/10', text: 'text-blue-400', label: 'text-blue-500' },
    indigo: { bg: 'bg-indigo-500/5', ring: 'ring-indigo-500/10', text: 'text-indigo-300', label: 'text-indigo-400' },
    orange: { bg: 'bg-orange-500/5', ring: 'ring-orange-500/10', text: 'text-orange-400', label: 'text-orange-500' },
    amber: { bg: 'bg-amber-500/5', ring: 'ring-amber-500/10', text: 'text-amber-400', label: 'text-amber-500' },
  };

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center gap-4">
        <div className="w-11 h-11 rounded-xl bg-indigo-500/10 ring-1 ring-indigo-500/10 flex items-center justify-center">
          <LayoutDashboard size={20} className="text-indigo-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Admin Dashboard</h1>
          <p className="text-zinc-500 text-sm">Platform overview and quick stats</p>
        </div>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 stagger-children">
        {stats.map((stat) => {
          const c = colorMap[stat.color];
          const Icon = stat.icon;
          return (
            <div key={stat.label} className={`${c.bg} ring-1 ${c.ring} rounded-2xl p-6`}>
              <div className="flex items-center justify-between mb-3">
                <p className={`text-[10px] font-semibold uppercase tracking-wider ${c.label}`}>
                  {stat.label}
                </p>
                <Icon size={16} className={c.text} />
              </div>
              <p className={`text-3xl font-bold tabular-nums ${c.text}`}>{stat.value}</p>
            </div>
          );
        })}
      </div>
    </div>
  );
}
