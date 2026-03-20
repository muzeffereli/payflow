import { create } from 'zustand';
import type { User } from './types';

interface AuthState {
  user: User | null;
  setAuth: (user: User) => void;
  clearAuth: () => void;
  isAuthenticated: () => boolean;
}

const storedUser = localStorage.getItem('user');

export const useAuthStore = create<AuthState>((set, get) => ({
  user: storedUser ? JSON.parse(storedUser) : null,

  setAuth: (user) => {
    localStorage.setItem('user', JSON.stringify(user));
    set({ user });
  },

  clearAuth: () => {
    localStorage.removeItem('user');
    set({ user: null });
  },

  isAuthenticated: () => !!get().user,
}));

export const useUser = () => useAuthStore((s) => s.user);
export const useIsAdmin = () => useAuthStore((s) => s.user?.role === 'admin');
export const useIsSeller = () => useAuthStore((s) => s.user?.role === 'seller');
