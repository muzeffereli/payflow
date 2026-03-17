import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, Tags, X } from 'lucide-react';
import { toast } from 'sonner';
import { attributesApi } from '../lib/api';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Input } from '../components/ui/Input';
import { Table, Thead, Th, Tbody, Tr, Td, EmptyRow } from '../components/ui/Table';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { formatDate } from '../lib/utils';
import type { GlobalAttribute } from '../lib/types';

export function AdminAttributes() {
  const qc = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<GlobalAttribute | null>(null);
  const [name, setName] = useState('');
  const [valueInput, setValueInput] = useState('');
  const [values, setValues] = useState<string[]>([]);
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['global-attributes'],
    queryFn: () => attributesApi.list(),
  });

  const createMutation = useMutation({
    mutationFn: (data: { name: string; values: string[] }) => attributesApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['global-attributes'] });
      closeModal();
      toast.success('Attribute created');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name?: string; values?: string[] } }) =>
      attributesApi.update(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['global-attributes'] });
      closeModal();
      toast.success('Attribute updated');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => attributesApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['global-attributes'] });
      setDeleteId(null);
      toast.success('Attribute deleted');
    },
  });

  const attributes: GlobalAttribute[] = data?.data?.attributes ?? [];

  function openCreate() {
    setEditing(null);
    setName('');
    setValues([]);
    setValueInput('');
    setModalOpen(true);
  }

  function openEdit(attr: GlobalAttribute) {
    setEditing(attr);
    setName(attr.name);
    setValues([...attr.values]);
    setValueInput('');
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditing(null);
    setName('');
    setValues([]);
    setValueInput('');
  }

  function addValue() {
    const v = valueInput.trim();
    if (v && !values.includes(v)) {
      setValues([...values, v]);
    }
    setValueInput('');
  }

  function removeValue(v: string) {
    setValues(values.filter((x) => x !== v));
  }

  function handleSubmit() {
    if (!name.trim() || values.length === 0) return;
    if (editing) {
      updateMutation.mutate({ id: editing.id, data: { name: name.trim(), values } });
    } else {
      createMutation.mutate({ name: name.trim(), values });
    }
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="w-11 h-11 rounded-xl bg-violet-500/10 ring-1 ring-violet-500/10 flex items-center justify-center">
            <Tags size={20} className="text-violet-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Global Attributes</h1>
            <p className="text-zinc-500 text-sm">
              Define attribute dimensions (e.g. Color, Size) that sellers can use on their products
            </p>
          </div>
        </div>
        <Button onClick={openCreate}>
          <Plus size={14} /> New Attribute
        </Button>
      </div>

      <Card>
        <Table>
          <Thead>
            <tr>
              <Th>Name</Th>
              <Th>Values</Th>
              <Th>Created</Th>
              <Th>Actions</Th>
            </tr>
          </Thead>
          <Tbody>
            {isLoading ? (
              <tr>
                <td colSpan={4}>
                  <PageSpinner />
                </td>
              </tr>
            ) : attributes.length === 0 ? (
              <EmptyRow cols={4} message="No attributes defined yet" icon={<Tags size={28} />} />
            ) : (
              attributes.map((attr) => (
                <Tr key={attr.id}>
                  <Td className="font-semibold text-zinc-100">{attr.name}</Td>
                  <Td>
                    <div className="flex flex-wrap gap-1">
                      {attr.values.map((v) => (
                        <span
                          key={v}
                          className="inline-flex items-center px-2 py-0.5 rounded-md bg-white/[0.06] text-xs text-zinc-300"
                        >
                          {v}
                        </span>
                      ))}
                    </div>
                  </Td>
                  <Td className="text-xs text-zinc-500">{formatDate(attr.created_at)}</Td>
                  <Td>
                    <div className="flex gap-1.5">
                      <Button size="sm" variant="ghost" onClick={() => openEdit(attr)}>
                        <Pencil size={13} /> Edit
                      </Button>
                      <Button size="sm" variant="danger" onClick={() => setDeleteId(attr.id)}>
                        <Trash2 size={13} /> Delete
                      </Button>
                    </div>
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Card>

      <Modal
        open={modalOpen}
        onClose={closeModal}
        title={editing ? 'Edit Attribute' : 'New Attribute'}
        size="sm"
      >
        <div className="space-y-4">
          <Input
            label="Attribute Name"
            placeholder="e.g. Color, Size, Material"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
          <div>
            <label className="block text-[13px] font-medium text-zinc-300 mb-1.5">Values</label>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Input
                placeholder="Add a value and press Enter"
                value={valueInput}
                onChange={(e) => setValueInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    addValue();
                  }
                }}
              />
              <Button variant="ghost" onClick={addValue} disabled={!valueInput.trim()} className="sm:self-end">
                <Plus size={14} />
              </Button>
            </div>
            {values.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mt-3">
                {values.map((v) => (
                    <span
                      key={v}
                      className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg bg-indigo-500/10 ring-1 ring-indigo-500/20 text-xs text-indigo-300"
                    >
                      {v}
                      <button
                        type="button"
                        onClick={() => removeValue(v)}
                        className="hover:text-red-400 transition-colors"
                      >
                      <X size={12} />
                    </button>
                  </span>
                ))}
              </div>
            )}
          </div>
          <div className="flex flex-col-reverse justify-end gap-2 pt-2 sm:flex-row">
            <Button variant="ghost" onClick={closeModal}>
              Cancel
            </Button>
            <Button
              loading={isSaving}
              disabled={!name.trim() || values.length === 0}
              onClick={handleSubmit}
            >
              {editing ? 'Save Changes' : 'Create Attribute'}
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        open={!!deleteId}
        onClose={() => setDeleteId(null)}
        title="Delete Attribute"
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-zinc-400">
            Are you sure you want to delete this attribute? Products using it will lose this attribute
            definition.
          </p>
          <div className="flex flex-col-reverse justify-end gap-2 pt-2 sm:flex-row">
            <Button variant="ghost" onClick={() => setDeleteId(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              loading={deleteMutation.isPending}
              onClick={() => deleteMutation.mutate(deleteId!)}
            >
              <Trash2 size={14} /> Delete
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
