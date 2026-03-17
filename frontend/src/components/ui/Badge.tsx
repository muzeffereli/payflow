import { cn, STATUS_COLORS } from '../../lib/utils';

interface Props {
  status: string;
  className?: string;
}

export function Badge({ status, className }: Props) {
  const color = STATUS_COLORS[status] ?? STATUS_COLORS._default;
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[11px] font-medium capitalize',
        color,
        className,
      )}
    >
      <span className="w-1 h-1 rounded-full bg-current" />
      {status.replace(/_/g, ' ')}
    </span>
  );
}
