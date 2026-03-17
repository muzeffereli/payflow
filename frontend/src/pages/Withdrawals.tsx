import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, ArrowDownToLine } from 'lucide-react';
import { getApiErrorMessage, withdrawalsApi, storesApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Input, Select } from '../components/ui/Input';
import { Table, Thead, Th, Tbody, Tr, Td, EmptyRow } from '../components/ui/Table';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { formatMoney, formatDate } from '../lib/utils';
import type { Withdrawal, Store } from '../lib/types';

const emptyForm = { amount: '', currency: 'USD', method: 'bank_transfer' };

export function Withdrawals() {
  const qc = useQueryClient();
  const [showModal, setShowModal] = useState(false);
  const [form, setForm] = useState({ ...emptyForm });
  const [formError, setFormError] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['my-withdrawals'],
    queryFn: () => withdrawalsApi.myList(),
  });

  const { data: storeData } = useQuery({
    queryKey: ['my-store'],
    queryFn: () => storesApi.getMe(),
    retry: false,
  });

  const requestMutation = useMutation({
    mutationFn: (d: { store_id: string; amount: number; currency: string; method: string }) =>
      withdrawalsApi.request(d),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['my-withdrawals'] });
      setShowModal(false);
      setForm({ ...emptyForm });
      setFormError('');
    },
    onError: (error: unknown) => setFormError(getApiErrorMessage(error, 'Request failed')),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');

    const amount = parseFloat(form.amount);
    if (isNaN(amount) || amount <= 0) { setFormError('Amount must be greater than 0'); return; }
    if (!myStore) { setFormError('Create a store before requesting a withdrawal'); return; }

    requestMutation.mutate({
      store_id: myStore.id,
      amount: Math.round(amount * 100),
      currency: form.currency,
      method: form.method,
    });
  };

  const withdrawals: Withdrawal[] = data?.data?.withdrawals ?? [];
  const myStore: Store | null = storeData?.data ?? null;

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Withdrawals</h1>
          <p className="text-zinc-500 text-sm mt-1">Request payouts from your wallet</p>
        </div>
        <Button onClick={() => setShowModal(true)} disabled={!myStore}>
          <Plus size={15} /> Request Withdrawal
        </Button>
      </div>

      {!myStore && (
        <p className="mb-4 rounded-lg bg-amber-500/10 px-4 py-3 text-sm text-amber-300">
          Create and activate your store before requesting a withdrawal.
        </p>
      )}

      <Card>
        <Table>
          <Thead>
            <tr>
              <Th>ID</Th>
              <Th>Amount</Th>
              <Th>Method</Th>
              <Th>Status</Th>
              <Th>Notes</Th>
              <Th>Requested</Th>
            </tr>
          </Thead>
          <Tbody>
            {isLoading ? (
              <tr><td colSpan={6}><PageSpinner /></td></tr>
            ) : withdrawals.length === 0 ? (
              <EmptyRow cols={6} message="No withdrawals yet" icon={<ArrowDownToLine size={28} />} />
            ) : (
              withdrawals.map((w) => (
                <Tr key={w.id}>
                  <Td><span className="font-mono text-xs text-zinc-500">#{w.id.slice(0, 8)}</span></Td>
                  <Td className="font-semibold tabular-nums">{formatMoney(w.amount, w.currency)}</Td>
                  <Td className="text-zinc-500 capitalize">{w.method.replace(/_/g, ' ')}</Td>
                  <Td><Badge status={w.status} /></Td>
                  <Td className="text-zinc-500 text-xs max-w-[200px] truncate">{w.notes || '--'}</Td>
                  <Td className="text-xs text-zinc-500">{formatDate(w.created_at)}</Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Card>

      <Modal open={showModal} onClose={() => setShowModal(false)} title="Request Withdrawal">
        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="Store"
            value={myStore?.name ?? 'No store available'}
            disabled
          />
          <Input
            label="Amount (USD)"
            type="number"
            step="0.01"
            min="0.01"
            placeholder="0.00"
            value={form.amount}
            onChange={(e) => setForm({ ...form, amount: e.target.value })}
            required
          />
          <Select
            label="Method"
            value={form.method}
            onChange={(e) => setForm({ ...form, method: e.target.value })}
          >
            <option value="bank_transfer">Bank Transfer</option>
            <option value="paypal">PayPal</option>
            <option value="crypto">Crypto</option>
          </Select>
          {formError && (
            <p className="text-sm text-red-400 bg-red-500/10 rounded-lg px-4 py-3">{formError}</p>
          )}
          <div className="flex gap-2 justify-end pt-3">
            <Button type="button" variant="ghost" onClick={() => setShowModal(false)}>Cancel</Button>
            <Button type="submit" loading={requestMutation.isPending} disabled={!myStore}>
              <ArrowDownToLine size={14} /> Request
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
