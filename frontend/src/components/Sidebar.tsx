import { NavLink, useNavigate, useLocation } from 'react-router-dom';
import { useEffect } from 'react';
import {
  LayoutDashboard, ShoppingBag, Package, Wallet,
  Store, CreditCard, LogOut, Bell,
  ShieldCheck, ArrowDownToLine, X, Tags, ShieldAlert, Users, Gauge, UserCircle, FolderTree,
} from 'lucide-react';
import { cn } from '../lib/utils';
import { useAuthStore, useIsAdmin, useIsSeller } from '../lib/store';
import { authApi } from '../lib/api';
import { ThemeToggle } from './ThemeToggle';

interface NavItem { to: string; icon: React.ReactNode; label: string }

interface SidebarProps {
  open: boolean;
  onClose: () => void;
}

export function Sidebar({ open, onClose }: SidebarProps) {
  const { user, clearAuth } = useAuthStore();
  const isAdmin = useIsAdmin();
  const isSeller = useIsSeller();
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    onClose();
  }, [location.pathname, onClose]);

  const navItems: NavItem[] = [
    { to: '/dashboard', icon: <LayoutDashboard size={16} />, label: 'Dashboard' },
    { to: '/shop',      icon: <ShoppingBag size={16} />,     label: 'Shop' },
    { to: '/orders',    icon: <Package size={16} />,         label: 'Orders' },
    { to: '/payments',  icon: <CreditCard size={16} />,      label: 'Payments' },
    { to: '/wallet',    icon: <Wallet size={16} />,          label: 'Wallet' },
    { to: '/notifications', icon: <Bell size={16} />,        label: 'Notifications' },
    { to: '/profile',       icon: <UserCircle size={16} />,   label: 'Profile' },
  ];

  const sellerItems: NavItem[] = [
    { to: '/stores',      icon: <Store size={16} />,          label: 'My Stores' },
    { to: '/products',    icon: <Package size={16} />,        label: 'Products' },
    { to: '/withdrawals', icon: <ArrowDownToLine size={16} />,label: 'Withdrawals' },
  ];

  const adminItems: NavItem[] = [
    { to: '/admin/dashboard', icon: <Gauge size={16} />, label: 'Dashboard' },
    { to: '/admin/users', icon: <Users size={16} />, label: 'Users' },
    { to: '/admin/stores', icon: <Store size={16} />, label: 'Stores' },
    { to: '/admin/categories', icon: <FolderTree size={16} />, label: 'Categories' },
    { to: '/admin/attributes', icon: <Tags size={16} />, label: 'Attributes' },
    { to: '/admin/fraud', icon: <ShieldAlert size={16} />, label: 'Fraud Alerts' },
    { to: '/admin/withdrawals', icon: <ShieldCheck size={16} />, label: 'Withdrawals' },
  ];

  const handleLogout = async () => {
    try { await authApi.logout(); } catch { /* best-effort server invalidation */ }
    clearAuth();
    navigate('/login');
  };

  return (
    <>
      {open && (
        <div
          className="fixed inset-0 z-40 bg-black/60 md:hidden"
          onClick={onClose}
        />
      )}

      <aside
        className={cn(
          'fixed top-0 left-0 h-full flex flex-col z-50 border-r border-white/[0.06] bg-[var(--color-bg)] transition-transform duration-200',
          'md:translate-x-0',
          open ? 'translate-x-0' : '-translate-x-full',
        )}
        style={{ width: 'var(--sidebar-w)' }}
      >
        <div className="h-14 flex items-center justify-between px-5 shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-lg bg-indigo-600 flex items-center justify-center">
              <CreditCard size={13} className="text-white" />
            </div>
            <span className="text-sm font-semibold text-zinc-100 tracking-tight">PayFlow</span>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-md text-zinc-600 hover:text-zinc-300 md:hidden"
          >
            <X size={16} />
          </button>
        </div>

        <nav className="flex-1 px-3 py-3 overflow-y-auto space-y-5">
          <Section label="Menu">{navItems.map(i => <Link key={i.to} {...i} />)}</Section>
          {isSeller && <Section label="Seller">{sellerItems.map(i => <Link key={i.to} {...i} />)}</Section>}
          {isAdmin && <Section label="Admin">{adminItems.map(i => <Link key={i.to} {...i} />)}</Section>}
        </nav>

        <div className="p-3 border-t border-white/[0.06]">
          <ThemeToggle className="mb-3 w-full justify-center" />
          <div className="flex items-center gap-2.5 px-2 py-1.5">
            <div className="w-7 h-7 rounded-lg bg-indigo-600/20 flex items-center justify-center text-indigo-400 text-[11px] font-semibold shrink-0">
              {user?.name?.charAt(0).toUpperCase() ?? '?'}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-[13px] font-medium text-zinc-200 truncate">{user?.name}</p>
              <p className="text-[11px] text-zinc-600 truncate capitalize">{user?.role}</p>
            </div>
            <button
              onClick={handleLogout}
              className="p-1.5 rounded-md text-zinc-600 hover:text-red-400 hover:bg-red-500/10 transition-colors duration-150"
              title="Logout"
            >
              <LogOut size={14} />
            </button>
          </div>
        </div>
      </aside>
    </>
  );
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="text-[11px] font-medium text-zinc-600 px-2 mb-1.5">{label}</p>
      <div className="space-y-0.5">{children}</div>
    </div>
  );
}

function Link({ to, icon, label }: NavItem) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        cn(
          'flex items-center gap-2.5 px-2.5 py-[7px] rounded-lg text-[13px] transition-colors duration-150',
          isActive
            ? 'bg-white/[0.06] text-zinc-100 font-medium'
            : 'text-zinc-500 hover:text-zinc-300 hover:bg-white/[0.03]',
        )
      }
    >
      {icon}
      <span>{label}</span>
    </NavLink>
  );
}
