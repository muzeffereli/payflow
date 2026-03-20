import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FolderTree, Pencil, Plus, Tags, Trash2, X } from 'lucide-react';
import { toast } from 'sonner';
import { attributesApi, categoriesApi, getApiErrorMessage } from '../lib/api';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Input, Select } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { EmptyRow, Table, Tbody, Td, Th, Thead, Tr } from '../components/ui/Table';
import { PageSpinner } from '../components/ui/Spinner';
import { formatDate } from '../lib/utils';
import type { Category, GlobalAttribute, Subcategory } from '../lib/types';

export function AdminAttributes() {
  const qc = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<GlobalAttribute | null>(null);
  const [formCategoryId, setFormCategoryId] = useState('');
  const [formSubcategoryId, setFormSubcategoryId] = useState('');
  const [name, setName] = useState('');
  const [valueInput, setValueInput] = useState('');
  const [values, setValues] = useState<string[]>([]);
  const [activeSubcategoryId, setActiveSubcategoryId] = useState('all');
  const [deleteTarget, setDeleteTarget] = useState<GlobalAttribute | null>(null);

  const { data: categoriesData } = useQuery({
    queryKey: ['categories'],
    queryFn: () => categoriesApi.list(),
  });

  const { data: formSubcategoriesData } = useQuery({
    queryKey: ['subcategories', formCategoryId],
    queryFn: () => categoriesApi.listSubcategories(formCategoryId),
    enabled: !!formCategoryId,
  });

  const { data, isLoading } = useQuery({
    queryKey: ['category-attributes'],
    queryFn: () => attributesApi.list(),
  });

  const categoryOptions: Category[] = categoriesData?.data?.categories ?? [];
  const formSubcategoryOptions: Subcategory[] = formSubcategoriesData?.data?.subcategories ?? [];
  const attributes: GlobalAttribute[] = data?.data?.attributes ?? [];

  // Collect subcategories that actually have attributes for the filter bar
  const subcategoryIdsWithAttrs = useMemo(
    () => ['all', ...Array.from(new Set(attributes.map((a) => a.subcategory_id).filter(Boolean)))],
    [attributes],
  );

  const filteredAttributes =
    activeSubcategoryId === 'all'
      ? attributes
      : attributes.filter((a) => a.subcategory_id === activeSubcategoryId);

  const createMutation = useMutation({
    mutationFn: (payload: { subcategory_id: string; name: string; values: string[] }) =>
      attributesApi.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['category-attributes'] });
      closeModal();
      toast.success('Attribute created');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to create attribute'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: { subcategory_id?: string; name?: string; values?: string[] } }) =>
      attributesApi.update(id, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['category-attributes'] });
      closeModal();
      toast.success('Attribute updated');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to update attribute'));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => attributesApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['category-attributes'] });
      setDeleteTarget(null);
      toast.success('Attribute deleted');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to delete attribute'));
    },
  });

  function openCreate() {
    setEditing(null);
    const defaultCatId = categoryOptions[0]?.id ?? '';
    setFormCategoryId(defaultCatId);
    setFormSubcategoryId('');
    setName('');
    setValueInput('');
    setValues([]);
    setModalOpen(true);
  }

  function openEdit(attribute: GlobalAttribute) {
    setEditing(attribute);
    setFormCategoryId(attribute.category_id);
    setFormSubcategoryId(attribute.subcategory_id);
    setName(attribute.name);
    setValues([...attribute.values]);
    setValueInput('');
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditing(null);
    setFormCategoryId('');
    setFormSubcategoryId('');
    setName('');
    setValueInput('');
    setValues([]);
  }

  function handleCategoryChange(categoryId: string) {
    setFormCategoryId(categoryId);
    setFormSubcategoryId('');
  }

  function addValue() {
    const nextValue = valueInput.trim();
    if (!nextValue) return;
    if (!values.some((v) => v.toLowerCase() === nextValue.toLowerCase())) {
      setValues((current) => [...current, nextValue]);
    }
    setValueInput('');
  }

  function removeValue(value: string) {
    setValues((current) => current.filter((v) => v !== value));
  }

  function handleSubmit() {
    if (!formSubcategoryId || !name.trim() || values.length === 0) return;
    const payload = { subcategory_id: formSubcategoryId, name: name.trim(), values };
    if (editing) {
      updateMutation.mutate({ id: editing.id, payload });
      return;
    }
    createMutation.mutate(payload);
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

  // Helper: label for a subcategory in the filter bar
  function subcategoryLabel(subcategoryId: string) {
    const attr = attributes.find((a) => a.subcategory_id === subcategoryId);
    if (!attr) return subcategoryId;
    return attr.category ? `${attr.category} / ${attr.subcategory}` : attr.subcategory;
  }

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-4">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-indigo-500/10 ring-1 ring-indigo-500/20">
            <FolderTree size={20} className="text-indigo-300" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-zinc-100">Subcategory Attributes</h1>
            <p className="text-sm text-zinc-500">
              Define each subcategory's attribute set and the default values sellers can start from.
            </p>
          </div>
        </div>
        <Button onClick={openCreate}>
          <Plus size={14} /> New Attribute
        </Button>
      </div>

      {subcategoryIdsWithAttrs.length > 1 && (
        <div className="mb-6 flex flex-wrap gap-2">
          {subcategoryIdsWithAttrs.map((item) => (
            <button
              key={item}
              type="button"
              onClick={() => setActiveSubcategoryId(item)}
              className={[
                'rounded-full px-3.5 py-1.5 text-sm transition-colors',
                activeSubcategoryId === item
                  ? 'bg-indigo-600 text-white'
                  : 'bg-white/5 text-zinc-400 hover:text-zinc-200',
              ].join(' ')}
            >
              {item === 'all' ? 'All' : subcategoryLabel(item)}
            </button>
          ))}
        </div>
      )}

      <Card>
        <Table>
          <Thead>
            <tr>
              <Th>Subcategory</Th>
              <Th>Attribute</Th>
              <Th>Default Values</Th>
              <Th>Created</Th>
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
            ) : filteredAttributes.length === 0 ? (
              <EmptyRow cols={5} message="No subcategory attributes defined yet" icon={<Tags size={28} />} />
            ) : (
              filteredAttributes.map((attribute) => (
                <Tr key={attribute.id}>
                  <Td className="font-medium text-zinc-200">
                    {attribute.category
                      ? `${attribute.category} / ${attribute.subcategory}`
                      : attribute.subcategory}
                  </Td>
                  <Td className="font-semibold text-zinc-100">{attribute.name}</Td>
                  <Td>
                    <div className="flex flex-wrap gap-1.5">
                      {attribute.values.map((value) => (
                        <span
                          key={value}
                          className="inline-flex rounded-lg bg-indigo-500/10 px-2.5 py-1 text-xs text-indigo-300 ring-1 ring-indigo-500/20"
                        >
                          {value}
                        </span>
                      ))}
                    </div>
                  </Td>
                  <Td className="text-xs text-zinc-500">{formatDate(attribute.created_at)}</Td>
                  <Td>
                    <div className="flex gap-1.5">
                      <Button size="sm" variant="ghost" onClick={() => openEdit(attribute)}>
                        <Pencil size={13} /> Edit
                      </Button>
                      <Button size="sm" variant="danger" onClick={() => setDeleteTarget(attribute)}>
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
            placeholder="e.g. RAM"
            value={name}
            onChange={(event) => setName(event.target.value)}
            required
          />
          <Select
            label="Category"
            value={formCategoryId}
            onChange={(event) => handleCategoryChange(event.target.value)}
            required
          >
            <option value="">Select a category</option>
            {categoryOptions.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
          </Select>
          <Select
            label="Subcategory"
            value={formSubcategoryId}
            onChange={(event) => setFormSubcategoryId(event.target.value)}
            disabled={!formCategoryId}
            required
          >
            <option value="">Select a subcategory</option>
            {formSubcategoryOptions.map((subcategory) => (
              <option key={subcategory.id} value={subcategory.id}>
                {subcategory.name}
              </option>
            ))}
          </Select>

          <div>
            <label className="mb-1.5 block text-[13px] font-medium text-zinc-300">Default Values</label>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Input
                placeholder="Add a value and press Enter"
                value={valueInput}
                onChange={(event) => setValueInput(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    event.preventDefault();
                    addValue();
                  }
                }}
              />
              <Button variant="ghost" onClick={addValue} disabled={!valueInput.trim()} className="sm:self-end">
                <Plus size={14} />
              </Button>
            </div>

            {values.length > 0 && (
              <div className="mt-3 flex flex-wrap gap-1.5">
                {values.map((value) => (
                  <span
                    key={value}
                    className="inline-flex items-center gap-1 rounded-lg bg-indigo-500/10 px-2.5 py-1 text-xs text-indigo-300 ring-1 ring-indigo-500/20"
                  >
                    {value}
                    <button
                      type="button"
                      onClick={() => removeValue(value)}
                      className="transition-colors hover:text-red-300"
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
              onClick={handleSubmit}
              loading={isSaving}
              disabled={!formSubcategoryId || !name.trim() || values.length === 0}
            >
              {editing ? 'Save Changes' : 'Create Attribute'}
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        title="Delete Attribute"
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-zinc-400">
            Delete <span className="font-medium text-zinc-200">{deleteTarget?.name}</span> from{' '}
            <span className="font-medium text-zinc-200">{deleteTarget?.subcategory}</span>? Existing
            products will keep their saved values, but new products in that subcategory will stop using
            this template.
          </p>
          <div className="flex flex-col-reverse justify-end gap-2 pt-2 sm:flex-row">
            <Button variant="ghost" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              loading={deleteMutation.isPending}
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
            >
              <Trash2 size={14} /> Delete
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
