import axios from 'axios';
import { useAuthStore } from './store';

const api = axios.create({
  baseURL: '',
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true, // send httpOnly auth cookies on every request
});

let isRefreshing = false;

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config;
    if (
      error.response?.status === 401 &&
      !original._retry &&
      !original.url?.includes('/auth/refresh')
    ) {
      original._retry = true;
      if (!isRefreshing) {
        isRefreshing = true;
        try {
          await api.post('/auth/refresh'); // refresh_token cookie sent automatically
          isRefreshing = false;
          return api(original);
        } catch {
          isRefreshing = false;
          useAuthStore.getState().clearAuth();
          window.location.href = '/login';
        }
      }
    }
    return Promise.reject(error);
  },
);

export default api;

export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error ?? fallback;
  }
  return fallback;
}

export const authApi = {
  register: (data: { email: string; name: string; password: string }) =>
    api.post('/auth/register', data),
  login: (data: { email: string; password: string }) =>
    api.post('/auth/login', data),
  me: () => api.get('/auth/me'),
  logout: () => api.post('/auth/logout'),
};

export const ordersApi = {
  list: (limit = 20, offset = 0) =>
    api.get(`/api/v1/orders?limit=${limit}&offset=${offset}`),
  get: (id: string) => api.get(`/api/v1/orders/${id}`),
  create: (data: object, idempotencyKey: string) =>
    api.post('/api/v1/orders', data, {
      headers: { 'Idempotency-Key': idempotencyKey },
    }),
  cancel: (id: string) => api.delete(`/api/v1/orders/${id}`),
  sellerOrders: (storeId?: string, limit = 20, offset = 0) =>
    api.get(
      `/api/v1/seller/orders?${storeId ? `store_id=${storeId}&` : ''}limit=${limit}&offset=${offset}`,
    ),
  sellerAnalytics: (storeId: string) =>
    api.get(`/api/v1/seller/analytics?store_id=${storeId}`),
};

export const paymentsApi = {
  get: (id: string) => api.get(`/api/v1/payments/${id}`),
  getByOrder: (orderId: string) =>
    api.get(`/api/v1/payments/order/${orderId}`),
  refund: (id: string) => api.post(`/api/v1/payments/${id}/refund`),
};

export const walletApi = {
  get: () => api.get('/api/v1/wallet'),
  create: (currency = 'USD') =>
    api.post('/api/v1/wallet', { currency }, {
      headers: { 'Idempotency-Key': `create-wallet-${Date.now()}` },
    }),
  topup: (amount: number) =>
    api.post('/api/v1/wallet/topup', { amount }),
  transactions: (limit = 20, offset = 0) =>
    api.get(`/api/v1/wallet/transactions?limit=${limit}&offset=${offset}`),
};

