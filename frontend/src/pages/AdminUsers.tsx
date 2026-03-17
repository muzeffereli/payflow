import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Users, ChevronLeft, ChevronRight } from 'lucide-react';
import { toast } from 'sonner';
import { adminApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Select } from '../components/ui/Input';
import { Table, Thead, Th, Tbody, Tr, Td, EmptyRow } from '../components/ui/Table';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { formatDate } from '../lib/utils';

interface UserItem {
  id: string;
  email: string;
  name: string;
  role: string;
  created_at: string;
}

const LIMIT = 20;

export function AdminUsers() {
  const qc = useQueryClient();
  const [offset, setOffset] = useState(0);
  const [roleModal, setRoleModal] = useState<UserItem | null>(null);
  const [newRole, setNewRole] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['admin-users', offset],
    queryFn: () => adminApi.listUsers(LIMIT, offset),
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ id, role }: { id: string; role: string }) => adminApi.updateRole(id, role),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-users'] });
      setRoleModal(null);
      toast.success('Role updated');
    },
  });

  const users: UserItem[] = data?.data?.users ?? [];
  const total: number = data?.data?.total ?? 0;
  const hasNext = offset + LIMIT < total;
  const hasPrev = offset > 0;

  function openRoleModal(user: UserItem) {
    setRoleModal(user);
    setNewRole(user.role);
  }

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center gap-4">
        <div className="w-11 h-11 rounded-xl bg-blue-500/10 ring-1 ring-blue-500/10 flex items-center justify-center">
          <Users size={20} className="text-blue-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">User Management</h1>
          <p className="text-zinc-500 text-sm">
            {total} total user{total !== 1 ? 's' : ''}
          </p>
        </div>
      </div>

      <Card>
        <Table>
          <Thead>
            <tr>
              <Th>Name</Th>
              <Th>Email</Th>
              <Th>Role</Th>
              <Th>Joined</Th>
              <Th>Actions</Th>
            </tr>
          </Thead>
          <Tbody>
            {isLoading ? (
              <tr>
                <td colSpan={5}>
                  <PageSpinner />
                </td>
              </tr>
            ) : users.length === 0 ? (
              <EmptyRow cols={5} message="No users found" icon={<Users size={28} />} />
            ) : (
              users.map((u) => (
                <Tr key={u.id}>
                  <Td>
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-lg bg-indigo-600/20 flex items-center justify-center text-indigo-400 text-xs font-semibold shrink-0">
                        {u.name.charAt(0).toUpperCase()}
                      </div>
                      <span className="font-medium text-zinc-200">{u.name}</span>
                    </div>
                  </Td>
                  <Td className="text-zinc-400">{u.email}</Td>
                  <Td>
                    <Badge status={u.role} />
                  </Td>
                  <Td className="text-xs text-zinc-500">{formatDate(u.created_at)}</Td>
                  <Td>
                    <Button size="sm" variant="ghost" onClick={() => openRoleModal(u)}>
                      Change Role
                    </Button>
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>

        {total > LIMIT && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-white/[0.06]">
            <p className="text-xs text-zinc-500">
              Showing {offset + 1}â€“{Math.min(offset + LIMIT, total)} of {total}
            </p>
            <div className="flex gap-1.5">
              <Button size="sm" variant="ghost" disabled={!hasPrev} onClick={() => setOffset(offset - LIMIT)}>
                <ChevronLeft size={14} /> Prev
              </Button>
              <Button size="sm" variant="ghost" disabled={!hasNext} onClick={() => setOffset(offset + LIMIT)}>
                Next <ChevronRight size={14} />
              </Button>
            </div>
          </div>
        )}
      </Card>

      <Modal open={!!roleModal} onClose={() => setRoleModal(null)} title="Change User Role" size="sm">
        {roleModal && (
          <div className="space-y-4">
            <p className="text-sm text-zinc-400">
              Changing role for <strong className="text-zinc-200">{roleModal.name}</strong> ({roleModal.email})
            </p>
            <Select label="Role" value={newRole} onChange={(e) => setNewRole(e.target.value)}>
              <option value="customer">Customer</option>
              <option value="seller">Seller</option>
              <option value="admin">Admin</option>
            </Select>
            <div className="flex gap-2 justify-end pt-2">
              <Button variant="ghost" onClick={() => setRoleModal(null)}>
                Cancel
              </Button>
              <Button
                loading={updateRoleMutation.isPending}
                disabled={newRole === roleModal.role}
                onClick={() => updateRoleMutation.mutate({ id: roleModal.id, role: newRole })}
              >
                Update Role
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
