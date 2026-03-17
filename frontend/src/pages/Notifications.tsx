import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Bell, Check, CheckCheck, ChevronLeft, ChevronRight } from 'lucide-react';
import { notificationsApi } from '../lib/api';
import { Button } from '../components/ui/Button';
import { Card, CardHeader, CardBody } from '../components/ui/Card';
import { PageSpinner } from '../components/ui/Spinner';
import { cn, formatDate } from '../lib/utils';
import type { Notification } from '../lib/types';

const PAGE_SIZE = 20;

export function Notifications() {
  const qc = useQueryClient();
  const [page, setPage] = useState(0);

  const { data, isLoading } = useQuery({
    queryKey: ['notifications', page],
    queryFn: () => notificationsApi.list(PAGE_SIZE, page * PAGE_SIZE),
  });

  const markReadMutation = useMutation({
    mutationFn: (id: string) => notificationsApi.markRead(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications'] });
      qc.invalidateQueries({ queryKey: ['notifications-unread'] });
    },
  });

  const markAllMutation = useMutation({
    mutationFn: () => notificationsApi.markAllRead(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications'] });
      qc.invalidateQueries({ queryKey: ['notifications-unread'] });
    },
  });

  const notifications: Notification[] = data?.data?.notifications ?? [];
  const total = data?.data?.total ?? 0;
  const totalPages = Math.ceil(total / PAGE_SIZE);

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Notifications</h1>
          <p className="text-zinc-500 text-sm mt-1">{total} notifications</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => markAllMutation.mutate()} loading={markAllMutation.isPending}>
          <CheckCheck size={14} /> Mark All Read
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <div className="w-7 h-7 rounded-lg bg-indigo-500/10 flex items-center justify-center">
                <Bell size={14} className="text-indigo-400" />
              </div>
              <h2 className="text-sm font-semibold text-zinc-200">All Notifications</h2>
            </div>
            {totalPages > 1 && (
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                  disabled={page === 0}
                  className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-white/5 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
                >
                  <ChevronLeft size={14} />
                </button>
                <span className="text-xs text-zinc-500 tabular-nums">
                  {page + 1} / {totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                  disabled={page >= totalPages - 1}
                  className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-white/5 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
                >
                  <ChevronRight size={14} />
                </button>
              </div>
            )}
          </div>
        </CardHeader>
        <CardBody className="p-0">
          {isLoading ? (
            <div className="py-12"><PageSpinner /></div>
          ) : notifications.length === 0 ? (
            <div className="py-16 text-center">
              <div className="w-14 h-14 rounded-2xl bg-white/5 flex items-center justify-center mx-auto mb-4">
                <Bell size={24} className="text-zinc-600" />
              </div>
              <p className="text-zinc-500 text-sm">No notifications yet</p>
            </div>
          ) : (
            <div className="divide-y divide-white/[0.04]">
              {notifications.map((n) => (
                <div
                  key={n.id}
                  className={cn(
                    'px-6 py-4 flex items-start gap-4 hover:bg-white/[0.02] transition-colors',
                    !n.read && 'bg-indigo-500/5',
                  )}
                >
                  <div className={cn(
                    'w-2.5 h-2.5 rounded-full mt-1.5 shrink-0',
                    n.read ? 'bg-zinc-800' : 'bg-indigo-500',
                  )} />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <p className="text-sm font-medium text-zinc-200">{n.title}</p>
                        <p className="text-sm text-zinc-500 mt-1">{n.body}</p>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className="text-[11px] text-zinc-600">{formatDate(n.created_at)}</span>
                        {!n.read && (
                          <button
                            onClick={() => markReadMutation.mutate(n.id)}
                            className="p-1.5 rounded-md text-zinc-600 hover:text-indigo-400 hover:bg-indigo-500/10 transition-colors cursor-pointer"
                            title="Mark as read"
                          >
                            <Check size={13} />
                          </button>
                        )}
                      </div>
                    </div>
                    <span className="inline-block mt-2 text-[10px] uppercase tracking-wider font-medium text-zinc-600 bg-white/5 px-2 py-0.5 rounded">
                      {n.type}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
