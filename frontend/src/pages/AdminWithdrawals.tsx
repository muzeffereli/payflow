import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { CheckCircle, XCircle, ShieldCheck } from 'lucide-react';
import { toast } from 'sonner';
import { withdrawalsApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Input } from '../components/ui/Input';
import { Table, Thead, Th, Tbody, Tr, Td, EmptyRow } from '../components/ui/Table';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { formatMoney, formatDate } from '../lib/utils';
import type { Withdrawal } from '../lib/types';

export function AdminWithdrawals() {
  const qc = useQueryClient();
  const [rejectId, setRejectId] = useState<string | null>(null);
  const [reason, setReason] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['admin-withdrawals'],
    queryFn: () => withdrawalsApi.pendingList(),
  });

  const approveMutation = useMutation({
    mutationFn: (id: string) => withdrawalsApi.approve(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-withdrawals'] }); toast.success('Withdrawal approved'); },
  });

  const rejectMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      withdrawalsApi.reject(id, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-withdrawals'] });
      setRejectId(null);
      setReason('');
      toast.success('Withdrawal rejected');
    },
  });

  const withdrawals: Withdrawal[] = data?.data?.withdrawals ?? [];

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center gap-4">
        <div className="w-11 h-11 rounded-xl bg-red-500/10 ring-1 ring-red-500/10 flex items-center justify-center">
          <ShieldCheck size={20} className="text-red-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Admin &mdash; Withdrawals</h1>
          <p className="text-zinc-500 text-sm">Review and approve seller withdrawal requests</p>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4 mb-8 stagger-children">
        <div className="bg-amber-500/5 ring-1 ring-amber-500/10 rounded-2xl p-5">
          <p className="text-[10px] text-amber-500 mb-1 font-semibold uppercase tracking-wider">Pending</p>
          <p className="text-2xl font-bold text-amber-400 tabular-nums">{withdrawals.length}</p>
        </div>
        <div className="bg-indigo-500/5 ring-1 ring-indigo-500/10 rounded-2xl p-5">
          <p className="text-[10px] text-indigo-400 mb-1 font-semibold uppercase tracking-wider">Total Requested</p>
          <p className="text-2xl font-bold text-indigo-300 tabular-nums">
            {formatMoney(withdrawals.reduce((s, w) => s + w.amount, 0))}
          </p>
        </div>
        <div className="bg-white/[0.03] ring-1 ring-white/[0.06] rounded-2xl p-5">
          <p className="text-[10px] text-zinc-500 mb-1 font-semibold uppercase tracking-wider">Avg Amount</p>
          <p className="text-2xl font-bold text-zinc-200 tabular-nums">
            {withdrawals.length > 0
              ? formatMoney(
                  Math.round(withdrawals.reduce((s, w) => s + w.amount, 0) / withdrawals.length),
                )
              : '--'}
          </p>
        </div>
      </div>

      <Card>
        <Table>
          <Thead>
            <tr>
              <Th>ID</Th>
              <Th>User</Th>
              <Th>Amount</Th>
              <Th>Method</Th>
              <Th>Status</Th>
              <Th>Requested</Th>
              <Th>Actions</Th>
            </tr>
          </Thead>
          <Tbody>
            {isLoading ? (
              <tr><td colSpan={7}><PageSpinner /></td></tr>
            ) : withdrawals.length === 0 ? (
              <EmptyRow cols={7} message="No pending withdrawals" icon={<ShieldCheck size={28} />} />
            ) : (
              withdrawals.map((w) => (
                <Tr key={w.id}>
                  <Td><span className="font-mono text-xs text-zinc-500">#{w.id.slice(0, 8)}</span></Td>
                  <Td><span className="font-mono text-xs text-zinc-500">{w.user_id.slice(0, 12)}...</span></Td>
                  <Td className="font-semibold text-zinc-100 tabular-nums">{formatMoney(w.amount, w.currency)}</Td>
                  <Td className="text-zinc-500 capitalize">{w.method.replace(/_/g, ' ')}</Td>
                  <Td><Badge status={w.status} /></Td>
                  <Td className="text-xs text-zinc-500">{formatDate(w.created_at)}</Td>
                  <Td>
                    {w.status === 'pending' && (
                      <div className="flex gap-1.5">
                        <Button
                          size="sm"
                          variant="success"
                          loading={approveMutation.isPending}
                          onClick={() => approveMutation.mutate(w.id)}
                        >
                          <CheckCircle size={13} /> Approve
                        </Button>
                        <Button
                          size="sm"
                          variant="danger"
                          onClick={() => { setRejectId(w.id); setReason(''); }}
                        >
                          <XCircle size={13} /> Reject
                        </Button>
                      </div>
                    )}
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Card>

      <Modal
        open={!!rejectId}
        onClose={() => setRejectId(null)}
        title="Reject Withdrawal"
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-zinc-500">Provide a reason for rejecting this withdrawal request.</p>
          <Input
            label="Reason"
            placeholder="e.g. Insufficient documentation"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            required
          />
          <div className="flex gap-2 justify-end pt-2">
            <Button variant="ghost" onClick={() => setRejectId(null)}>Cancel</Button>
            <Button
              variant="danger"
              loading={rejectMutation.isPending}
              disabled={!reason.trim()}
              onClick={() => rejectMutation.mutate({ id: rejectId!, reason })}
            >
              <XCircle size={14} /> Reject
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
