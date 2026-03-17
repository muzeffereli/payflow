import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Store as StoreIcon, Pencil } from 'lucide-react';
import { storesApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card, CardBody } from '../components/ui/Card';
import { Input, Textarea } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { formatDate } from '../lib/utils';
import type { Store } from '../lib/types';

const emptyForm = { name: '', description: '' };

export function Stores() {
  const qc = useQueryClient();
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<Store | null>(null);
  const [form, setForm] = useState({ ...emptyForm });

  const { data, isLoading } = useQuery({
    queryKey: ['my-store'],
    queryFn: () => storesApi.getMe(),
    retry: false,
  });

  const createMutation = useMutation({
    mutationFn: (data: object) => storesApi.create(data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['my-store'] }); closeModal(); },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: object }) => storesApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['my-store'] }); closeModal(); },
  });

  const openCreate = () => { setEditing(null); setForm({ ...emptyForm }); setShowModal(true); };
  const openEdit = (s: Store) => {
    setEditing(s);
    setForm({ name: s.name, description: s.description });
    setShowModal(true);
  };
  const closeModal = () => { setShowModal(false); setEditing(null); };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (editing) {
      updateMutation.mutate({ id: editing.id, data: form });
    } else {
      createMutation.mutate(form);
    }
  };

  if (isLoading) return <PageSpinner />;

  const store: Store | null = data?.data ?? null;
  const stores: Store[] = store ? [store] : [];

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">My Stores</h1>
          <p className="text-zinc-500 text-sm mt-1">{stores.length} {stores.length === 1 ? 'store' : 'stores'}</p>
        </div>
        <Button onClick={openCreate}>
          <Plus size={15} /> Create Store
        </Button>
      </div>

      {stores.length === 0 ? (
        <Card>
          <CardBody className="py-20 text-center">
            <div className="w-16 h-16 rounded-2xl bg-indigo-500/10 flex items-center justify-center mx-auto mb-5">
              <StoreIcon size={28} className="text-indigo-400" />
            </div>
            <p className="text-zinc-300 font-medium mb-1">No stores yet</p>
            <p className="text-zinc-500 text-sm mb-8">Create your first store to start selling</p>
            <Button onClick={openCreate} size="lg">
              <Plus size={15} /> Create Store
            </Button>
          </CardBody>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 stagger-children">
          {stores.map((store) => (
            <Card key={store.id} hover>
              <CardBody>
                <div className="flex items-start justify-between mb-4">
                  <div className="w-11 h-11 rounded-xl bg-gradient-to-br from-indigo-500/10 to-violet-500/10 ring-1 ring-indigo-500/10 flex items-center justify-center">
                    <StoreIcon size={18} className="text-indigo-400" />
                  </div>
                  <div className="flex items-center gap-1.5">
                    <Badge status={store.status} />
                    <Button variant="ghost" size="sm" onClick={() => openEdit(store)}>
                      <Pencil size={13} />
                    </Button>
                  </div>
                </div>
                <h3 className="font-semibold text-zinc-100 mb-1">{store.name}</h3>
                {store.description && (
                  <p className="text-sm text-zinc-500 mb-4 line-clamp-2">{store.description}</p>
                )}
                <div className="flex items-center justify-between text-xs text-zinc-600 mt-4 pt-4 border-t border-white/[0.06]">
                  <span>Commission: {store.commission}%</span>
                  <span>{formatDate(store.created_at)}</span>
                </div>
              </CardBody>
            </Card>
          ))}
        </div>
      )}

      <Modal open={showModal} onClose={closeModal} title={editing ? 'Edit Store' : 'Create Store'}>
        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="Store Name"
            placeholder="My Awesome Store"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
          />
          <Textarea
            label="Description"
            rows={3}
            placeholder="What does your store sell?"
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
          />
          <div className="flex flex-col-reverse justify-end gap-2 pt-3 sm:flex-row">
            <Button type="button" variant="ghost" onClick={closeModal}>Cancel</Button>
            <Button
              type="submit"
              loading={createMutation.isPending || updateMutation.isPending}
            >
              {editing ? 'Save Changes' : 'Create Store'}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
