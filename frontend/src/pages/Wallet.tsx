import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Wallet as WalletIcon, Plus, ArrowUpRight, ArrowDownLeft, TrendingUp, ChevronLeft, ChevronRight, DollarSign } from 'lucide-react';
import { toast } from 'sonner';
import { walletApi } from '../lib/api';
import { Button } from '../components/ui/Button';
import { Card, CardHeader, CardBody } from '../components/ui/Card';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { formatMoney, formatDate, cn } from '../lib/utils';
import type { WalletTransaction } from '../lib/types';
import { useUser } from '../lib/store';

const PAGE_SIZE = 20;
const PRESET_AMOUNTS = [1000, 2500, 5000, 10000]; // cents: $10, $25, $50, $100
const WALLET_REFRESH_MS = 5000;

export function Wallet() {
  const qc = useQueryClient();
  const user = useUser();
  const [page, setPage] = useState(0);
  const [showTopUp, setShowTopUp] = useState(false);
  const [topUpAmount, setTopUpAmount] = useState('');
  const walletQueryKey = ['wallet', user?.id];
  const walletTransactionsQueryKey = ['wallet-transactions', user?.id, page];

  const { data, isLoading, error } = useQuery({
    queryKey: walletQueryKey,
    queryFn: () => walletApi.get(),
    enabled: !!user?.id,
    retry: false,
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
    refetchInterval: WALLET_REFRESH_MS,
    refetchIntervalInBackground: false,
  });

  const { data: txData, isLoading: txLoading } = useQuery({
    queryKey: walletTransactionsQueryKey,
    queryFn: () => walletApi.transactions(PAGE_SIZE, page * PAGE_SIZE),
    enabled: !!data?.data,
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
    refetchInterval: WALLET_REFRESH_MS,
    refetchIntervalInBackground: false,
  });

  const createMutation = useMutation({
    mutationFn: () => walletApi.create(),
    onSuccess: () => qc.invalidateQueries({ queryKey: walletQueryKey }),
  });

  const topUpMutation = useMutation({
    mutationFn: (amount: number) => walletApi.topup(amount),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: walletQueryKey });
      qc.invalidateQueries({ queryKey: ['wallet-transactions', user?.id] });
      setShowTopUp(false);
      toast.success('Wallet topped up successfully');
      setTopUpAmount('');
    },
  });

  if (isLoading) return <PageSpinner />;

  const wallet = data?.data?.wallet ?? data?.data;

  if (!wallet || error) {
    return (
      <div className="animate-fade-in">
        <h1 className="text-2xl font-bold text-zinc-100 tracking-tight mb-8">Wallet</h1>
        <Card>
          <CardBody className="py-20 text-center">
            <div className="w-16 h-16 rounded-2xl bg-indigo-500/10 flex items-center justify-center mx-auto mb-5">
              <WalletIcon size={28} className="text-indigo-400" />
            </div>
            <p className="text-zinc-300 font-medium mb-1">You don't have a wallet yet</p>
            <p className="text-zinc-500 text-sm mb-8">Create one to start managing your balance</p>
            <Button
              onClick={() => createMutation.mutate()}
              loading={createMutation.isPending}
              size="lg"
            >
              <Plus size={16} /> Create Wallet
            </Button>
          </CardBody>
        </Card>
      </div>
    );
  }

  const transactions = txData?.data?.transactions ?? [];
  const totalTxns = txData?.data?.total ?? 0;
  const totalPages = Math.ceil(totalTxns / PAGE_SIZE);
  const topUpDisplayAmount = topUpAmount ? formatMoney(Math.round(Number(topUpAmount) * 100), 'USD') : '$0.00';

  return (
    <div className="animate-fade-in">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Wallet</h1>
        <p className="text-zinc-500 text-sm mt-1">Manage your balance and transactions</p>
      </div>

      <div className="relative rounded-2xl overflow-hidden mb-8 ring-1 ring-white/10">
        <div className="absolute inset-0 bg-gradient-to-br from-indigo-600 to-violet-700" />
        <div className="absolute inset-0 dot-grid opacity-20" />
        <div className="absolute top-0 right-0 w-64 h-64 bg-white/5 rounded-full blur-[80px]" />

        <div className="relative p-8">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-indigo-200/70 text-xs font-medium uppercase tracking-wider mb-2">Available Balance</p>
              <p className="text-4xl font-bold text-white tracking-tight tabular-nums">
                {formatMoney(wallet.balance, wallet.currency)}
              </p>
              <p className="text-indigo-200/50 text-sm mt-3">
                {wallet.currency} &middot; Updated {formatDate(wallet.updated_at)}
              </p>
            </div>
            <div className="flex flex-col items-end gap-3">
              <div className="w-12 h-12 rounded-xl bg-white/10 flex items-center justify-center backdrop-blur-sm">
                <WalletIcon size={22} className="text-white" />
              </div>
              <button
                onClick={() => setShowTopUp(true)}
                className="px-4 py-2 rounded-lg bg-white/15 hover:bg-white/25 text-white text-sm font-medium transition-colors backdrop-blur-sm"
              >
                <Plus size={14} className="inline mr-1.5 -mt-0.5" />
                Top Up
              </button>
            </div>
          </div>
        </div>
      </div>

      <Modal open={showTopUp} onClose={() => setShowTopUp(false)} title="Top Up Wallet" size="sm">
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            const cents = Math.round(Number(topUpAmount) * 100);
            if (cents > 0) topUpMutation.mutate(cents);
          }}
        >
          <div className="rounded-xl bg-white/[0.03] ring-1 ring-white/[0.06] p-3">
            <div className="flex items-center gap-2.5 mb-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-500/10">
                <DollarSign size={16} className="text-indigo-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-zinc-200">Choose an amount</p>
                <p className="text-xs text-zinc-500">Top up instantly in USD</p>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              {PRESET_AMOUNTS.map((amt) => (
                <button
                  key={amt}
                  type="button"
                  onClick={() => setTopUpAmount(String(amt / 100))}
                  className={cn(
                    'rounded-lg px-3 py-2 text-sm font-medium transition-colors ring-1',
                    Number(topUpAmount) === amt / 100
                      ? 'bg-indigo-600 text-white ring-indigo-500/40'
                      : 'bg-white/[0.03] text-zinc-400 ring-white/[0.06] hover:bg-white/[0.06] hover:text-zinc-200',
                  )}
                >
                  ${amt / 100}
                </button>
              ))}
            </div>
          </div>

          <Input
            label="Custom Amount (USD)"
            type="number"
            min="1"
            step="0.01"
            placeholder="0.00"
            value={topUpAmount}
            onChange={(e) => setTopUpAmount(e.target.value)}
            icon={<DollarSign size={15} />}
          />

          {topUpMutation.isError && (
            <p className="rounded-lg bg-red-500/10 px-4 py-3 text-xs text-red-400">
              Top-up failed. Please try again.
            </p>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="ghost" onClick={() => setShowTopUp(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              size="lg"
              disabled={!topUpAmount || Number(topUpAmount) <= 0 || topUpMutation.isPending}
              loading={topUpMutation.isPending}
            >
              Add {topUpDisplayAmount}
            </Button>
          </div>
        </form>
      </Modal>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <div className="w-7 h-7 rounded-lg bg-indigo-500/10 flex items-center justify-center">
                <TrendingUp size={14} className="text-indigo-400" />
              </div>
              <h2 className="text-sm font-semibold text-zinc-200">Transaction History</h2>
              {totalTxns > 0 && (
                <span className="text-xs text-zinc-500">{totalTxns} total</span>
              )}
            </div>
            {totalPages > 1 && (
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                  disabled={page === 0}
                  className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-white/5 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
                >
                  <ChevronLeft size={14} />
                </button>
                <span className="text-xs text-zinc-500 tabular-nums">
                  {page + 1} / {totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                  disabled={page >= totalPages - 1}
                  className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-white/5 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
                >
                  <ChevronRight size={14} />
                </button>
              </div>
            )}
          </div>
        </CardHeader>
        <CardBody className="p-0">
          {txLoading ? (
            <div className="py-12"><PageSpinner /></div>
          ) : transactions.length === 0 ? (
            <div className="py-16 text-center">
              <div className="w-14 h-14 rounded-2xl bg-white/5 flex items-center justify-center mx-auto mb-4">
                <TrendingUp size={24} className="text-zinc-600" />
              </div>
              <p className="text-zinc-500 text-sm">No transactions yet</p>
            </div>
          ) : (
            <div className="divide-y divide-white/[0.04]">
              {transactions.map((tx: WalletTransaction) => (
                <div key={tx.id} className="px-6 py-4 flex items-center gap-4 hover:bg-white/[0.02] transition-colors duration-150">
                  <div
                    className={cn(
                      'w-10 h-10 rounded-xl flex items-center justify-center shrink-0',
                      tx.type === 'credit'
                        ? 'bg-emerald-500/10 text-emerald-400'
                        : 'bg-red-500/10 text-red-400',
                    )}
                  >
                    {tx.type === 'credit' ? (
                      <ArrowDownLeft size={16} />
                    ) : (
                      <ArrowUpRight size={16} />
                    )}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-zinc-200 capitalize">
                      {tx.source.replace(/_/g, ' ')}
                    </p>
                    <p className="text-xs text-zinc-600 mt-0.5">
                      {formatDate(tx.created_at)}
                      {tx.reference_id && (
                        <span className="ml-2 text-zinc-700">ref: {tx.reference_id.slice(0, 8)}...</span>
                      )}
                    </p>
                  </div>
                  <div className="text-right">
                    <p
                      className={cn(
                        'text-sm font-semibold tabular-nums',
                        tx.type === 'credit' ? 'text-emerald-400' : 'text-red-400',
                      )}
                    >
                      {tx.type === 'credit' ? '+' : '-'}
                      {formatMoney(tx.amount)}
                    </p>
                    <p className="text-xs text-zinc-600 mt-0.5 tabular-nums">
                      Balance: {formatMoney(tx.balance_after)}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
