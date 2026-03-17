import { Moon, Sun } from 'lucide-react';
import { useThemeStore } from '../lib/theme';
import { cn } from '../lib/utils';

interface ThemeToggleProps {
  className?: string;
}

export function ThemeToggle({ className }: ThemeToggleProps) {
  const theme = useThemeStore((state) => state.theme);
  const toggleTheme = useThemeStore((state) => state.toggleTheme);

  const isDark = theme === 'dark';
  const nextMode = isDark ? 'Light' : 'Dark';

  return (
    <button
      type="button"
      onClick={toggleTheme}
      className={cn(
        'inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-medium transition-colors',
        'border-[var(--color-border)] bg-[var(--color-card)] text-[var(--color-text)] hover:border-[var(--color-border-h)] hover:bg-[var(--color-surface)]',
        className,
      )}
      aria-label={`Switch to ${isDark ? 'light' : 'dark'} mode`}
      title={`Switch to ${isDark ? 'light' : 'dark'} mode`}
    >
      <span
        className={cn(
          'inline-flex h-7 w-7 items-center justify-center rounded-lg transition-colors',
          isDark ? 'bg-indigo-500/15 text-indigo-300' : 'bg-amber-500/15 text-amber-500',
        )}
      >
        {isDark ? <Sun size={15} /> : <Moon size={15} />}
      </span>
      <span>{nextMode} mode</span>
    </button>
  );
}
