import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ShieldAlert, ChevronLeft, ChevronRight } from 'lucide-react';
import { fraudApi } from '../lib/api';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Select } from '../components/ui/Input';
import { Table, Thead, Th, Tbody, Tr, Td, EmptyRow } from '../components/ui/Table';
import { PageSpinner } from '../components/ui/Spinner';
import { formatMoney, formatDate } from '../lib/utils';
import type { FraudCheck } from '../lib/types';

const LIMIT = 20;

function riskColor(score: number) {
  if (score >= 70) return 'text-red-400';
  if (score >= 40) return 'text-amber-400';
  return 'text-emerald-400';
}

function riskBar(score: number) {
  const bg =
    score >= 70 ? 'bg-red-500' : score >= 40 ? 'bg-amber-500' : 'bg-emerald-500';
  return (
    <div className="flex items-center gap-2">
      <div className="w-16 h-1.5 rounded-full bg-white/[0.06] overflow-hidden">
        <div className={`h-full rounded-full ${bg}`} style={{ width: `${score}%` }} />
      </div>
      <span className={`text-xs font-semibold tabular-nums ${riskColor(score)}`}>{score}</span>
    </div>
  );
}

export function AdminFraudAlerts() {
  const [decision, setDecision] = useState('');
  const [offset, setOffset] = useState(0);

  const { data, isLoading } = useQuery({
    queryKey: ['fraud-checks', decision, offset],
    queryFn: () =>
      fraudApi.list({
        ...(decision ? { decision } : {}),
        limit: LIMIT,
        offset,
      }),
  });

  const checks: FraudCheck[] = data?.data?.fraud_checks ?? [];
  const total: number = data?.data?.total ?? 0;
  const hasNext = offset + LIMIT < total;
  const hasPrev = offset > 0;

  const approved = checks.filter((c) => c.decision === 'approved').length;
  const review = checks.filter((c) => c.decision === 'review').length;
  const rejected = checks.filter((c) => c.decision === 'rejected').length;

  return (
    <div className="animate-fade-in">
      <div className="mb-8 flex items-center gap-4">
        <div className="w-11 h-11 rounded-xl bg-orange-500/10 ring-1 ring-orange-500/10 flex items-center justify-center">
          <ShieldAlert size={20} className="text-orange-400" />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">Fraud Alerts</h1>
          <p className="text-zinc-500 text-sm">Review fraud check results for all payments</p>
        </div>
      </div>

      <div className="grid grid-cols-4 gap-4 mb-8 stagger-children">
        <div className="bg-white/[0.03] ring-1 ring-white/[0.06] rounded-2xl p-5">
          <p className="text-[10px] text-zinc-500 mb-1 font-semibold uppercase tracking-wider">Total</p>
          <p className="text-2xl font-bold text-zinc-200 tabular-nums">{total}</p>
        </div>
        <div className="bg-emerald-500/5 ring-1 ring-emerald-500/10 rounded-2xl p-5">
          <p className="text-[10px] text-emerald-500 mb-1 font-semibold uppercase tracking-wider">Approved</p>
          <p className="text-2xl font-bold text-emerald-400 tabular-nums">{approved}</p>
        </div>
        <div className="bg-amber-500/5 ring-1 ring-amber-500/10 rounded-2xl p-5">
          <p className="text-[10px] text-amber-500 mb-1 font-semibold uppercase tracking-wider">Review</p>
          <p className="text-2xl font-bold text-amber-400 tabular-nums">{review}</p>
        </div>
        <div className="bg-red-500/5 ring-1 ring-red-500/10 rounded-2xl p-5">
          <p className="text-[10px] text-red-500 mb-1 font-semibold uppercase tracking-wider">Rejected</p>
          <p className="text-2xl font-bold text-red-400 tabular-nums">{rejected}</p>
        </div>
      </div>

      <div className="mb-4 flex items-center gap-3">
        <Select
          label=""
          value={decision}
          onChange={(e) => {
            setDecision(e.target.value);
            setOffset(0);
          }}
        >
          <option value="">All Decisions</option>
          <option value="approved">Approved</option>
          <option value="review">Review</option>
          <option value="rejected">Rejected</option>
        </Select>
      </div>

      <Card>
        <Table>
          <Thead>
            <tr>
              <Th>ID</Th>
              <Th>Payment</Th>
              <Th>Order</Th>
              <Th>Amount</Th>
              <Th>Risk Score</Th>
              <Th>Decision</Th>
              <Th>Rules Triggered</Th>
              <Th>Date</Th>
            </tr>
          </Thead>
          <Tbody>
            {isLoading ? (
              <tr>
                <td colSpan={8}>
                  <PageSpinner />
                </td>
              </tr>
            ) : checks.length === 0 ? (
              <EmptyRow cols={8} message="No fraud checks found" icon={<ShieldAlert size={28} />} />
            ) : (
              checks.map((c) => (
                <Tr key={c.id}>
                  <Td>
                    <span className="font-mono text-xs text-zinc-500">#{c.id.slice(0, 8)}</span>
                  </Td>
                  <Td>
                    <span className="font-mono text-xs text-zinc-500">{c.payment_id.slice(0, 12)}...</span>
                  </Td>
                  <Td>
                    <span className="font-mono text-xs text-zinc-500">{c.order_id.slice(0, 12)}...</span>
                  </Td>
                  <Td className="font-semibold text-zinc-100 tabular-nums">
                    {formatMoney(c.amount, c.currency)}
                  </Td>
                  <Td>{riskBar(c.risk_score)}</Td>
                  <Td>
                    <Badge status={c.decision} />
                  </Td>
                  <Td>
                    <div className="flex flex-wrap gap-1">
                      {(c.rules ?? []).map((r) => (
                        <span
                          key={r}
                          className="inline-flex px-2 py-0.5 rounded-md bg-white/[0.04] ring-1 ring-white/[0.06] text-[10px] text-zinc-400"
                        >
                          {r}
                        </span>
                      ))}
                      {(!c.rules || c.rules.length === 0) && (
                        <span className="text-xs text-zinc-600">none</span>
                      )}
                    </div>
                  </Td>
                  <Td className="text-xs text-zinc-500">{formatDate(c.created_at)}</Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>

        {total > LIMIT && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-white/[0.06]">
            <p className="text-xs text-zinc-500">
              Showing {offset + 1}â€“{Math.min(offset + LIMIT, total)} of {total}
            </p>
            <div className="flex gap-1.5">
              <Button size="sm" variant="ghost" disabled={!hasPrev} onClick={() => setOffset(offset - LIMIT)}>
                <ChevronLeft size={14} /> Prev
              </Button>
              <Button size="sm" variant="ghost" disabled={!hasNext} onClick={() => setOffset(offset + LIMIT)}>
                Next <ChevronRight size={14} />
              </Button>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
