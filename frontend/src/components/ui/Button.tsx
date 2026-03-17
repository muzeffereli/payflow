import { cn } from '../../lib/utils';
import type { ButtonHTMLAttributes, ReactNode } from 'react';

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'success';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  children: ReactNode;
}

const variants = {
  primary:
    'bg-indigo-600 hover:bg-indigo-500 text-white shadow-sm shadow-indigo-600/20',
  secondary:
    'bg-[var(--color-surface)] hover:bg-[var(--color-hover)] text-[var(--color-text)] ring-1 ring-[var(--color-border)] hover:ring-[var(--color-border-h)]',
  ghost:
    'bg-transparent hover:bg-[var(--color-hover)] text-[var(--color-text-soft)] hover:text-[var(--color-text)]',
  danger:
    'bg-red-500/10 hover:bg-red-500/20 text-red-400 ring-1 ring-red-500/20',
  success:
    'bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 ring-1 ring-emerald-500/20',
};

const sizes = {
  sm: 'h-7 px-2.5 text-xs gap-1.5 rounded-lg',
  md: 'h-9 px-3.5 text-sm gap-2 rounded-lg',
  lg: 'h-11 px-5 text-sm gap-2 rounded-xl',
};

export function Button({
  variant = 'primary',
  size = 'md',
  loading,
  className,
  disabled,
  children,
  ...props
}: Props) {
  return (
    <button
      {...props}
      disabled={disabled || loading}
      className={cn(
        'inline-flex items-center justify-center font-medium',
        'transition-all duration-150 cursor-pointer',
        'disabled:opacity-40 disabled:cursor-not-allowed disabled:shadow-none',
        'active:scale-[0.97]',
        variants[variant],
        sizes[size],
        className,
      )}
    >
      {loading && (
        <svg className="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
        </svg>
      )}
      {children}
    </button>
  );
}
