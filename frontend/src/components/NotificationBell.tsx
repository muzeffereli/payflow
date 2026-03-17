import { useState, useRef, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Bell, Check, CheckCheck } from 'lucide-react';
import { notificationsApi } from '../lib/api';
import { cn, formatDate } from '../lib/utils';
import { useNavigate } from 'react-router-dom';
import type { Notification } from '../lib/types';

export function NotificationBell() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const { data: countData } = useQuery({
    queryKey: ['notifications-unread'],
    queryFn: () => notificationsApi.unreadCount(),
    refetchInterval: 30_000,
  });

  const { data: listData } = useQuery({
    queryKey: ['notifications-recent'],
    queryFn: () => notificationsApi.list(8, 0),
    enabled: open,
  });

  const markReadMutation = useMutation({
    mutationFn: (id: string) => notificationsApi.markRead(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications-unread'] });
      qc.invalidateQueries({ queryKey: ['notifications-recent'] });
    },
  });

  const markAllMutation = useMutation({
    mutationFn: () => notificationsApi.markAllRead(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications-unread'] });
      qc.invalidateQueries({ queryKey: ['notifications-recent'] });
    },
  });

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    if (open) document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  const unread = countData?.data?.unread_count ?? 0;
  const notifications: Notification[] = listData?.data?.notifications ?? [];

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setOpen(!open)}
        className="relative p-1.5 rounded-md text-zinc-400 hover:text-zinc-200 hover:bg-white/[0.06] transition-colors cursor-pointer"
      >
        <Bell size={20} />
        {unread > 0 && (
          <span className="absolute -top-1 -right-1 min-w-[18px] h-[18px] flex items-center justify-center bg-indigo-600 text-white text-[10px] font-bold rounded-full px-1">
            {unread > 99 ? '99+' : unread}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-2 w-[360px] max-h-[480px] bg-[var(--color-surface)] border border-white/[0.06] rounded-xl shadow-2xl shadow-black/40 z-50 overflow-hidden animate-fade-in">
          <div className="px-4 py-3 border-b border-white/[0.06] flex items-center justify-between">
            <span className="text-sm font-semibold text-zinc-200">Notifications</span>
            <div className="flex items-center gap-2">
              {unread > 0 && (
                <button
                  onClick={() => markAllMutation.mutate()}
                  className="text-xs text-indigo-400 hover:text-indigo-300 cursor-pointer flex items-center gap-1"
                >
                  <CheckCheck size={12} /> Mark all read
                </button>
              )}
            </div>
          </div>

          <div className="overflow-y-auto max-h-[360px] divide-y divide-white/[0.04]">
            {notifications.length === 0 ? (
              <div className="py-12 text-center">
                <Bell size={24} className="text-zinc-700 mx-auto mb-2" />
                <p className="text-sm text-zinc-500">No notifications</p>
              </div>
            ) : (
              notifications.map((n) => (
                <div
                  key={n.id}
                  className={cn(
                    'px-4 py-3 hover:bg-white/[0.02] transition-colors cursor-pointer',
                    !n.read && 'bg-indigo-500/5',
                  )}
                  onClick={() => {
                    if (!n.read) markReadMutation.mutate(n.id);
                  }}
                >
                  <div className="flex items-start gap-3">
                    <div className={cn(
                      'w-2 h-2 rounded-full mt-1.5 shrink-0',
                      n.read ? 'bg-transparent' : 'bg-indigo-500',
                    )} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-zinc-200">{n.title}</p>
                      <p className="text-xs text-zinc-500 mt-0.5 line-clamp-2">{n.body}</p>
                      <p className="text-[11px] text-zinc-600 mt-1">{formatDate(n.created_at)}</p>
                    </div>
                    {!n.read && (
                      <button
                        onClick={(e) => { e.stopPropagation(); markReadMutation.mutate(n.id); }}
                        className="p-1 text-zinc-600 hover:text-indigo-400 cursor-pointer"
                      >
                        <Check size={12} />
                      </button>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>

          <div className="px-4 py-2.5 border-t border-white/[0.06]">
            <button
              onClick={() => { setOpen(false); navigate('/notifications'); }}
              className="text-xs text-indigo-400 hover:text-indigo-300 cursor-pointer w-full text-center"
            >
              View all notifications
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
