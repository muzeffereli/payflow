import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Store, CheckCircle, Ban, RefreshCw } from 'lucide-react';
import { toast } from 'sonner';
import { storesApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Table, Thead, Th, Tbody, Tr, Td, EmptyRow } from '../components/ui/Table';
import { PageSpinner } from '../components/ui/Spinner';
import { formatDate } from '../lib/utils';
import type { Store as StoreType } from '../lib/types';

export function AdminStores() {
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['admin-stores'],
    queryFn: () => storesApi.list(100, 0),
  });

  const approveMut = useMutation({
    mutationFn: (id: string) => storesApi.approve(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-stores'] }); toast.success('Store approved'); },
  });
  const suspendMut = useMutation({
    mutationFn: (id: string) => storesApi.suspend(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-stores'] }); toast.success('Store suspended'); },
  });
  const reactivateMut = useMutation({
    mutationFn: (id: string) => storesApi.reactivate(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['admin-stores'] }); toast.success('Store reactivated'); },
  });

  const stores: StoreType[] = data?.data?.stores ?? [];

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center gap-4">
        <div className="w-11 h-11 rounded-xl bg-emerald-500/10 ring-1 ring-emerald-500/10 flex items-center justify-center">
          <Store size={20} className="text-emerald-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Store Management</h1>
          <p className="text-zinc-500 text-sm">{stores.length} stores</p>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4 mb-8 stagger-children">
        <div className="bg-amber-500/5 ring-1 ring-amber-500/10 rounded-2xl p-5">
          <p className="text-[10px] text-amber-500 mb-1 font-semibold uppercase tracking-wider">Pending</p>
          <p className="text-2xl font-bold text-amber-400 tabular-nums">
            {stores.filter((s) => s.status === 'pending').length}
          </p>
        </div>
        <div className="bg-emerald-500/5 ring-1 ring-emerald-500/10 rounded-2xl p-5">
          <p className="text-[10px] text-emerald-500 mb-1 font-semibold uppercase tracking-wider">Active</p>
          <p className="text-2xl font-bold text-emerald-400 tabular-nums">
            {stores.filter((s) => s.status === 'active').length}
          </p>
        </div>
        <div className="bg-red-500/5 ring-1 ring-red-500/10 rounded-2xl p-5">
          <p className="text-[10px] text-red-500 mb-1 font-semibold uppercase tracking-wider">Suspended</p>
          <p className="text-2xl font-bold text-red-400 tabular-nums">
            {stores.filter((s) => s.status === 'suspended').length}
          </p>
        </div>
      </div>

      <Card>
        <Table>
          <Thead>
            <tr>
              <Th>Store</Th>
              <Th>Owner</Th>
              <Th>Status</Th>
              <Th>Commission</Th>
              <Th>Created</Th>
              <Th>Actions</Th>
            </tr>
          </Thead>
          <Tbody>
            {isLoading ? (
              <tr>
                <td colSpan={6}>
                  <PageSpinner />
                </td>
              </tr>
            ) : stores.length === 0 ? (
              <EmptyRow cols={6} message="No stores yet" icon={<Store size={28} />} />
            ) : (
              stores.map((s) => (
                <Tr key={s.id}>
                  <Td>
                    <div>
                      <p className="font-medium text-zinc-200">{s.name}</p>
                      {s.description && (
                        <p className="text-xs text-zinc-600 truncate max-w-[200px]">{s.description}</p>
                      )}
                    </div>
                  </Td>
                  <Td>
                    <span className="font-mono text-xs text-zinc-500">{s.owner_id.slice(0, 12)}...</span>
                  </Td>
                  <Td>
                    <Badge status={s.status} />
                  </Td>
                  <Td className="tabular-nums text-zinc-300">{s.commission}%</Td>
                  <Td className="text-xs text-zinc-500">{formatDate(s.created_at)}</Td>
                  <Td>
                    <div className="flex gap-1.5">
                      {s.status === 'pending' && (
                        <Button
                          size="sm"
                          variant="success"
                          loading={approveMut.isPending}
                          onClick={() => approveMut.mutate(s.id)}
                        >
                          <CheckCircle size={13} /> Approve
                        </Button>
                      )}
                      {s.status === 'active' && (
                        <Button
                          size="sm"
                          variant="danger"
                          loading={suspendMut.isPending}
                          onClick={() => suspendMut.mutate(s.id)}
                        >
                          <Ban size={13} /> Suspend
                        </Button>
                      )}
                      {s.status === 'suspended' && (
                        <Button
                          size="sm"
                          variant="ghost"
                          loading={reactivateMut.isPending}
                          onClick={() => reactivateMut.mutate(s.id)}
                        >
                          <RefreshCw size={13} /> Reactivate
                        </Button>
                      )}
                    </div>
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Card>
    </div>
  );
}
