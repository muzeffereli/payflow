import { useEffect, useRef, useState } from 'react';
import type { ChangeEvent, FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, ImagePlus, Layers, Package, Pencil, Plus, Trash2, X } from 'lucide-react';
import { toast } from 'sonner';
import { attributesApi, categoriesApi, getApiErrorMessage, productsApi, storesApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Input, Select, Textarea } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { EmptyRow, Table, Tbody, Td, Th, Thead, Tr } from '../components/ui/Table';
import { formatDate, formatMoney } from '../lib/utils';
import type { Category, GlobalAttribute, Product, ProductVariant, Subcategory } from '../lib/types';

interface LocalImage {
  key: string;
  file?: File;
  previewURL: string;
  uploadedURL?: string;
}

interface AttributeSelectionState {
  id: string;
  name: string;
  defaultValues: string[];
  selectedValues: string[];
  customValue: string;
}

const emptyForm = {
  name: '',
  description: '',
  price: '',
  currency: 'USD',
  stock: '',
  categoryId: '',
  subcategoryId: '',
  sku: '',
};

export function Products() {
  const qc = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<Product | null>(null);
  const [form, setForm] = useState({ ...emptyForm });
  const [selectedAttrs, setSelectedAttrs] = useState<AttributeSelectionState[]>([]);
  const [localImages, setLocalImages] = useState<LocalImage[]>([]);
  const [formError, setFormError] = useState('');
  const [isUploading, setIsUploading] = useState(false);
  const [variantProduct, setVariantProduct] = useState<Product | null>(null);
  const [showVariantModal, setShowVariantModal] = useState(false);
  const [editingVariant, setEditingVariant] = useState<ProductVariant | null>(null);
  const [variantForm, setVariantForm] = useState({ sku: '', price: '', stock: '0' });
  const [variantAttrValues, setVariantAttrValues] = useState<Record<string, string>>({});
  const [variantError, setVariantError] = useState('');
  const [variantSkuTouched, setVariantSkuTouched] = useState(false);

  const { data: storeData, isLoading: storeLoading } = useQuery({
    queryKey: ['my-store'],
    queryFn: () => storesApi.getMe(),
    retry: false,
  });

  const { data: attrData } = useQuery({
    queryKey: ['subcategory-attributes', form.subcategoryId],
    queryFn: () => attributesApi.list({ subcategory_id: form.subcategoryId }),
    enabled: !!form.subcategoryId,
  });

  const { data: categoriesData } = useQuery({
    queryKey: ['categories'],
    queryFn: () => categoriesApi.list(),
  });

  const { data: subcategoriesData } = useQuery({
    queryKey: ['subcategories', form.categoryId],
    queryFn: () => categoriesApi.listSubcategories(form.categoryId),
    enabled: !!form.categoryId,
  });

  const myStore = storeData?.data;
  const subcategoryAttrDefs: GlobalAttribute[] = attrData?.data?.attributes ?? [];
  const categoryOptions: Category[] = categoriesData?.data?.categories ?? [];
  const subcategoryOptions: Subcategory[] = subcategoriesData?.data?.subcategories ?? [];

  // Rebuild selectedAttrs whenever subcategory attributes load or subcategoryId changes
  const attrDataRef = attrData?.data;
  useEffect(() => {
    if (!showModal) return;
    if (!form.subcategoryId) {
      setSelectedAttrs([]);
      return;
    }
    const defs: GlobalAttribute[] = attrDataRef?.attributes ?? [];
    setSelectedAttrs(buildSelectedAttrs(defs, editing?.attributes));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attrDataRef, form.subcategoryId, showModal]);

  const { data, isLoading } = useQuery({
    queryKey: ['products', myStore?.id],
    queryFn: () => productsApi.list({ store_id: myStore?.id }),
    enabled: !!myStore?.id,
  });

  const products: Product[] = data?.data?.products ?? [];

  const createMutation = useMutation({
    mutationFn: (payload: object) => productsApi.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['products'] });
      closeModal();
      toast.success('Product created');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to create product'));
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: object }) => productsApi.update(id, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['products'] });
      closeModal();
      toast.success('Product updated');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to update product'));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => productsApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['products'] });
      toast.success('Product deleted');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to delete product'));
    },
  });

  const createVariantMutation = useMutation({
    mutationFn: ({ productId, payload }: { productId: string; payload: object }) => productsApi.createVariant(productId, payload),
    onSuccess: async (_, variables) => {
      qc.invalidateQueries({ queryKey: ['products'] });
      await refreshVariantProduct(variables.productId);
      closeVariantModal();
      toast.success('Variant created');
    },
    onError: (error) => {
      setVariantError(getApiErrorMessage(error, 'Failed to create variant'));
    },
  });

  const updateVariantMutation = useMutation({
    mutationFn: ({ productId, variantId, payload }: { productId: string; variantId: string; payload: object }) =>
      productsApi.updateVariant(productId, variantId, payload),
    onSuccess: async (_, variables) => {
      qc.invalidateQueries({ queryKey: ['products'] });
      await refreshVariantProduct(variables.productId);
      closeVariantModal();
      toast.success('Variant updated');
    },
    onError: (error) => {
      setVariantError(getApiErrorMessage(error, 'Failed to update variant'));
    },
  });

  const deleteVariantMutation = useMutation({
    mutationFn: ({ productId, variantId }: { productId: string; variantId: string }) => productsApi.deleteVariant(productId, variantId),
    onSuccess: async (_, variables) => {
      qc.invalidateQueries({ queryKey: ['products'] });
      await refreshVariantProduct(variables.productId);
      toast.success('Variant deleted');
    },
    onError: (error) => {
      toast.error(getApiErrorMessage(error, 'Failed to delete variant'));
    },
  });

  async function refreshVariantProduct(productId: string) {
    const response = await productsApi.get(productId);
    setVariantProduct(response.data);
  }

  function openCreate() {
    const categoryId = categoryOptions[0]?.id ?? '';
    setEditing(null);
    setForm({ ...emptyForm, categoryId });
    setSelectedAttrs([]);
    setLocalImages([]);
    setFormError('');
    setShowModal(true);
  }

  function openEdit(product: Product) {
    setEditing(product);
    setForm({
      name: product.name,
      description: product.description,
      price: String(product.price / 100),
      currency: product.currency,
      stock: String(product.stock),
      categoryId: product.category_id,
      subcategoryId: product.subcategory_id ?? '',
      sku: '',
    });
    setSelectedAttrs([]);
    setLocalImages(buildLocalImages(product));
    setFormError('');
    setShowModal(true);
  }

  function closeModal() {
    localImages.forEach((image) => {
      if (image.file) {
        URL.revokeObjectURL(image.previewURL);
      }
    });
    setShowModal(false);
    setEditing(null);
    setForm({ ...emptyForm });
    setSelectedAttrs([]);
    setLocalImages([]);
    setFormError('');
  }

  async function openVariants(product: Product) {
    const response = await productsApi.get(product.id);
    setVariantProduct(response.data);
  }

  function closeVariantModal() {
    setShowVariantModal(false);
    setEditingVariant(null);
    setVariantForm({ sku: '', price: '', stock: '0' });
    setVariantAttrValues({});
    setVariantError('');
    setVariantSkuTouched(false);
  }

  function openCreateVariant() {
    setEditingVariant(null);
    setVariantForm({ sku: '', price: '', stock: '0' });
    setVariantAttrValues({});
    setVariantError('');
    setVariantSkuTouched(false);
    setShowVariantModal(true);
  }

  function openEditVariant(variant: ProductVariant) {
    setEditingVariant(variant);
    setVariantForm({
      sku: variant.sku,
      price: variant.price != null ? String(variant.price / 100) : '',
      stock: String(variant.stock),
    });
    setVariantAttrValues({ ...variant.attribute_values });
    setVariantError('');
    setVariantSkuTouched(false);
    setShowVariantModal(true);
  }

  function handleCategoryChange(categoryId: string) {
    setForm((current) => ({ ...current, categoryId, subcategoryId: '' }));
    setSelectedAttrs([]);
  }

  function toggleAttributeValue(attributeName: string, value: string) {
    setSelectedAttrs((current) =>
      current.map((attribute) => {
        if (attribute.name !== attributeName) {
          return attribute;
        }
        const selected = attribute.selectedValues.includes(value);
        return {
          ...attribute,
          selectedValues: selected
            ? attribute.selectedValues.filter((item) => item !== value)
            : [...attribute.selectedValues, value],
        };
      }),
    );
  }

  function updateCustomValue(attributeName: string, value: string) {
    setSelectedAttrs((current) =>
      current.map((attribute) =>
        attribute.name === attributeName ? { ...attribute, customValue: value } : attribute,
      ),
    );
  }

  function addCustomValue(attributeName: string) {
    setSelectedAttrs((current) =>
      current.map((attribute) => {
        if (attribute.name !== attributeName) {
          return attribute;
        }
        const nextValue = attribute.customValue.trim();
        if (!nextValue) {
          return attribute;
        }
        if (attribute.selectedValues.some((value) => value.toLowerCase() === nextValue.toLowerCase())) {
          return { ...attribute, customValue: '' };
        }
        return {
          ...attribute,
          selectedValues: [...attribute.selectedValues, nextValue],
          customValue: '',
        };
      }),
    );
  }

  function removeSelectedValue(attributeName: string, value: string) {
    setSelectedAttrs((current) =>
      current.map((attribute) =>
        attribute.name === attributeName
          ? { ...attribute, selectedValues: attribute.selectedValues.filter((item) => item !== value) }
          : attribute,
      ),
    );
  }

  function handleImageSelect(event: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(event.target.files ?? []);
    const nextImages: LocalImage[] = [];

    for (const file of files) {
      if (!file.type.startsWith('image/')) {
        toast.error(`${file.name}: only image files are allowed`);
        continue;
      }
      if (file.size > 5 * 1024 * 1024) {
        toast.error(`${file.name}: image must be under 5MB`);
        continue;
      }
      nextImages.push({
        key: crypto.randomUUID(),
        file,
        previewURL: URL.createObjectURL(file),
      });
    }

    setLocalImages((current) => [...current, ...nextImages]);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }

  function removeImage(key: string) {
    setLocalImages((current) => {
      const target = current.find((image) => image.key === key);
      if (target?.file) {
        URL.revokeObjectURL(target.previewURL);
      }
      return current.filter((image) => image.key !== key);
    });
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setFormError('');

    const price = Number(form.price);
    const stock = Number(form.stock);
    if (!form.name.trim()) {
      setFormError('Product name is required.');
      return;
    }
    if (!form.categoryId) {
      setFormError('Select a category configured by the admin.');
      return;
    }
    if (!Number.isFinite(price) || price <= 0) {
      setFormError('Price must be greater than 0.');
      return;
    }
    if (!Number.isInteger(stock) || stock < 0) {
      setFormError('Stock must be 0 or more.');
      return;
    }

    const incompleteAttribute = selectedAttrs.find((attribute) => attribute.selectedValues.length === 0);
    if (incompleteAttribute) {
      setFormError(`Select at least one value for ${incompleteAttribute.name}.`);
      return;
    }

    setIsUploading(true);
    let uploadedURLs: string[] = [];
    try {
      uploadedURLs = await Promise.all(
        localImages.map(async (image) => {
          if (image.uploadedURL) {
            return image.uploadedURL;
          }
          const response = await productsApi.uploadImage(image.file!);
          return response.data.url as string;
        }),
      );
    } catch (error) {
      setFormError(getApiErrorMessage(error, 'Image upload failed.'));
      setIsUploading(false);
      return;
    }
    setIsUploading(false);

    const payload: Record<string, unknown> = {
      name: form.name.trim(),
      description: form.description.trim(),
      price: Math.round(price * 100),
      currency: form.currency,
      stock,
      category_id: form.categoryId,
      images: uploadedURLs,
      attributes: selectedAttrs.map((attribute) => ({
        name: attribute.name,
        values: attribute.selectedValues,
      })),
    };
    payload.subcategory_id = form.subcategoryId;

    if (editing) {
      updateMutation.mutate({ id: editing.id, payload });
      return;
    }

    payload.sku = form.sku.trim() || `SKU-${Date.now()}`;
    payload.store_id = myStore?.id;
    createMutation.mutate(payload);
  }

  function handleVariantSubmit(event: FormEvent) {
    event.preventDefault();
    if (!variantProduct) {
      return;
    }

    const price = variantForm.price.trim() ? Number(variantForm.price) : null;
    const stock = Number(variantForm.stock);
    if (!variantForm.sku.trim()) {
      setVariantError('Variant SKU is required.');
      return;
    }
    if (!Number.isInteger(stock) || stock < 0) {
      setVariantError('Variant stock must be 0 or more.');
      return;
    }
    if (price != null && (!Number.isFinite(price) || price <= 0)) {
      setVariantError('Variant price must be greater than 0 when set.');
      return;
    }

    const requiredAttributes = variantProduct.attributes ?? [];
    for (const attribute of requiredAttributes) {
      if (!variantAttrValues[attribute.name]) {
        setVariantError(`Choose a value for ${attribute.name}.`);
        return;
      }
    }

    const payload: Record<string, unknown> = {
      sku: variantForm.sku.trim(),
      stock,
      attribute_values: requiredAttributes.reduce<Record<string, string>>((acc, attribute) => {
        acc[attribute.name] = variantAttrValues[attribute.name];
        return acc;
      }, {}),
    };
    if (price != null) {
      payload.price = Math.round(price * 100);
    }

    if (editingVariant) {
      updateVariantMutation.mutate({ productId: variantProduct.id, variantId: editingVariant.id, payload });
      return;
    }

    createVariantMutation.mutate({ productId: variantProduct.id, payload });
  }

  const isSaving = createMutation.isPending || updateMutation.isPending || isUploading;

  if (!myStore && !storeLoading) {
    return (
      <div className="animate-fade-in">
        <div className="mb-8">
          <h1 className="text-2xl font-bold tracking-tight text-zinc-100">Products</h1>
          <p className="mt-1 text-sm text-zinc-500">Create your seller store first before listing products.</p>
        </div>

        <Card className="p-8">
          <div className="flex items-start gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-indigo-500/10">
              <Package size={22} className="text-indigo-400" />
            </div>
            <div className="space-y-2">
              <p className="font-medium text-zinc-100">No store connected yet</p>
              <p className="max-w-lg text-sm leading-relaxed text-zinc-500">
                Your catalog belongs to your store. Once the store exists, category attributes,
                product variants, and pricing rules will all attach to it from here.
              </p>
              <Button onClick={() => window.location.assign('/stores')}>Go to Stores</Button>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  if (variantProduct) {
    const variants = variantProduct.variants ?? [];
    const attrs = variantProduct.attributes ?? [];

    return (
      <div className="animate-fade-in">
        <div className="mb-8 flex items-center justify-between">
          <div>
            <button
              type="button"
              onClick={() => setVariantProduct(null)}
              className="mb-1 cursor-pointer text-xs text-indigo-400 hover:text-indigo-300"
            >
              &larr; Back to Products
            </button>
            <h1 className="text-2xl font-bold tracking-tight text-zinc-100">{variantProduct.name} Variants</h1>
            <p className="mt-1 text-sm text-zinc-500">
              Base price: {formatMoney(variantProduct.price, variantProduct.currency)}
              {attrs.length > 0 ? ` · Exact combinations required across ${attrs.map((attribute) => attribute.name).join(', ')}` : ''}
            </p>
          </div>
          <Button onClick={openCreateVariant}>
            <Plus size={15} /> Add Variant
          </Button>
        </div>

        <Card>
          <Table>
            <Thead>
              <tr>
                <Th>SKU</Th>
                {attrs.map((attribute) => (
                  <Th key={attribute.id}>{attribute.name}</Th>
                ))}
                <Th>Price</Th>
                <Th>Stock</Th>
                <Th>Status</Th>
                <Th />
              </tr>
            </Thead>
            <Tbody>
              {variants.length === 0 ? (
                <EmptyRow cols={4 + attrs.length} message="No exact variant combinations yet" icon={<Layers size={28} />} />
              ) : (
                variants.map((variant) => (
                  <Tr key={variant.id}>
                    <Td className="font-mono text-xs text-zinc-400">{variant.sku}</Td>
                    {attrs.map((attribute) => (
                      <Td key={attribute.id} className="text-zinc-300">
                        {variant.attribute_values[attribute.name] ?? '--'}
                      </Td>
                    ))}
                    <Td className="font-semibold tabular-nums">
                      {variant.price != null ? formatMoney(variant.price, variantProduct.currency) : 'Base'}
                    </Td>
                    <Td className={variant.stock === 0 ? 'text-red-400 font-medium' : 'text-zinc-300'}>
                      {variant.stock}
                    </Td>
                    <Td>
                      <Badge status={variant.status} />
                    </Td>
                    <Td>
                      <div className="flex gap-1">
                        <Button variant="ghost" size="sm" onClick={() => openEditVariant(variant)}>
                          <Pencil size={13} />
                        </Button>
                        <Button
                          variant="danger"
                          size="sm"
                          loading={deleteVariantMutation.isPending}
                          onClick={() => deleteVariantMutation.mutate({ productId: variantProduct.id, variantId: variant.id })}
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

        <Modal open={showVariantModal} onClose={closeVariantModal} title={editingVariant ? 'Edit Variant' : 'New Variant'}>
          <form onSubmit={handleVariantSubmit} className="space-y-4">
            <Input
              label="SKU"
              value={variantForm.sku}
              onChange={(event) => setVariantForm((current) => ({ ...current, sku: event.target.value }))}
              onBlur={() => setVariantSkuTouched(true)}
              error={variantSkuTouched && !variantForm.sku.trim() ? 'SKU is required.' : undefined}
              required
            />
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <Input
                label="Price Override"
                type="number"
                step="0.01"
                min="0.01"
                placeholder="Leave empty to use base price"
                value={variantForm.price}
                onChange={(event) => setVariantForm((current) => ({ ...current, price: event.target.value }))}
              />
              <Input
                label="Stock"
                type="number"
                min="0"
                value={variantForm.stock}
                onChange={(event) => setVariantForm((current) => ({ ...current, stock: event.target.value }))}
                required
              />
            </div>

            {attrs.length > 0 && (
              <div className="space-y-3">
                <p className="text-xs font-medium uppercase tracking-wider text-zinc-500">Exact Variant Combination</p>
                {attrs.map((attribute) => (
                  <Select
                    key={attribute.id}
                    label={attribute.name}
                    value={variantAttrValues[attribute.name] ?? ''}
                    onChange={(event) =>
                      setVariantAttrValues((current) => ({ ...current, [attribute.name]: event.target.value }))
                    }
                  >
                    <option value="">Select {attribute.name}</option>
                    {attribute.values.map((value) => (
                      <option key={value} value={value}>
                        {value}
                      </option>
                    ))}
                  </Select>
                ))}
              </div>
            )}

            {variantError && <p className="rounded-lg bg-red-500/10 px-4 py-3 text-sm text-red-300">{variantError}</p>}

            <div className="flex flex-col-reverse justify-end gap-2 pt-3 sm:flex-row">
              <Button type="button" variant="ghost" onClick={closeVariantModal}>
                Cancel
              </Button>
              <Button type="submit" loading={createVariantMutation.isPending || updateVariantMutation.isPending}>
                {editingVariant ? 'Save Variant' : 'Create Variant'}
              </Button>
            </div>
          </form>
        </Modal>
      </div>
    );
  }

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-100">Products</h1>
          <p className="mt-1 text-sm text-zinc-500">{products.length} products</p>
        </div>
        <Button onClick={openCreate}>
          <Plus size={15} /> Add Product
        </Button>
      </div>

      <Card>
        <Table>
          <Thead>
            <tr>
              <Th>Name</Th>
              <Th>Category</Th>
              <Th>Price</Th>
              <Th>Stock</Th>
              <Th>Status</Th>
              <Th>Variants</Th>
              <Th>Created</Th>
              <Th />
            </tr>
          </Thead>
          <Tbody>
            {isLoading ? (
              <tr>
                <td colSpan={8}>
                  <PageSpinner />
                </td>
              </tr>
            ) : products.length === 0 ? (
              <EmptyRow cols={8} message="No products yet" icon={<Package size={28} />} />
            ) : (
              products.map((product) => {
                const thumbnail = product.images?.[0]?.url ?? product.image_url;
                return (
                  <Tr key={product.id}>
                    <Td>
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-white/5 ring-1 ring-white/[0.06]">
                          {thumbnail ? (
                            <img src={thumbnail} alt={product.name} className="h-full w-full object-cover" />
                          ) : (
                            <Package size={14} className="text-zinc-400" />
                          )}
                        </div>
                        <div>
                          <p className="font-medium text-zinc-200">{product.name}</p>
                          {product.description && (
                            <p className="max-w-[220px] truncate text-xs text-zinc-600">{product.description}</p>
                          )}
                        </div>
                      </div>
                    </Td>
                    <Td className="text-zinc-500">
                      {product.category ? `${product.category}${product.subcategory ? ` / ${product.subcategory}` : ''}` : '--'}
                    </Td>
                    <Td className="font-semibold tabular-nums">{formatMoney(product.price, product.currency)}</Td>
                    <Td className={product.stock === 0 ? 'text-red-400 font-medium' : 'text-zinc-300'}>{product.stock}</Td>
                    <Td>
                      <Badge status={product.status} />
                    </Td>
                    <Td>
                      <button
                        type="button"
                        onClick={() => openVariants(product)}
                        className="flex cursor-pointer items-center gap-1 text-xs text-indigo-400 hover:text-indigo-300"
                      >
                        <Layers size={12} />
                        {(product.attributes ?? []).map((attribute) => attribute.name).join(', ') || 'Manage'}
                      </button>
                    </Td>
                    <Td className="text-xs text-zinc-500">{formatDate(product.created_at)}</Td>
                    <Td>
                      <div className="flex gap-1">
                        <Button variant="ghost" size="sm" onClick={() => openEdit(product)}>
                          <Pencil size={13} />
                        </Button>
                        <Button
                          variant="danger"
                          size="sm"
                          loading={deleteMutation.isPending}
                          onClick={() => deleteMutation.mutate(product.id)}
                        >
                          <Trash2 size={13} />
                        </Button>
                      </div>
                    </Td>
                  </Tr>
                );
              })
            )}
          </Tbody>
        </Table>
      </Card>

      <Modal open={showModal} onClose={closeModal} title={editing ? 'Edit Product' : 'New Product'} size="xl">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <Input
                label="Name"
                value={form.name}
                onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                required
              />
            </div>
            <div className="sm:col-span-2">
              <Textarea
                label="Description"
                rows={3}
                value={form.description}
                onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
              />
            </div>

            {!editing && (
              <div className="sm:col-span-2">
                <Input
                  label="SKU"
                  value={form.sku}
                  placeholder="Auto-generated if empty"
                  onChange={(event) => setForm((current) => ({ ...current, sku: event.target.value }))}
                />
              </div>
            )}

            <Input
              label={`Price (${form.currency})`}
              type="number"
              step="0.01"
              min="0.01"
              value={form.price}
              onChange={(event) => setForm((current) => ({ ...current, price: event.target.value }))}
              required
            />
            <Input
              label="Stock"
              type="number"
              min="0"
              value={form.stock}
              onChange={(event) => setForm((current) => ({ ...current, stock: event.target.value }))}
              required
            />
            <Select label="Category" value={form.categoryId} onChange={(event) => handleCategoryChange(event.target.value)}>
              <option value="">Select a category</option>
              {categoryOptions.map((category) => (
                <option key={category.id} value={category.id}>
                  {category.name}
                </option>
              ))}
            </Select>
            <Select
              label="Subcategory"
              value={form.subcategoryId}
              onChange={(event) => setForm((current) => ({ ...current, subcategoryId: event.target.value }))}
              disabled={!form.categoryId}
            >
              <option value="">None</option>
              {subcategoryOptions.map((subcategory) => (
                <option key={subcategory.id} value={subcategory.id}>
                  {subcategory.name}
                </option>
              ))}
            </Select>
            <Select
              label="Currency"
              value={form.currency}
              onChange={(event) => setForm((current) => ({ ...current, currency: event.target.value }))}
            >
              <option value="USD">USD</option>
              <option value="EUR">EUR</option>
              <option value="GBP">GBP</option>
            </Select>
          </div>

          <div>
            <label className="mb-2 block text-xs font-medium text-zinc-400">
              Images
              {localImages.length > 0 ? <span className="ml-1 text-zinc-600">({localImages.length})</span> : null}
            </label>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              multiple
              className="hidden"
              onChange={handleImageSelect}
            />
            <div className="flex flex-wrap gap-2">
              {localImages.map((image, index) => (
                <div
                  key={image.key}
                  className="relative h-20 w-20 shrink-0 overflow-hidden rounded-xl bg-white/[0.02] ring-1 ring-white/[0.08]"
                >
                  <img src={image.previewURL} alt={`Product ${index + 1}`} className="h-full w-full object-cover" />
                  {index === 0 && (
                    <div className="absolute bottom-0 left-0 right-0 bg-black/70 py-0.5 text-center text-[9px] text-zinc-300">
                      Main
                    </div>
                  )}
                  <button
                    type="button"
                    onClick={() => removeImage(image.key)}
                    className="absolute right-1 top-1 flex h-5 w-5 items-center justify-center rounded bg-black/70 text-zinc-300 hover:text-white"
                  >
                    <X size={11} />
                  </button>
                </div>
              ))}

              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="flex h-20 w-20 shrink-0 flex-col items-center justify-center gap-1 rounded-xl bg-white/[0.02] text-zinc-500 ring-1 ring-dashed ring-white/[0.12] transition-colors hover:text-zinc-300 hover:ring-white/25"
              >
                <ImagePlus size={16} />
                <span className="text-[10px]">Add</span>
              </button>
            </div>
            {localImages.length === 0 && (
              <p className="mt-1.5 text-[11px] text-zinc-600">The first image becomes the product thumbnail.</p>
            )}
          </div>

          <div className="space-y-3">
            <div>
              <p className="text-xs font-medium uppercase tracking-wider text-zinc-500">Subcategory Attributes</p>
              <p className="mt-1 text-sm text-zinc-500">
                Choose admin default values or add seller-specific values that still work in filtering and variants.
              </p>
            </div>

            {!form.subcategoryId ? (
              <p className="rounded-xl bg-white/[0.03] px-4 py-3 text-sm text-zinc-500 ring-1 ring-white/[0.05]">
                Select a subcategory to load its admin-defined attributes.
              </p>
            ) : subcategoryAttrDefs.length === 0 ? (
              <p className="rounded-xl bg-amber-500/10 px-4 py-3 text-sm text-amber-300 ring-1 ring-amber-500/20">
                This subcategory has no admin-defined attributes yet. Ask an admin to configure it first.
              </p>
            ) : (
              selectedAttrs.map((attribute) => {
                const selectedSet = new Set(attribute.selectedValues.map((value) => value.toLowerCase()));
                const customSelected = attribute.selectedValues.filter(
                  (value) => !attribute.defaultValues.some((defaultValue) => defaultValue.toLowerCase() === value.toLowerCase()),
                );

                return (
                  <div key={attribute.id} className="rounded-2xl bg-white/[0.02] p-4 ring-1 ring-white/[0.06]">
                    <div className="mb-3">
                      <p className="font-medium text-zinc-100">{attribute.name}</p>
                      <p className="text-xs text-zinc-500">Pick the values buyers should see and filter by.</p>
                    </div>

                    <div className="mb-3 flex flex-wrap gap-1.5">
                      {attribute.defaultValues.map((value) => {
                        const selected = selectedSet.has(value.toLowerCase());
                        return (
                          <button
                            key={value}
                            type="button"
                            onClick={() => toggleAttributeValue(attribute.name, value)}
                            className={[
                              'rounded-lg px-2.5 py-1 text-xs transition-colors',
                              selected
                                ? 'bg-indigo-500/20 text-indigo-300 ring-1 ring-indigo-500/30'
                                : 'bg-white/[0.04] text-zinc-400 ring-1 ring-white/[0.08] hover:text-zinc-200',
                            ].join(' ')}
                          >
                            {selected ? <span className="mr-1 inline-flex"><Check size={11} /></span> : null}
                            {value}
                          </button>
                        );
                      })}
                    </div>

                    <div className="flex flex-col gap-2 sm:flex-row">
                      <Input
                        label="Custom Value"
                        placeholder={`Add custom ${attribute.name.toLowerCase()} value`}
                        value={attribute.customValue}
                        onChange={(event) => updateCustomValue(attribute.name, event.target.value)}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter') {
                            event.preventDefault();
                            addCustomValue(attribute.name);
                          }
                        }}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        className="sm:self-end"
                        onClick={() => addCustomValue(attribute.name)}
                        disabled={!attribute.customValue.trim()}
                      >
                        <Plus size={14} /> Add Custom
                      </Button>
                    </div>

                    <div className="mt-3 flex flex-wrap gap-1.5">
                      {attribute.selectedValues.map((value) => (
                        <span
                          key={`${attribute.name}-${value}`}
                          className="inline-flex items-center gap-1 rounded-lg bg-emerald-500/10 px-2.5 py-1 text-xs text-emerald-300 ring-1 ring-emerald-500/20"
                        >
                          {value}
                          {customSelected.includes(value) ? <span className="text-[10px] text-emerald-200/80">custom</span> : null}
                          <button
                            type="button"
                            onClick={() => removeSelectedValue(attribute.name, value)}
                            className="transition-colors hover:text-red-300"
                          >
                            <X size={11} />
                          </button>
                        </span>
                      ))}
                    </div>
                  </div>
                );
              })
            )}
          </div>

          {formError && <p className="rounded-lg bg-red-500/10 px-4 py-3 text-sm text-red-300">{formError}</p>}

          <div className="flex flex-col-reverse justify-end gap-2 pt-2 sm:flex-row">
            <Button type="button" variant="ghost" onClick={closeModal}>
              Cancel
            </Button>
            <Button type="submit" loading={isSaving}>
              {editing ? 'Save Changes' : 'Create Product'}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

function buildSelectedAttrs(definitions: GlobalAttribute[], existingProductAttrs?: Product['attributes']): AttributeSelectionState[] {
  const byName = new Map((existingProductAttrs ?? []).map((attribute) => [attribute.name, attribute.values]));
  return [...definitions]
    .sort((a, b) => a.position - b.position || a.name.localeCompare(b.name))
    .map((definition) => ({
      id: definition.id,
      name: definition.name,
      defaultValues: definition.values,
      selectedValues: [...(byName.get(definition.name) ?? [])],
      customValue: '',
    }));
}

function buildLocalImages(product: Product): LocalImage[] {
  if (product.images && product.images.length > 0) {
    return product.images.map((image) => ({
      key: image.id,
      previewURL: image.url,
      uploadedURL: image.url,
    }));
  }
  if (product.image_url) {
    return [
      {
        key: 'legacy',
        previewURL: product.image_url,
        uploadedURL: product.image_url,
      },
    ];
  }
  return [];
}
