import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FolderTree, Layers3, Pencil, Plus, Tags, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { categoriesApi, getApiErrorMessage } from '../lib/api';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { EmptyRow, Table, Tbody, Td, Th, Thead, Tr } from '../components/ui/Table';
import { formatDate } from '../lib/utils';
import type { Category, Subcategory } from '../lib/types';

type CategoryFormState = {
  mode: 'create' | 'edit';
  category: Category | null;
};

type SubcategoryFormState = {
  mode: 'create' | 'edit';
  subcategory: Subcategory | null;
};

type DeleteTarget =
  | { type: 'category'; id: string; name: string }
  | { type: 'subcategory'; id: string; name: string };

export function AdminCategories() {
  const qc = useQueryClient();
  const [selectedCategoryId, setSelectedCategoryId] = useState('');
  const [categoryForm, setCategoryForm] = useState<CategoryFormState | null>(null);
  const [subcategoryForm, setSubcategoryForm] = useState<SubcategoryFormState | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null);
  const [nameInput, setNameInput] = useState('');

  const { data: categoriesData, isLoading: categoriesLoading } = useQuery({
    queryKey: ['admin-categories'],
    queryFn: () => categoriesApi.list(),
  });

  const categories: Category[] = categoriesData?.data?.categories ?? [];

  useEffect(() => {
    if (categories.length === 0) {
      setSelectedCategoryId('');
      return;
    }
    const stillExists = categories.some((category) => category.id === selectedCategoryId);
    if (!stillExists) {
      setSelectedCategoryId(categories[0].id);
    }
  }, [categories, selectedCategoryId]);

  const selectedCategory = categories.find((category) => category.id === selectedCategoryId) ?? null;

  const { data: subcategoriesData, isLoading: subcategoriesLoading } = useQuery({
    queryKey: ['admin-subcategories', selectedCategoryId],
    queryFn: () => categoriesApi.listSubcategories(selectedCategoryId),
    enabled: !!selectedCategoryId,
  });

  const subcategories: Subcategory[] = subcategoriesData?.data?.subcategories ?? [];

  const stats = useMemo(
    () => ({
      categories: categories.length,
      subcategories: subcategories.length,
      selectedName: selectedCategory?.name ?? 'None selected',
    }),
    [categories.length, selectedCategory?.name, subcategories.length],
  );

  const createCategoryMutation = useMutation({
    mutationFn: (payload: { name: string }) => categoriesApi.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-categories'] });
      closeForm();
      toast.success('Category created');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to create category'));
    },
  });

  const updateCategoryMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: { name?: string } }) => categoriesApi.update(id, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-categories'] });
      closeForm();
      toast.success('Category updated');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to update category'));
    },
  });

  const deleteCategoryMutation = useMutation({
    mutationFn: (id: string) => categoriesApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-categories'] });
      qc.invalidateQueries({ queryKey: ['admin-subcategories'] });
      setDeleteTarget(null);
      toast.success('Category deleted');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to delete category'));
    },
  });

  const createSubcategoryMutation = useMutation({
    mutationFn: (payload: { category_id: string; name: string }) => categoriesApi.createSubcategory(payload),
    onSuccess: (_, payload) => {
      qc.invalidateQueries({ queryKey: ['admin-subcategories', payload.category_id] });
      closeForm();
      toast.success('Subcategory created');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to create subcategory'));
    },
  });

  const updateSubcategoryMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: { category_id?: string; name?: string } }) =>
      categoriesApi.updateSubcategory(id, payload),
    onSuccess: (_, { payload }) => {
      qc.invalidateQueries({ queryKey: ['admin-subcategories', payload.category_id ?? selectedCategoryId] });
      closeForm();
      toast.success('Subcategory updated');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to update subcategory'));
    },
  });

  const deleteSubcategoryMutation = useMutation({
    mutationFn: (id: string) => categoriesApi.deleteSubcategory(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-subcategories', selectedCategoryId] });
      setDeleteTarget(null);
      toast.success('Subcategory deleted');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to delete subcategory'));
    },
  });

  function openCreateCategory() {
    setCategoryForm({ mode: 'create', category: null });
    setSubcategoryForm(null);
    setNameInput('');
  }

  function openEditCategory(category: Category) {
    setCategoryForm({ mode: 'edit', category });
    setSubcategoryForm(null);
    setNameInput(category.name);
  }

  function openCreateSubcategory() {
    setSubcategoryForm({ mode: 'create', subcategory: null });
    setCategoryForm(null);
    setNameInput('');
  }

  function openEditSubcategory(subcategory: Subcategory) {
    setSubcategoryForm({ mode: 'edit', subcategory });
    setCategoryForm(null);
    setNameInput(subcategory.name);
  }

  function closeForm() {
    setCategoryForm(null);
    setSubcategoryForm(null);
    setNameInput('');
  }

  function handleCategorySubmit() {
    const name = nameInput.trim();
    if (!name) {
      return;
    }
    if (categoryForm?.mode === 'edit' && categoryForm.category) {
      updateCategoryMutation.mutate({ id: categoryForm.category.id, payload: { name } });
      return;
    }
    createCategoryMutation.mutate({ name });
  }

  function handleSubcategorySubmit() {
    const name = nameInput.trim();
    if (!name || !selectedCategoryId) {
      return;
    }
    if (subcategoryForm?.mode === 'edit' && subcategoryForm.subcategory) {
      updateSubcategoryMutation.mutate({
        id: subcategoryForm.subcategory.id,
        payload: { category_id: selectedCategoryId, name },
      });
      return;
    }
    createSubcategoryMutation.mutate({ category_id: selectedCategoryId, name });
  }

  function handleDeleteConfirm() {
    if (!deleteTarget) {
      return;
    }
    if (deleteTarget.type === 'category') {
      deleteCategoryMutation.mutate(deleteTarget.id);
      return;
    }
    deleteSubcategoryMutation.mutate(deleteTarget.id);
  }

  const categorySaving = createCategoryMutation.isPending || updateCategoryMutation.isPending;
  const subcategorySaving = createSubcategoryMutation.isPending || updateSubcategoryMutation.isPending;
  const deleting = deleteCategoryMutation.isPending || deleteSubcategoryMutation.isPending;

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-4">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-sky-500/10 ring-1 ring-sky-500/20">
            <FolderTree size={20} className="text-sky-300" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-zinc-100">Category Management</h1>
            <p className="text-sm text-zinc-500">
              Create the product taxonomy that attributes, seller listings, and storefront filters depend on.
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" onClick={openCreateCategory}>
            <Plus size={14} /> New Category
          </Button>
          <Button onClick={openCreateSubcategory} disabled={!selectedCategoryId}>
            <Plus size={14} /> New Subcategory
          </Button>
        </div>
      </div>

      <div className="mb-8 grid grid-cols-1 gap-4 md:grid-cols-3">
        <div className="rounded-2xl bg-sky-500/5 p-5 ring-1 ring-sky-500/10">
          <p className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-sky-400">Categories</p>
          <p className="text-2xl font-bold text-sky-300 tabular-nums">{stats.categories}</p>
        </div>
        <div className="rounded-2xl bg-indigo-500/5 p-5 ring-1 ring-indigo-500/10">
          <p className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-indigo-400">Selected Subcategories</p>
          <p className="text-2xl font-bold text-indigo-300 tabular-nums">{stats.subcategories}</p>
        </div>
        <div className="rounded-2xl bg-emerald-500/5 p-5 ring-1 ring-emerald-500/10">
          <p className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-emerald-400">Active Category</p>
          <p className="truncate text-lg font-semibold text-emerald-300">{stats.selectedName}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-[1.2fr_1fr]">
        <Card className="overflow-hidden">
          <div className="flex items-center justify-between border-b border-[var(--color-border)] px-5 py-4">
            <div>
              <p className="text-sm font-semibold text-zinc-100">Categories</p>
              <p className="text-xs text-zinc-500">Pick a category to manage its subcategories.</p>
            </div>
            <Button size="sm" variant="ghost" onClick={openCreateCategory}>
              <Plus size={13} /> Add
            </Button>
          </div>
          <Table>
            <Thead>
              <tr>
                <Th>Name</Th>
                <Th>Slug</Th>
                <Th>Created</Th>
                <Th>Actions</Th>
              </tr>
            </Thead>
            <Tbody>
              {categoriesLoading ? (
                <tr>
                  <td colSpan={4}>
                    <PageSpinner />
                  </td>
                </tr>
              ) : categories.length === 0 ? (
                <EmptyRow cols={4} message="No categories defined yet" icon={<Tags size={28} />} />
              ) : (
                categories.map((category) => (
                  <Tr
                    key={category.id}
                    className={selectedCategoryId === category.id ? 'bg-white/[0.03]' : undefined}
                  >
                    <Td>
                      <button
                        type="button"
                        onClick={() => setSelectedCategoryId(category.id)}
                        className="text-left"
                      >
                        <p className="font-medium text-zinc-200">{category.name}</p>
                        <p className="text-xs text-zinc-600">{category.id.slice(0, 10)}...</p>
                      </button>
                    </Td>
                    <Td className="text-zinc-500">{category.slug}</Td>
                    <Td className="text-xs text-zinc-500">{formatDate(category.created_at)}</Td>
                    <Td>
                      <div className="flex gap-1.5">
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={`Edit category ${category.name}`}
                          onClick={() => openEditCategory(category)}
                        >
                          <Pencil size={13} />
                        </Button>
                        <Button
                          size="sm"
                          variant="danger"
                          aria-label={`Delete category ${category.name}`}
                          onClick={() => setDeleteTarget({ type: 'category', id: category.id, name: category.name })}
                        >
                          <Trash2 size={13} />
                        </Button>
                      </div>
                    </Td>
                  </Tr>
                ))
              )}
            </Tbody>
          </Table>
        </Card>

        <Card className="overflow-hidden">
          <div className="flex items-center justify-between border-b border-[var(--color-border)] px-5 py-4">
            <div>
              <p className="text-sm font-semibold text-zinc-100">Subcategories</p>
              <p className="text-xs text-zinc-500">
                {selectedCategory ? `Nested under ${selectedCategory.name}` : 'Select a category first.'}
              </p>
            </div>
            <Button size="sm" variant="ghost" onClick={openCreateSubcategory} disabled={!selectedCategory}>
              <Plus size={13} /> Add
            </Button>
          </div>
          <Table>
            <Thead>
              <tr>
                <Th>Name</Th>
                <Th>Slug</Th>
                <Th>Created</Th>
                <Th>Actions</Th>
              </tr>
            </Thead>
            <Tbody>
              {!selectedCategory ? (
                <EmptyRow cols={4} message="Choose a category to manage subcategories" icon={<Layers3 size={28} />} />
              ) : subcategoriesLoading ? (
                <tr>
                  <td colSpan={4}>
                    <PageSpinner />
                  </td>
                </tr>
              ) : subcategories.length === 0 ? (
                <EmptyRow cols={4} message="No subcategories in this category yet" icon={<Layers3 size={28} />} />
              ) : (
                subcategories.map((subcategory) => (
                  <Tr key={subcategory.id}>
                    <Td>
                      <p className="font-medium text-zinc-200">{subcategory.name}</p>
                      <p className="text-xs text-zinc-600">{subcategory.id.slice(0, 10)}...</p>
                    </Td>
                    <Td className="text-zinc-500">{subcategory.slug}</Td>
                    <Td className="text-xs text-zinc-500">{formatDate(subcategory.created_at)}</Td>
                    <Td>
                      <div className="flex gap-1.5">
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={`Edit subcategory ${subcategory.name}`}
                          onClick={() => openEditSubcategory(subcategory)}
                        >
                          <Pencil size={13} />
                        </Button>
                        <Button
                          size="sm"
                          variant="danger"
                          aria-label={`Delete subcategory ${subcategory.name}`}
                          onClick={() => setDeleteTarget({ type: 'subcategory', id: subcategory.id, name: subcategory.name })}
                        >
                          <Trash2 size={13} />
                        </Button>
                      </div>
                    </Td>
                  </Tr>
                ))
              )}
            </Tbody>
          </Table>
        </Card>
      </div>

      <Modal
        open={!!categoryForm}
        onClose={closeForm}
        title={categoryForm?.mode === 'edit' ? 'Edit Category' : 'New Category'}
        size="sm"
      >
        <div className="space-y-4">
          <Input
            label="Category Name"
            placeholder="e.g. Shoes"
            value={nameInput}
            onChange={(event) => setNameInput(event.target.value)}
            required
          />
          <div className="flex flex-col-reverse justify-end gap-2 pt-2 sm:flex-row">
            <Button variant="ghost" onClick={closeForm}>
              Cancel
            </Button>
            <Button onClick={handleCategorySubmit} loading={categorySaving} disabled={!nameInput.trim()}>
              {categoryForm?.mode === 'edit' ? 'Save Changes' : 'Create Category'}
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        open={!!subcategoryForm}
        onClose={closeForm}
        title={subcategoryForm?.mode === 'edit' ? 'Edit Subcategory' : 'New Subcategory'}
        size="sm"
      >
        <div className="space-y-4">
          <div className="rounded-xl bg-white/[0.03] px-4 py-3 ring-1 ring-white/[0.05]">
            <p className="text-xs font-medium uppercase tracking-wider text-zinc-500">Parent Category</p>
            <p className="mt-1 text-sm font-medium text-zinc-200">{selectedCategory?.name ?? 'No category selected'}</p>
          </div>
          <Input
            label="Subcategory Name"
            placeholder="e.g. Running"
            value={nameInput}
            onChange={(event) => setNameInput(event.target.value)}
            required
          />
          <div className="flex flex-col-reverse justify-end gap-2 pt-2 sm:flex-row">
            <Button variant="ghost" onClick={closeForm}>
              Cancel
            </Button>
            <Button
              onClick={handleSubcategorySubmit}
              loading={subcategorySaving}
              disabled={!selectedCategoryId || !nameInput.trim()}
            >
              {subcategoryForm?.mode === 'edit' ? 'Save Changes' : 'Create Subcategory'}
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        title={`Delete ${deleteTarget?.type === 'category' ? 'Category' : 'Subcategory'}`}
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-zinc-400">
            Delete <span className="font-medium text-zinc-200">{deleteTarget?.name}</span>?
            {deleteTarget?.type === 'category'
              ? ' Its subcategories will be removed too, and products will keep their stored IDs only if the backend allows the delete.'
              : ' Products using it will lose that subcategory reference if the backend allows the delete.'}
          </p>
          <div className="flex flex-col-reverse justify-end gap-2 pt-2 sm:flex-row">
            <Button variant="ghost" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button variant="danger" onClick={handleDeleteConfirm} loading={deleting}>
              <Trash2 size={14} /> Delete
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
