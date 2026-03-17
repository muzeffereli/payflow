import { useState } from 'react';
import { Outlet, Navigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Menu, CreditCard, ShoppingCart } from 'lucide-react';
import { Sidebar } from './Sidebar';
import { CartDrawer } from './CartDrawer';
import { NotificationBell } from './NotificationBell';
import { ThemeToggle } from './ThemeToggle';
import { useAuthStore, useUser } from '../lib/store';
import { cartApi } from '../lib/api';

export function Layout() {
  const { isAuthenticated } = useAuthStore();
  const user = useUser();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [cartOpen, setCartOpen] = useState(false);

  const { data: cartData } = useQuery({
    queryKey: ['cart', user?.id],
    queryFn: () => cartApi.get(),
    enabled: !!user?.id,
    refetchInterval: 60_000,
  });

  const cartCount = (cartData?.data?.items ?? []).reduce(
    (sum: number, it: { quantity: number }) => sum + it.quantity,
    0,
  );

  if (!isAuthenticated()) return <Navigate to="/login" replace />;

  return (
    <div className="flex min-h-screen">
      <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />

      <div className="fixed top-0 left-0 right-0 z-30 h-14 flex items-center gap-2.5 px-4 border-b border-white/[0.06] bg-[var(--color-bg)] md:hidden">
        <button
          onClick={() => setSidebarOpen(true)}
          className="p-1.5 rounded-md text-zinc-400 hover:text-zinc-200 hover:bg-white/[0.06]"
        >
          <Menu size={20} />
        </button>
        <div className="w-6 h-6 rounded-md bg-indigo-600 flex items-center justify-center">
          <CreditCard size={11} className="text-white" />
        </div>
        <span className="text-sm font-semibold text-zinc-100 flex-1">PayFlow</span>
        <ThemeToggle className="hidden sm:inline-flex" />
        <NotificationBell />
        <button
          onClick={() => setCartOpen(true)}
          className="relative p-1.5 rounded-md text-zinc-400 hover:text-zinc-200 hover:bg-white/[0.06]"
        >
          <ShoppingCart size={20} />
          {cartCount > 0 && (
            <span className="absolute -top-1 -right-1 min-w-[18px] h-[18px] flex items-center justify-center bg-indigo-600 text-white text-[10px] font-bold rounded-full px-1">
              {cartCount}
            </span>
          )}
        </button>
      </div>

      <main className="flex-1 min-w-0 pt-14 md:pt-0 main-content">
        <div className="max-w-[1100px] mx-auto px-4 py-6 md:px-8 md:pb-8 md:pt-24">
          <Outlet />
        </div>
      </main>

      <div className="hidden md:flex fixed top-5 right-6 z-30">
        <div className="flex items-center gap-3">
          <ThemeToggle />
          <NotificationBell />
        </div>
      </div>
      <button
        onClick={() => setCartOpen(true)}
        className="hidden md:flex fixed bottom-6 right-6 z-30 items-center gap-2 px-4 py-3 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white shadow-lg shadow-indigo-600/25 transition-all duration-150 active:scale-95 cursor-pointer"
      >
        <ShoppingCart size={18} />
        {cartCount > 0 && (
          <span className="min-w-[20px] h-5 flex items-center justify-center bg-white/20 text-xs font-bold rounded-full px-1.5">
            {cartCount}
          </span>
        )}
      </button>

      <CartDrawer open={cartOpen} onClose={() => setCartOpen(false)} />
    </div>
  );
}
