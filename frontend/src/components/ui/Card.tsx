import { cn } from '../../lib/utils';
import type { HTMLAttributes, ReactNode } from 'react';

interface Props extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  className?: string;
  hover?: boolean;
}

export function Card({ children, className, hover, ...props }: Props) {
  return (
    <div
      {...props}
      className={cn(
        'rounded-xl bg-[var(--color-card)] ring-1 ring-[var(--color-border)]',
        hover && 'transition-all duration-200 hover:ring-[var(--color-border-h)] hover:bg-[var(--color-surface)]',
        className,
      )}
    >
      {children}
    </div>
  );
}

export function CardHeader({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('px-5 py-3.5 border-b border-[var(--color-border)]', className)}>
      {children}
    </div>
  );
}

export function CardBody({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cn('p-5', className)}>{children}</div>;
}

interface StatCardProps {
  title: string;
  value: string | number;
  icon: ReactNode;
  trend?: { value: string; positive: boolean };
  color?: string;
}

export function StatCard({ title, value, icon, trend, color = 'indigo' }: StatCardProps) {
  const colorMap: Record<string, { bg: string; icon: string }> = {
    indigo: { bg: 'bg-indigo-500/5 ring-indigo-500/10', icon: 'bg-indigo-500/10 text-indigo-400' },
    green:  { bg: 'bg-emerald-500/5 ring-emerald-500/10', icon: 'bg-emerald-500/10 text-emerald-400' },
    amber:  { bg: 'bg-amber-500/5 ring-amber-500/10', icon: 'bg-amber-500/10 text-amber-400' },
    blue:   { bg: 'bg-blue-500/5 ring-blue-500/10', icon: 'bg-blue-500/10 text-blue-400' },
    purple: { bg: 'bg-purple-500/5 ring-purple-500/10', icon: 'bg-purple-500/10 text-purple-400' },
    rose:   { bg: 'bg-rose-500/5 ring-rose-500/10', icon: 'bg-rose-500/10 text-rose-400' },
  };

  const c = colorMap[color] ?? colorMap.indigo;

  return (
    <div
      className={cn(
        'rounded-xl p-5 ring-1 transition-all duration-200 hover:ring-opacity-20',
        c.bg,
      )}
    >
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <p className="text-xs font-medium text-zinc-500">{title}</p>
          <p className="text-2xl font-semibold text-zinc-100 tracking-tight tabular-nums">{value}</p>
          {trend && (
            <p
              className={cn(
                'text-xs font-medium',
                trend.positive ? 'text-emerald-400' : 'text-red-400',
              )}
            >
              {trend.positive ? 'â†‘' : 'â†“'} {trend.value}
            </p>
          )}
        </div>
        <div className={cn('p-2.5 rounded-lg', c.icon)}>
          {icon}
        </div>
      </div>
    </div>
  );
}
