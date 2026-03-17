import { cn } from '../../lib/utils';
import type { InputHTMLAttributes, ReactNode, TextareaHTMLAttributes } from 'react';

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  icon?: ReactNode;
}

export function Input({ label, error, icon, className, ...props }: Props) {
  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label className="text-xs font-medium text-zinc-500">{label}</label>
      )}
      <div className="relative group">
        {icon && (
          <div className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-600 group-focus-within:text-zinc-400 transition-colors duration-150">
            {icon}
          </div>
        )}
        <input
          {...props}
          className={cn(
            'w-full h-9 bg-[var(--color-input-bg)] ring-1 ring-[var(--color-border)] rounded-lg px-3 text-sm text-[var(--color-text-strong)]',
            'placeholder:text-[var(--color-text-muted)] outline-none',
            'focus:ring-indigo-500/50 focus:bg-[var(--color-surface)]',
            'transition-all duration-150',
            'hover:ring-[var(--color-border-h)]',
            'disabled:opacity-40',
            icon && 'pl-9',
            error && 'ring-red-500/50 focus:ring-red-500/50',
            className,
          )}
        />
      </div>
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  );
}

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
  children: ReactNode;
}

export function Select({ label, error, className, children, ...props }: SelectProps) {
  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label className="text-xs font-medium text-zinc-500">{label}</label>
      )}
      <select
        {...props}
        className={cn(
          'w-full h-9 bg-[var(--color-input-bg)] ring-1 ring-[var(--color-border)] rounded-lg px-3 text-sm text-[var(--color-text-strong)]',
          'outline-none focus:ring-indigo-500/50',
          'transition-all duration-150 hover:ring-[var(--color-border-h)]',
          error && 'ring-red-500/50',
          className,
        )}
      >
        {children}
      </select>
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  );
}

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
}

export function Textarea({ label, error, className, ...props }: TextareaProps) {
  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label className="text-xs font-medium text-zinc-500">{label}</label>
      )}
      <textarea
        {...props}
        className={cn(
          'w-full rounded-lg bg-[var(--color-input-bg)] px-3 py-2.5 text-sm text-[var(--color-text-strong)] ring-1 ring-[var(--color-border)]',
          'placeholder:text-[var(--color-text-muted)] outline-none',
          'focus:ring-indigo-500/50 focus:bg-[var(--color-surface)]',
          'transition-all duration-150 hover:ring-[var(--color-border-h)]',
          'disabled:opacity-40 resize-none',
          error && 'ring-red-500/50 focus:ring-red-500/50',
          className,
        )}
      />
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  );
}