export const productsApi = {
  list: (params?: {
    store_id?: string;
    category?: string;
    category_id?: string;
    subcategory_id?: string;
    status?: string;
    search?: string;
    limit?: number;
    offset?: number;
    attribute_filters?: Record<string, string[]>;
  }) => {
    const queryParams: Record<string, string | number> = {
      limit: params?.limit ?? 20,
      offset: params?.offset ?? 0,
    };
    if (params?.store_id) queryParams.store_id = params.store_id;
    if (params?.category) queryParams.category = params.category;
    if (params?.category_id) queryParams.category_id = params.category_id;
    if (params?.subcategory_id) queryParams.subcategory_id = params.subcategory_id;
    if (params?.status) queryParams.status = params.status;
    if (params?.search) queryParams.search = params.search;
    for (const [name, values] of Object.entries(params?.attribute_filters ?? {})) {
      if (values.length === 0) continue;
      queryParams[`attr.${name}`] = values.join(',');
    }
    return api.get('/api/v1/products', { params: queryParams });
  },
  get: (id: string) => api.get(`/api/v1/products/${id}`),
  create: (data: object) => api.post('/api/v1/products', data),
  update: (id: string, data: object) =>
    api.patch(`/api/v1/products/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/products/${id}`),
  uploadImage: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return api.post('/api/v1/products/upload-image', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
  listVariants: (productId: string) =>
    api.get(`/api/v1/products/${productId}/variants`),
  createVariant: (productId: string, data: object) =>
    api.post(`/api/v1/products/${productId}/variants`, data),
  updateVariant: (productId: string, variantId: string, data: object) =>
    api.patch(`/api/v1/products/${productId}/variants/${variantId}`, data),
  deleteVariant: (productId: string, variantId: string) =>
    api.delete(`/api/v1/products/${productId}/variants/${variantId}`),
};

export const attributesApi = {
  list: (params?: { subcategory_id?: string; category_id?: string }) => api.get('/api/v1/attributes', { params }),
  categories: () => api.get('/api/v1/attributes/categories'),
  create: (data: { subcategory_id: string; name: string; values: string[] }) =>
    api.post('/api/v1/admin/attributes', data),
  update: (id: string, data: { subcategory_id?: string; name?: string; values?: string[] }) =>
    api.patch(`/api/v1/admin/attributes/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/admin/attributes/${id}`),
};

export const categoriesApi = {
  list: () => api.get('/api/v1/categories'),
  listSubcategories: (categoryId: string) => api.get(`/api/v1/categories/${categoryId}/subcategories`),
  create: (data: { name: string }) => api.post('/api/v1/admin/categories', data),
  update: (id: string, data: { name?: string }) => api.patch(`/api/v1/admin/categories/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/admin/categories/${id}`),
  createSubcategory: (data: { category_id: string; name: string }) => api.post('/api/v1/admin/subcategories', data),
  updateSubcategory: (id: string, data: { category_id?: string; name?: string }) =>
    api.patch(`/api/v1/admin/subcategories/${id}`, data),
  deleteSubcategory: (id: string) => api.delete(`/api/v1/admin/subcategories/${id}`),
};

export const storesApi = {
  list: (limit = 20, offset = 0) =>
    api.get(`/api/v1/stores?limit=${limit}&offset=${offset}`),
  getMe: () => api.get('/api/v1/stores/me'),
  get: (id: string) => api.get(`/api/v1/stores/${id}`),
  create: (data: object) => api.post('/api/v1/stores', data),
  update: (id: string, data: object) =>
    api.patch(`/api/v1/stores/${id}`, data),
  approve: (id: string) => api.post(`/api/v1/stores/${id}/approve`),
  suspend: (id: string) => api.post(`/api/v1/stores/${id}/suspend`),
  reactivate: (id: string) => api.post(`/api/v1/stores/${id}/reactivate`),
};

export const cartApi = {
  get: () => api.get('/api/v1/cart'),
  addItem: (data: { product_id: string; variant_id?: string; quantity: number }) =>
    api.post('/api/v1/cart/items', data),
  updateQuantity: (productId: string, quantity: number, variantId?: string) =>
    api.patch(`/api/v1/cart/items/${productId}`, { quantity }, {
      params: variantId ? { variant_id: variantId } : undefined,
    }),
  removeItem: (productId: string, variantId?: string) =>
    api.delete(`/api/v1/cart/items/${productId}`, {
      params: variantId ? { variant_id: variantId } : undefined,
    }),
  checkout: (idempotencyKey: string, paymentMethod: 'card' | 'wallet' = 'card') =>
    api.post(
      '/api/v1/cart/checkout',
      { payment_method: paymentMethod },
      { headers: { 'Idempotency-Key': idempotencyKey } },
    ),
};

export const notificationsApi = {
  list: (limit = 20, offset = 0) =>
    api.get(`/api/v1/notifications?limit=${limit}&offset=${offset}`),
  unreadCount: () => api.get('/api/v1/notifications/unread-count'),
  markRead: (id: string) => api.patch(`/api/v1/notifications/${id}/read`),
  markAllRead: () => api.post('/api/v1/notifications/read-all'),
};

export const withdrawalsApi = {
  myList: (limit = 20, offset = 0) =>
    api.get(`/api/v1/seller/withdrawals?limit=${limit}&offset=${offset}`),
  request: (data: {
    store_id: string;
    amount: number;
    currency: string;
    method: string;
  }) => api.post('/api/v1/seller/withdrawals', data),
  pendingList: (limit = 20, offset = 0) =>
    api.get(`/api/v1/admin/withdrawals?limit=${limit}&offset=${offset}`),
  approve: (id: string) =>
    api.post(`/api/v1/admin/withdrawals/${id}/approve`),
  reject: (id: string, reason: string) =>
    api.post(`/api/v1/admin/withdrawals/${id}/reject`, { reason }),
};

export const adminApi = {
  listUsers: (limit = 20, offset = 0) =>
    api.get(`/api/v1/admin/users?limit=${limit}&offset=${offset}`),
  updateRole: (userId: string, role: string) =>
    api.patch(`/api/v1/admin/users/${userId}/role`, { role }),
};

export const fraudApi = {
  list: (params?: { decision?: string; limit?: number; offset?: number }) =>
    api.get('/api/v1/admin/fraud-checks', { params: { limit: 20, offset: 0, ...params } }),
  get: (id: string) => api.get(`/api/v1/admin/fraud-checks/${id}`),
};
