import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatMoney(cents: number, currency = 'USD'): string {
  const code = currency || 'USD';
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: code,
    minimumFractionDigits: 2,
  }).format(cents / 100);
}

export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso));
}

export function formatDateShort(iso: string): string {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  }).format(new Date(iso));
}

export function generateKey(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function formatVariantLabel(values?: Record<string, string>): string {
  if (!values) return '';
  const entries = Object.entries(values).sort(([a], [b]) => a.localeCompare(b));
  return entries.map(([key, value]) => `${key}: ${value}`).join(' / ');
}

export const STATUS_COLORS: Record<string, string> = {
  pending:     'bg-amber-500/10 text-amber-400',
  confirmed:   'bg-blue-500/10 text-blue-400',
  paid:        'bg-emerald-500/10 text-emerald-400',
  cancelled:   'bg-red-500/10 text-red-400',
  refunded:    'bg-purple-500/10 text-purple-400',
  processing:  'bg-blue-500/10 text-blue-400',
  succeeded:   'bg-emerald-500/10 text-emerald-400',
  failed:      'bg-red-500/10 text-red-400',
  active:      'bg-emerald-500/10 text-emerald-400',
  suspended:   'bg-red-500/10 text-red-400',
  inactive:    'bg-zinc-500/10 text-zinc-400',
  out_of_stock:'bg-orange-500/10 text-orange-400',
  approved:    'bg-emerald-500/10 text-emerald-400',
  rejected:    'bg-red-500/10 text-red-400',
  _default:    'bg-zinc-500/10 text-zinc-500',
};
