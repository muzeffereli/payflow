import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from 'sonner';
import type { ReactElement } from 'react';
import { Layout } from './components/Layout';
import { Login } from './pages/Login';
import { Register } from './pages/Register';
import { Dashboard } from './pages/Dashboard';
import { Orders } from './pages/Orders';
import { Payments } from './pages/Payments';
import { Wallet } from './pages/Wallet';
import { Products } from './pages/Products';
import { Stores } from './pages/Stores';
import { Shop } from './pages/Shop';
import { ProductDetail } from './pages/ProductDetail';
import { Withdrawals } from './pages/Withdrawals';
import { AdminWithdrawals } from './pages/AdminWithdrawals';
import { AdminAttributes } from './pages/AdminAttributes';
import { AdminCategories } from './pages/AdminCategories';
import { AdminFraudAlerts } from './pages/AdminFraudAlerts';
import { AdminUsers } from './pages/AdminUsers';
import { AdminStores } from './pages/AdminStores';
import { AdminDashboard } from './pages/AdminDashboard';
import { Notifications } from './pages/Notifications';
import { NotFound } from './pages/NotFound';
import { Profile } from './pages/Profile';
import { useAuthStore } from './lib/store';
import { useThemeStore } from './lib/theme';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1 },
  },
});

export default function App() {
  const theme = useThemeStore((state) => state.theme);

  return (
    <QueryClientProvider client={queryClient}>
      <Toaster
        theme={theme}
        position="top-right"
        toastOptions={{
          style: {
            background: 'var(--color-card)',
            border: '1px solid var(--color-border)',
            color: 'var(--color-text-strong)',
          },
        }}
      />
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />

          <Route element={<Layout />}>
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/orders" element={<Orders />} />
            <Route path="/payments" element={<Payments />} />
            <Route path="/wallet" element={<Wallet />} />
            <Route
              path="/products"
              element={
                <RoleRoute allowedRoles={['seller']}>
                  <Products />
                </RoleRoute>
              }
            />
            <Route
              path="/stores"
              element={
                <RoleRoute allowedRoles={['seller']}>
                  <Stores />
                </RoleRoute>
              }
            />
            <Route path="/shop" element={<Shop />} />
            <Route path="/shop/:id" element={<ProductDetail />} />
            <Route
              path="/withdrawals"
              element={
                <RoleRoute allowedRoles={['seller']}>
                  <Withdrawals />
                </RoleRoute>
              }
            />
            <Route path="/notifications" element={<Notifications />} />
            <Route
              path="/admin/withdrawals"
              element={
                <RoleRoute allowedRoles={['admin']}>
                  <AdminWithdrawals />
                </RoleRoute>
              }
            />
            <Route
              path="/admin/categories"
              element={
                <RoleRoute allowedRoles={['admin']}>
                  <AdminCategories />
                </RoleRoute>
              }
            />
            <Route
              path="/admin/attributes"
              element={
                <RoleRoute allowedRoles={['admin']}>
                  <AdminAttributes />
                </RoleRoute>
              }
            />
            <Route
              path="/admin/fraud"
              element={
                <RoleRoute allowedRoles={['admin']}>
                  <AdminFraudAlerts />
                </RoleRoute>
              }
            />
            <Route
              path="/admin/users"
              element={
                <RoleRoute allowedRoles={['admin']}>
                  <AdminUsers />
                </RoleRoute>
              }
            />
            <Route
              path="/admin/stores"
              element={
                <RoleRoute allowedRoles={['admin']}>
                  <AdminStores />
                </RoleRoute>
              }
            />
            <Route
              path="/admin/dashboard"
              element={
                <RoleRoute allowedRoles={['admin']}>
                  <AdminDashboard />
                </RoleRoute>
              }
            />
            <Route path="/profile" element={<Profile />} />
          </Route>

          <Route path="*" element={<NotFound />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

function RoleRoute({
  allowedRoles,
  children,
}: {
  allowedRoles: string[];
  children: ReactElement;
}) {
  const user = useAuthStore((state) => state.user);

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  if (!allowedRoles.includes(user.role)) {
    return <Navigate to="/dashboard" replace />;
  }

  return children;
}
