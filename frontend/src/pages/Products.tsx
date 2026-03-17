import { useState, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, Package, X, Layers, Check, ImagePlus } from 'lucide-react';
import { toast } from 'sonner';
import { productsApi, storesApi, attributesApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Input, Select, Textarea } from '../components/ui/Input';
import { Table, Thead, Th, Tbody, Tr, Td, EmptyRow } from '../components/ui/Table';
import { Modal } from '../components/ui/Modal';
import { PageSpinner } from '../components/ui/Spinner';
import { formatMoney, formatDate } from '../lib/utils';
import type { Product, ProductVariant, GlobalAttribute } from '../lib/types';

interface SelectedAttr { globalAttrId: string; name: string; selectedValues: string[] }
interface LocalImage { key: string; file?: File; previewURL: string; uploadedURL?: string }

const emptyForm = { name: '', description: '', price: '', currency: 'USD', stock: '', category: '', sku: '' };

export function Products() {
  const qc = useQueryClient();
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<Product | null>(null);
  const [form, setForm] = useState({ ...emptyForm });
  const [selectedAttrs, setSelectedAttrs] = useState<SelectedAttr[]>([]);
  const [formError, setFormError] = useState('');
  const [localImages, setLocalImages] = useState<LocalImage[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [variantProduct, setVariantProduct] = useState<Product | null>(null);
  const [showVariantModal, setShowVariantModal] = useState(false);
  const [variantForm, setVariantForm] = useState({ sku: '', price: '', stock: '0' });
  const [variantAttrValues, setVariantAttrValues] = useState<Record<string, string>>({});
  const [editingVariant, setEditingVariant] = useState<ProductVariant | null>(null);

  const { data: globalAttrsData } = useQuery({
    queryKey: ['global-attributes'],
    queryFn: () => attributesApi.list(),
  });
  const globalAttrs: GlobalAttribute[] = globalAttrsData?.data?.attributes ?? [];

  const { data: storeData, isLoading: storeLoading } = useQuery({
    queryKey: ['my-store'],
    queryFn: () => storesApi.getMe(),
    retry: false,
  });
  const myStore = storeData?.data;

  const { data, isLoading } = useQuery({
    queryKey: ['products', myStore?.id],
    queryFn: () => productsApi.list({ store_id: myStore?.id }),
    enabled: !!myStore?.id,
  });

  const createMutation = useMutation({
    mutationFn: (data: object) => productsApi.create(data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['products'] }); closeModal(); toast.success('Product created'); },
    onError: () => { toast.error('Failed to create product'); },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: object }) => productsApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['products'] }); closeModal(); toast.success('Product updated'); },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => productsApi.delete(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['products'] }); toast.success('Product deleted'); },
  });

  const createVariantMutation = useMutation({
    mutationFn: ({ productId, data }: { productId: string; data: object }) =>
      productsApi.createVariant(productId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['products'] });
      if (variantProduct) refreshVariantProduct(variantProduct.id);
      closeVariantModal();
    },
  });

  const updateVariantMutation = useMutation({
    mutationFn: ({ productId, variantId, data }: { productId: string; variantId: string; data: object }) =>
      productsApi.updateVariant(productId, variantId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['products'] });
      if (variantProduct) refreshVariantProduct(variantProduct.id);
      closeVariantModal();
    },
  });

  const deleteVariantMutation = useMutation({
    mutationFn: ({ productId, variantId }: { productId: string; variantId: string }) =>
      productsApi.deleteVariant(productId, variantId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['products'] });
      if (variantProduct) refreshVariantProduct(variantProduct.id);
    },
  });

  const refreshVariantProduct = async (id: string) => {
    const res = await productsApi.get(id);
    setVariantProduct(res.data);
  };

  const openCreate = () => {
    setEditing(null);
    setForm({ ...emptyForm });
    setSelectedAttrs([]);
    setFormError('');
    setLocalImages([]);
    setShowModal(true);
  };

  const openEdit = (p: Product) => {
    setEditing(p);
    setForm({
      name: p.name, description: p.description, price: String(p.price / 100),
      currency: p.currency, stock: String(p.stock), category: p.category, sku: '',
    });
    setSelectedAttrs(
      (p.attributes ?? []).map((a) => ({
        globalAttrId: '',
        name: a.name,
        selectedValues: a.values,
      }))
    );
    setFormError('');
    const existingImages: LocalImage[] =
      (p.images ?? []).length > 0
        ? p.images!.map((img) => ({ key: img.id, previewURL: img.url, uploadedURL: img.url }))
        : p.image_url
        ? [{ key: 'legacy', previewURL: p.image_url, uploadedURL: p.image_url }]
        : [];
    setLocalImages(existingImages);
    setShowModal(true);
  };

  const closeModal = () => {
    localImages.forEach((img) => { if (img.file) URL.revokeObjectURL(img.previewURL); });
    setShowModal(false);
    setEditing(null);
    setLocalImages([]);
  };

  const handleImageSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? []);
    const newImages: LocalImage[] = [];
    for (const file of files) {
      if (!file.type.startsWith('image/')) { toast.error(`${file.name}: only image files allowed`); continue; }
      if (file.size > 5 * 1024 * 1024) { toast.error(`${file.name}: must be under 5 MB`); continue; }
      newImages.push({ key: crypto.randomUUID(), file, previewURL: URL.createObjectURL(file) });
    }
    setLocalImages((prev) => [...prev, ...newImages]);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const removeImage = (key: string) => {
    setLocalImages((prev) => {
      const img = prev.find((i) => i.key === key);
      if (img?.file) URL.revokeObjectURL(img.previewURL);
      return prev.filter((i) => i.key !== key);
    });
  };

  const openVariants = async (p: Product) => {
    const res = await productsApi.get(p.id);
    setVariantProduct(res.data);
  };

  const openCreateVariant = () => {
    setEditingVariant(null);
    setVariantForm({ sku: '', price: '', stock: '0' });
    setVariantAttrValues({});
    setShowVariantModal(true);
  };

  const openEditVariant = (v: ProductVariant) => {
    setEditingVariant(v);
    setVariantForm({
      sku: v.sku,
      price: v.price != null ? String(v.price / 100) : '',
      stock: String(v.stock),
    });
    setVariantAttrValues({ ...v.attribute_values });
    setShowVariantModal(true);
  };

  const closeVariantModal = () => { setShowVariantModal(false); setEditingVariant(null); };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');

    const price = parseFloat(form.price);
    const stock = parseInt(form.stock);

    if (!form.name.trim()) { setFormError('Name is required'); return; }
    if (isNaN(price) || price <= 0) { setFormError('Price must be greater than 0'); return; }
    if (isNaN(stock) || stock < 0) { setFormError('Stock must be 0 or more'); return; }

    setIsUploading(true);
    let uploadedURLs: string[];
    try {
      uploadedURLs = await Promise.all(
        localImages.map(async (img) => {
          if (img.uploadedURL) return img.uploadedURL;
          const res = await productsApi.uploadImage(img.file!);
          return (res.data as { url: string }).url;
        })
      );
    } catch {
      toast.error('Image upload failed');
      setIsUploading(false);
      return;
    } finally {
      setIsUploading(false);
    }

    const parsedAttrs = selectedAttrs
      .filter((a) => a.name && a.selectedValues.length > 0)
      .map((a) => ({ name: a.name, values: a.selectedValues }));

    const payload: Record<string, unknown> = {
      name: form.name.trim(),
      description: form.description.trim(),
      price: Math.round(price * 100),
      currency: form.currency,
      stock,
      category: form.category.trim(),
      images: uploadedURLs,
    };
    if (parsedAttrs.length > 0 || editing) {
      payload.attributes = parsedAttrs;
    }

    if (editing) {
      updateMutation.mutate({ id: editing.id, data: payload });
    } else {
      payload.sku = form.sku.trim() || `SKU-${Date.now()}`;
      payload.store_id = myStore?.id;
      createMutation.mutate(payload);
    }
  };

  const handleVariantSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!variantProduct) return;

    const payload: Record<string, unknown> = {
      sku: variantForm.sku.trim(),
      stock: parseInt(variantForm.stock) || 0,
      attribute_values: variantAttrValues,
    };
    if (variantForm.price.trim()) {
      payload.price = Math.round(parseFloat(variantForm.price) * 100);
    }

    if (editingVariant) {
      updateVariantMutation.mutate({ productId: variantProduct.id, variantId: editingVariant.id, data: payload });
    } else {
      createVariantMutation.mutate({ productId: variantProduct.id, data: payload });
    }
  };

  const toggleGlobalAttr = (ga: GlobalAttribute) => {
    const exists = selectedAttrs.find((a) => a.name === ga.name);
    if (exists) {
      setSelectedAttrs(selectedAttrs.filter((a) => a.name !== ga.name));
    } else {
      setSelectedAttrs([...selectedAttrs, { globalAttrId: ga.id, name: ga.name, selectedValues: [] }]);
    }
  };

  const toggleAttrValue = (attrName: string, value: string) => {
    setSelectedAttrs(
      selectedAttrs.map((a) => {
        if (a.name !== attrName) return a;
        const has = a.selectedValues.includes(value);
        return {
          ...a,
          selectedValues: has
            ? a.selectedValues.filter((v) => v !== value)
            : [...a.selectedValues, value],
        };
      })
    );
  };

  const products: Product[] = data?.data?.products ?? [];
  const isSaving = createMutation.isPending || updateMutation.isPending || isUploading;

  if (!myStore && !storeLoading) {
    return (
      <div className="animate-fade-in">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Products</h1>
          <p className="mt-1 text-sm text-zinc-500">
            Create a store first before you start listing products.
          </p>
        </div>

        <Card className="p-8">
          <div className="flex items-start gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-indigo-500/10">
              <Package size={22} className="text-indigo-400" />
            </div>
            <div className="space-y-2">
              <p className="font-medium text-zinc-100">No store connected yet</p>
              <p className="max-w-lg text-sm leading-relaxed text-zinc-500">
                Your catalog belongs to your seller store. Once you create and activate a store,
                your products and thumbnails will show up here.
              </p>
              <Button onClick={() => window.location.assign('/stores')}>
                Go to Stores
              </Button>
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
            <button onClick={() => setVariantProduct(null)} className="text-xs text-indigo-400 hover:text-indigo-300 mb-1 cursor-pointer">&larr; Back to Products</button>
            <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">{variantProduct.name} â€” Variants</h1>
            <p className="text-zinc-500 text-sm mt-1">
              Base price: {formatMoney(variantProduct.price, variantProduct.currency)} &middot;{' '}
              {attrs.length > 0 && `Attributes: ${attrs.map((a) => a.name).join(', ')}`}
            </p>
          </div>
          <Button onClick={openCreateVariant}><Plus size={15} /> Add Variant</Button>
        </div>

        <Card>
          <Table>
            <Thead>
              <tr>
                <Th>SKU</Th>
                {attrs.map((a) => <Th key={a.id}>{a.name}</Th>)}
                <Th>Price</Th>
                <Th>Stock</Th>
                <Th>Status</Th>
                <Th />
              </tr>
            </Thead>
            <Tbody>
              {variants.length === 0 ? (
                <EmptyRow cols={4 + attrs.length} message="No variants yet" icon={<Layers size={28} />} />
              ) : (
                variants.map((v) => (
                  <Tr key={v.id}>
                    <Td className="font-mono text-xs text-zinc-400">{v.sku}</Td>
                    {attrs.map((a) => (
                      <Td key={a.id} className="text-zinc-300">{v.attribute_values[a.name] ?? '--'}</Td>
                    ))}
                    <Td className="font-semibold tabular-nums">
                      {v.price != null
                        ? formatMoney(v.price, variantProduct.currency)
                        : <span className="text-zinc-600">base</span>
                      }
                    </Td>
                    <Td>
                      <span className={v.stock < 10 ? (v.stock === 0 ? 'text-red-400 font-medium' : 'text-amber-400') : 'text-zinc-300'}>
                        {v.stock}
                      </span>
                    </Td>
                    <Td><Badge status={v.status} /></Td>
                    <Td>
                      <div className="flex gap-1">
                        <Button variant="ghost" size="sm" onClick={() => openEditVariant(v)}><Pencil size={13} /></Button>
                        <Button
                          variant="danger" size="sm"
                          loading={deleteVariantMutation.isPending}
                          onClick={() => deleteVariantMutation.mutate({ productId: variantProduct.id, variantId: v.id })}
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
            <Input label="SKU" value={variantForm.sku} onChange={(e) => setVariantForm({ ...variantForm, sku: e.target.value })} required />
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <Input
                label="Price Override (leave empty for base)"
                type="number" step="0.01" min="0.01" placeholder="Base price"
                value={variantForm.price}
                onChange={(e) => setVariantForm({ ...variantForm, price: e.target.value })}
              />
              <Input
                label="Stock" type="number" min="0"
                value={variantForm.stock}
                onChange={(e) => setVariantForm({ ...variantForm, stock: e.target.value })}
                required
              />
            </div>
            {attrs.length > 0 && (
              <div className="space-y-3">
                <label className="text-xs font-medium text-zinc-400">Attribute Values</label>
                {attrs.map((a) => (
                  <Select
                    key={a.id} label={a.name}
                    value={variantAttrValues[a.name] ?? ''}
                    onChange={(e) => setVariantAttrValues({ ...variantAttrValues, [a.name]: e.target.value })}
                  >
                    <option value="">Select {a.name}</option>
                    {a.values.map((v) => <option key={v} value={v}>{v}</option>)}
                  </Select>
                ))}
              </div>
            )}
            <div className="flex flex-col-reverse justify-end gap-2 pt-3 sm:flex-row">
              <Button type="button" variant="ghost" onClick={closeVariantModal}>Cancel</Button>
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
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Products</h1>
          <p className="text-zinc-500 text-sm mt-1">{products.length} products</p>
        </div>
        <Button onClick={openCreate}><Plus size={15} /> Add Product</Button>
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
              <tr><td colSpan={8}><PageSpinner /></td></tr>
            ) : products.length === 0 ? (
              <EmptyRow cols={8} message="No products yet" icon={<Package size={28} />} />
            ) : (
              products.map((p) => {
                const thumbURL = p.images?.[0]?.url ?? p.image_url;
                return (
                  <Tr key={p.id}>
                    <Td>
                      <div className="flex items-center gap-3">
                        <div className="w-9 h-9 rounded-xl bg-white/5 ring-1 ring-white/[0.06] flex items-center justify-center shrink-0 overflow-hidden">
                          {thumbURL ? (
                            <img src={thumbURL} alt={p.name} className="w-full h-full object-cover" />
                          ) : (
                            <Package size={14} className="text-zinc-400" />
                          )}
                        </div>
                        <div>
                          <p className="font-medium text-zinc-200">{p.name}</p>
                          {p.description && (
                            <p className="text-xs text-zinc-600 truncate max-w-[200px]">{p.description}</p>
                          )}
                        </div>
                      </div>
                    </Td>
                    <Td className="text-zinc-500">{p.category || '--'}</Td>
                    <Td className="font-semibold tabular-nums">{formatMoney(p.price, p.currency)}</Td>
                    <Td>
                      <span className={p.stock === 0 ? 'text-red-400 font-medium' : p.stock < 10 ? 'text-amber-400' : 'text-zinc-300'}>
                        {p.stock}
                      </span>
                    </Td>
                    <Td><Badge status={p.status} /></Td>
                    <Td>
                      <button
                        onClick={() => openVariants(p)}
                        className="text-xs text-indigo-400 hover:text-indigo-300 cursor-pointer flex items-center gap-1"
                      >
                        <Layers size={12} />
                        {(p.attributes?.length ?? 0) > 0
                          ? p.attributes!.map((a) => a.name).join(', ')
                          : 'Manage'
                        }
                      </button>
                    </Td>
                    <Td className="text-xs text-zinc-500">{formatDate(p.created_at)}</Td>
                    <Td>
                      <div className="flex gap-1">
                        <Button variant="ghost" size="sm" onClick={() => openEdit(p)}>
                          <Pencil size={13} />
                        </Button>
                        <Button
                          variant="danger" size="sm"
                          loading={deleteMutation.isPending}
                          onClick={() => deleteMutation.mutate(p.id)}
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
              <Input label="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
            </div>
            <div className="sm:col-span-2">
              <Textarea
                label="Description"
                rows={3}
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
              />
            </div>
            {!editing && (
              <div className="sm:col-span-2">
                <Input label="SKU" value={form.sku} onChange={(e) => setForm({ ...form, sku: e.target.value })} placeholder="Auto-generated if empty" />
              </div>
            )}
            <Input
              label={`Price (${form.currency})`}
              type="number" step="0.01" min="0.01" placeholder="0.00"
              value={form.price}
              onChange={(e) => setForm({ ...form, price: e.target.value })}
              required
            />
            <Input
              label="Stock" type="number" min="0"
              value={form.stock}
              onChange={(e) => setForm({ ...form, stock: e.target.value })}
              required
            />
            <Input label="Category" value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} />
            <Select label="Currency" value={form.currency} onChange={(e) => setForm({ ...form, currency: e.target.value })}>
              <option value="USD">USD</option>
              <option value="EUR">EUR</option>
              <option value="GBP">GBP</option>
            </Select>
          </div>

          <div>
            <label className="text-xs font-medium text-zinc-400 mb-2 block">
              Images{localImages.length > 0 && <span className="ml-1 text-zinc-600">({localImages.length})</span>}
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
              {localImages.map((img, idx) => (
                <div
                  key={img.key}
                  className="relative w-20 h-20 rounded-xl overflow-hidden ring-1 ring-white/[0.08] bg-white/[0.02] shrink-0"
                >
                  <img src={img.previewURL} alt={`Image ${idx + 1}`} className="w-full h-full object-cover" />
                  {idx === 0 && (
                    <div className="absolute bottom-0 left-0 right-0 bg-black/70 text-[9px] text-zinc-300 text-center py-0.5">
                      Main
                    </div>
                  )}
                  <button
                    type="button"
                    onClick={() => removeImage(img.key)}
                    className="absolute top-1 right-1 w-5 h-5 flex items-center justify-center rounded bg-black/70 text-zinc-300 hover:text-white transition-colors"
                  >
                    <X size={11} />
                  </button>
                </div>
              ))}
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="w-20 h-20 rounded-xl ring-1 ring-dashed ring-white/[0.12] bg-white/[0.02] flex flex-col items-center justify-center gap-1 text-zinc-500 hover:text-zinc-300 hover:ring-white/25 transition-colors shrink-0"
              >
                <ImagePlus size={16} />
                <span className="text-[10px]">Add</span>
              </button>
            </div>
            {localImages.length === 0 && (
              <p className="text-[11px] text-zinc-600 mt-1.5">First image becomes the thumbnail. Add up to 10 images.</p>
            )}
          </div>

          <div>
            <label className="text-xs font-medium text-zinc-400 mb-2 block">Attributes</label>
            {globalAttrs.length === 0 ? (
              <p className="text-xs text-zinc-600">No global attributes defined yet. Ask an admin to create some.</p>
            ) : (
              <div className="space-y-2">
                {globalAttrs.map((ga) => {
                  const selected = selectedAttrs.find((a) => a.name === ga.name);
                  return (
                    <div key={ga.id} className="rounded-xl ring-1 ring-white/[0.06] bg-white/[0.02] p-3">
                      <label className="flex items-center gap-2 cursor-pointer mb-2">
                        <button
                          type="button"
                          onClick={() => toggleGlobalAttr(ga)}
                          className={`w-4 h-4 rounded border flex items-center justify-center transition-colors ${
                            selected ? 'bg-indigo-600 border-indigo-600' : 'border-zinc-600 hover:border-zinc-400'
                          }`}
                        >
                          {selected && <Check size={10} className="text-white" />}
                        </button>
                        <span className="text-sm font-medium text-zinc-200">{ga.name}</span>
                      </label>
                      {selected && (
                        <div className="flex flex-wrap gap-1.5 ml-6">
                          {ga.values.map((v) => {
                            const isChosen = selected.selectedValues.includes(v);
                            return (
                              <button
                                key={v}
                                type="button"
                                onClick={() => toggleAttrValue(ga.name, v)}
                                className={`px-2.5 py-1 rounded-lg text-xs transition-colors ${
                                  isChosen
                                    ? 'bg-indigo-500/20 ring-1 ring-indigo-500/30 text-indigo-300'
                                    : 'bg-white/[0.04] ring-1 ring-white/[0.06] text-zinc-500 hover:text-zinc-300'
                                }`}
                              >
                                {v}
                              </button>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {formError && (
            <p className="text-sm text-red-400 bg-red-500/10 rounded-lg px-4 py-3">{formError}</p>
          )}
          <div className="flex flex-col-reverse justify-end gap-2 pt-2 sm:flex-row">
            <Button type="button" variant="ghost" onClick={closeModal}>Cancel</Button>
            <Button type="submit" loading={isSaving}>{editing ? 'Save Changes' : 'Create Product'}</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
