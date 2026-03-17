import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';

export function Table({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className="overflow-x-auto">
      <table className={cn('w-full text-sm', className)}>{children}</table>
    </div>
  );
}

export function Thead({ children }: { children: ReactNode }) {
  return <thead>{children}</thead>;
}

export function Th({ children, className }: { children?: ReactNode; className?: string }) {
  return (
    <th
      className={cn(
        'px-5 py-3 text-left text-xs font-medium text-[var(--color-text-soft)]',
        'border-b border-[var(--color-border)]',
        className,
      )}
    >
      {children}
    </th>
  );
}

export function Tbody({ children }: { children: ReactNode }) {
  return <tbody className="divide-y divide-[var(--color-border)]">{children}</tbody>;
}

export function Tr({
  children,
  className,
  onClick,
}: {
  children: ReactNode;
  className?: string;
  onClick?: () => void;
}) {
  return (
    <tr
      onClick={onClick}
      className={cn(
        'transition-colors duration-150',
        'hover:bg-[var(--color-hover)]',
        onClick && 'cursor-pointer',
        className,
      )}
    >
      {children}
    </tr>
  );
}

export function Td({ children, className }: { children?: ReactNode; className?: string }) {
  return (
    <td className={cn('px-5 py-3.5 text-[var(--color-text)]', className)}>{children}</td>
  );
}

interface EmptyRowProps {
  cols: number;
  message?: string;
  icon?: ReactNode;
}
export function EmptyRow({ cols, message = 'No records found', icon }: EmptyRowProps) {
  return (
    <tr>
      <td colSpan={cols} className="px-5 py-16 text-center">
        {icon && <div className="flex justify-center mb-3 text-zinc-700">{icon}</div>}
        <p className="text-zinc-600 text-sm">{message}</p>
      </td>
    </tr>
  );
}
