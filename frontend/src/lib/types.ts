export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  user: User;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
}

export type OrderStatus = 'pending' | 'confirmed' | 'paid' | 'cancelled' | 'refunded';

export interface OrderItem {
  product_id: string;
  variant_id?: string;
  variant_sku?: string;
  variant_label?: string;
  quantity: number;
  price: number;
}

export interface Order {
  id: string;
  user_id: string;
  store_id?: string;
  status: OrderStatus;
  total_amount: number;
  currency: string;
  idempotency_key: string;
  items: OrderItem[];
  created_at: string;
  updated_at: string;
}

export type PaymentStatus = 'pending' | 'processing' | 'succeeded' | 'failed' | 'refunded';

export interface Payment {
  id: string;
  order_id: string;
  user_id: string;
  amount: number;
  currency: string;
  status: PaymentStatus;
  method: string;
  transaction_id?: string;
  failure_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface Wallet {
  id: string;
  user_id: string;
  balance: number;
  currency: string;
  created_at: string;
  updated_at: string;
}

export interface WalletTransaction {
  id: string;
  wallet_id: string;
  type: 'credit' | 'debit';
  source: string;
  reference_id: string;
  amount: number;
  balance_before: number;
  balance_after: number;
  created_at: string;
}

export type ProductStatus = 'active' | 'inactive' | 'out_of_stock';

export interface ProductAttribute {
  id: string;
  product_id: string;
  name: string;
  values: string[];
  position: number;
  created_at: string;
}

export interface ProductVariant {
  id: string;
  product_id: string;
  sku: string;
  price: number | null;
  stock: number;
  attribute_values: Record<string, string>;
  status: ProductStatus;
  created_at: string;
  updated_at: string;
}

export interface ProductImage {
  id: string;
  product_id: string;
  url: string;
  position: number;
  created_at: string;
}

export interface Product {
  id: string;
  store_id?: string;
  name: string;
  description: string;
  sku: string;
  price: number;
  currency: string;
  stock: number;
  status: ProductStatus;
  category: string;
  image_url?: string;    // first image (thumbnail) â€” kept for backward compat
  images?: ProductImage[]; // full gallery
  attributes?: ProductAttribute[];
  variants?: ProductVariant[];
  created_at: string;
  updated_at: string;
}

export interface GlobalAttribute {
  id: string;
  name: string;
  values: string[];
  position: number;
  created_at: string;
  updated_at: string;
}

export type StoreStatus = 'pending' | 'active' | 'suspended';

export interface Store {
  id: string;
  owner_id: string;
  name: string;
  description: string;
  status: StoreStatus;
  commission: number;
  created_at: string;
  updated_at: string;
}

export interface CartItem {
  product_id: string;
  variant_id?: string;
  variant_label?: string;
  variant_sku?: string;
  name: string;
  price: number;
  currency: string;
  quantity: number;
  store_id?: string;
}

export interface Cart {
  items: CartItem[];
  total: number;
  currency: string;
}

export type WithdrawalStatus = 'pending' | 'approved' | 'rejected';

export interface Withdrawal {
  id: string;
  user_id: string;
  store_id: string;
  amount: number;
  currency: string;
  method: string;
  status: WithdrawalStatus;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface Notification {
  id: string;
  user_id: string;
  type: string;
  title: string;
  body: string;
  read: boolean;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface FraudCheck {
  id: string;
  payment_id: string;
  order_id: string;
  user_id: string;
  amount: number;
  currency: string;
  risk_score: number;
  decision: 'approved' | 'review' | 'rejected';
  rules: string[];
  created_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  limit: number;
  offset: number;
  total?: number;
}
