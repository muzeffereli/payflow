import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { CreditCard, RefreshCw } from 'lucide-react';
import { ordersApi, paymentsApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Modal } from '../components/ui/Modal';
import { Table, Thead, Th, Tbody, Tr, Td, EmptyRow } from '../components/ui/Table';
import { PageSpinner } from '../components/ui/Spinner';
import { formatMoney, formatDate } from '../lib/utils';
import { useUser } from '../lib/store';
import type { Order, Payment } from '../lib/types';

export function Payments() {
  const qc = useQueryClient();
  const user = useUser();
  const [selected, setSelected] = useState<Payment | null>(null);
  const [viewError, setViewError] = useState('');
  const ordersQueryKey = ['orders', user?.id];

  const { data: ordersData, isLoading } = useQuery({
    queryKey: ordersQueryKey,
    queryFn: () => ordersApi.list(50),
    enabled: !!user?.id,
  });

  const refundMutation = useMutation({
    mutationFn: (id: string) => paymentsApi.refund(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ordersQueryKey });
      setSelected(null);
    },
  });

  const orders: Order[] = ordersData?.data?.orders ?? [];
  const paidOrders = orders.filter((o) =>
    ['paid', 'refunded', 'confirmed', 'processing'].includes(o.status),
  );

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Payments</h1>
          <p className="text-zinc-500 text-sm mt-1">View and manage payment transactions</p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => qc.invalidateQueries({ queryKey: ordersQueryKey })}
        >
          <RefreshCw size={14} /> Refresh
        </Button>
      </div>

      <Card>
        <Table>
          <Thead>
            <tr>
              <Th>Order ID</Th>
              <Th>Amount</Th>
              <Th>Currency</Th>
              <Th>Order Status</Th>
              <Th>Date</Th>
              <Th />
            </tr>
          </Thead>
          <Tbody>
            {isLoading ? (
              <tr><td colSpan={6}><PageSpinner /></td></tr>
            ) : paidOrders.length === 0 ? (
              <EmptyRow cols={6} message="No payments found" icon={<CreditCard size={28} />} />
            ) : (
              paidOrders.map((order) => (
                <Tr key={order.id}>
                  <Td>
                    <span className="font-mono text-xs text-zinc-500">#{order.id.slice(0, 8)}</span>
                  </Td>
                  <Td className="font-semibold tabular-nums">{formatMoney(order.total_amount, order.currency)}</Td>
                  <Td className="text-zinc-500">{order.currency}</Td>
                  <Td><Badge status={order.status} /></Td>
                  <Td className="text-xs text-zinc-500">{formatDate(order.created_at)}</Td>
                  <Td>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={async () => {
                        setViewError('');
                        try {
                          const res = await paymentsApi.getByOrder(order.id);
                          setSelected(res.data);
                        } catch {
                          setViewError('Could not load payment for order #' + order.id.slice(0, 8));
                        }
                      }}
                    >
                      <CreditCard size={13} /> View
                    </Button>
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Card>

      {viewError && (
        <div className="mt-3 rounded-lg bg-red-500/10 px-4 py-3 text-sm text-red-400 flex items-center justify-between">
          <span>{viewError}</span>
          <button onClick={() => setViewError('')} className="text-red-400/60 hover:text-red-400 text-xs ml-4">Dismiss</button>
        </div>
      )}

      <Modal
        open={!!selected}
        onClose={() => setSelected(null)}
        title="Payment Details"
      >
        {selected && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              {[
                ['Payment ID', <span className="font-mono text-xs">{selected.id.slice(0, 16)}...</span>],
                ['Order ID', <span className="font-mono text-xs">{selected.order_id.slice(0, 16)}...</span>],
                ['Amount', <span className="font-semibold tabular-nums">{formatMoney(selected.amount, selected.currency)}</span>],
                ['Status', <Badge status={selected.status} />],
                ['Method', <span className="capitalize">{selected.method}</span>],
                ['Transaction ID', selected.transaction_id || '--'],
              ].map(([k, v]) => (
                <div key={String(k)} className="bg-white/5 rounded-lg ring-1 ring-white/[0.06] px-4 py-3">
                  <p className="text-[10px] text-zinc-500 mb-1 uppercase tracking-wider font-medium">{String(k)}</p>
                  <div className="text-sm text-zinc-300">{v}</div>
                </div>
              ))}
            </div>
            <div className="bg-white/[0.03] rounded-lg ring-1 ring-white/[0.06] px-4 py-3 text-xs text-zinc-500">
              <p>Created: {formatDate(selected.created_at)}</p>
              <p className="mt-0.5">Updated: {formatDate(selected.updated_at)}</p>
            </div>
            {selected.status === 'succeeded' && (
              <Button
                variant="danger"
                className="w-full"
                loading={refundMutation.isPending}
                onClick={() => refundMutation.mutate(selected.id)}
              >
                Process Refund
              </Button>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
