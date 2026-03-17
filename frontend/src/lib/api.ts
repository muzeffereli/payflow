import axios from 'axios';

const api = axios.create({
  baseURL: '',
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      localStorage.removeItem('user');
      window.location.href = '/login';
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
  logout: (refresh_token: string) =>
    api.post('/auth/logout', { refresh_token }),
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
  list: (params?: { store_id?: string; limit?: number; offset?: number }) =>
    api.get('/api/v1/products', { params: { limit: 20, offset: 0, ...params } }),
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
  list: () => api.get('/api/v1/attributes'),
  create: (data: { name: string; values: string[] }) =>
    api.post('/api/v1/admin/attributes', data),
  update: (id: string, data: { name?: string; values?: string[] }) =>
    api.patch(`/api/v1/admin/attributes/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/admin/attributes/${id}`),
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
