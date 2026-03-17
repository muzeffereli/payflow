import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { RefreshCw, Trash2, Package } from 'lucide-react';
import { ordersApi, paymentsApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Table, Thead, Th, Tbody, Tr, Td, EmptyRow } from '../components/ui/Table';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { formatMoney, formatDate } from '../lib/utils';
import { useUser } from '../lib/store';
import type { Order, Payment } from '../lib/types';

export function Orders() {
  const qc = useQueryClient();
  const user = useUser();
  const [selected, setSelected] = useState<Order | null>(null);
  const [payment, setPayment] = useState<Payment | null>(null);
  const ordersQueryKey = ['orders', user?.id];

  const { data, isLoading } = useQuery({
    queryKey: ordersQueryKey,
    queryFn: () => ordersApi.list(50),
    enabled: !!user?.id,
  });

  const cancelMutation = useMutation({
    mutationFn: (id: string) => ordersApi.cancel(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ordersQueryKey }),
  });

  const refundMutation = useMutation({
    mutationFn: (paymentId: string) => paymentsApi.refund(paymentId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ordersQueryKey });
      setPayment(null);
      setSelected(null);
    },
  });

  const [paymentError, setPaymentError] = useState('');

  const openOrder = async (order: Order) => {
    setSelected(order);
    setPayment(null);
    setPaymentError('');
    try {
      const res = await paymentsApi.getByOrder(order.id);
      setPayment(res.data);
    } catch {
      setPaymentError('Payment info unavailable');
    }
  };

  const orders: Order[] = data?.data?.orders ?? [];

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Orders</h1>
          <p className="text-zinc-500 text-sm mt-1">{orders.length} total orders</p>
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
              <Th>Items</Th>
              <Th>Amount</Th>
              <Th>Status</Th>
              <Th>Created</Th>
              <Th />
            </tr>
          </Thead>
          <Tbody>
            {isLoading ? (
              <tr><td colSpan={6}><PageSpinner /></td></tr>
            ) : orders.length === 0 ? (
              <EmptyRow cols={6} message="No orders found" icon={<Package size={28} />} />
            ) : (
              orders.map((order) => (
                <Tr key={order.id} onClick={() => openOrder(order)}>
                  <Td>
                    <span className="font-mono text-xs text-zinc-500">#{order.id.slice(0, 8)}</span>
                  </Td>
                  <Td>{order.items?.length ?? 0} item(s)</Td>
                  <Td className="font-semibold tabular-nums">{formatMoney(order.total_amount, order.currency)}</Td>
                  <Td><Badge status={order.status} /></Td>
                  <Td className="text-zinc-500 text-xs">{formatDate(order.created_at)}</Td>
                  <Td>
                    {order.status === 'pending' && (
                      <Button
                        variant="danger"
                        size="sm"
                        loading={cancelMutation.isPending}
                        onClick={(e) => {
                          e.stopPropagation();
                          cancelMutation.mutate(order.id);
                        }}
                      >
                        <Trash2 size={13} /> Cancel
                      </Button>
                    )}
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Card>

      <Modal
        open={!!selected}
        onClose={() => { setSelected(null); setPayment(null); }}
        title={`Order #${selected?.id.slice(0, 8)}`}
        size="lg"
      >
        {selected && (
          <div className="space-y-5">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              {[
                ['Status', <Badge status={selected.status} />],
                ['Total', <span className="font-semibold text-zinc-100 tabular-nums">{formatMoney(selected.total_amount, selected.currency)}</span>],
                ['Currency', selected.currency],
                ['Created', formatDate(selected.created_at)],
              ].map(([k, v]) => (
                <div key={String(k)} className="bg-white/5 rounded-lg ring-1 ring-white/[0.06] px-4 py-3">
                  <p className="text-[10px] text-zinc-500 mb-1 uppercase tracking-wider font-medium">{String(k)}</p>
                  <div className="text-sm text-zinc-300">{v}</div>
                </div>
              ))}
            </div>

            {selected.items?.length > 0 && (
              <div>
                <p className="text-[10px] text-zinc-500 mb-2.5 font-semibold uppercase tracking-wider">Items</p>
                <div className="space-y-2">
                  {selected.items.map((item, i) => (
                    <div key={i} className="flex flex-col gap-3 rounded-lg bg-white/[0.03] px-4 py-3 ring-1 ring-white/[0.06] sm:flex-row sm:items-center sm:justify-between">
                      <div>
                        <p className="text-sm text-zinc-200 font-mono">{item.product_id.slice(0, 12)}...</p>
                        {item.variant_label && (
                          <p className="text-xs text-zinc-500 mt-0.5">{item.variant_label}</p>
                        )}
                        {item.variant_sku && (
                          <p className="text-xs text-zinc-500 mt-0.5">SKU: {item.variant_sku}</p>
                        )}
                        <p className="text-xs text-zinc-500 mt-0.5">Qty: {item.quantity}</p>
                      </div>
                      <p className="text-sm font-semibold text-zinc-200 tabular-nums sm:text-right">{formatMoney(item.price * item.quantity)}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {payment && (
              <div className="bg-white/[0.03] rounded-lg ring-1 ring-white/[0.06] px-4 py-3">
                <p className="text-[10px] text-zinc-500 mb-2 font-semibold uppercase tracking-wider">Payment</p>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <p className="text-sm text-zinc-200">Method: {payment.method}</p>
                    {payment.transaction_id && (
                      <p className="text-xs text-zinc-500 font-mono mt-0.5">{payment.transaction_id}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-2 sm:justify-end">
                    <Badge status={payment.status} />
                    {payment.status === 'succeeded' && (
                      <Button
                        variant="danger"
                        size="sm"
                        loading={refundMutation.isPending}
                        onClick={() => refundMutation.mutate(payment.id)}
                      >
                        Refund
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            )}
            {paymentError && !payment && (
              <p className="text-xs text-zinc-500 bg-white/[0.03] rounded-lg ring-1 ring-white/[0.06] px-4 py-3 text-center">
                {paymentError}
              </p>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
