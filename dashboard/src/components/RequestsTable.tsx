import { useState, useEffect } from 'react';
import { getRecentRequests } from '../api';
import type { CostRequest } from '../api';

function statusBadge(status: string) {
  const base = 'px-2 py-0.5 rounded-full text-xs font-medium';
  if (status === 'success' || status === 'ok' || status === '200') {
    return <span className={`${base} bg-accent-green/15 text-accent-green`}>{status}</span>;
  }
  if (status === 'error' || status === 'failed' || status.startsWith('5')) {
    return <span className={`${base} bg-accent-red/15 text-accent-red`}>{status}</span>;
  }
  return <span className={`${base} bg-accent-amber/15 text-accent-amber`}>{status}</span>;
}

export default function RequestsTable() {
  const [requests, setRequests] = useState<CostRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getRecentRequests(50)
      .then(setRequests)
      .catch((err: unknown) => {
        const message = err instanceof Error ? err.message : 'Failed to load requests';
        setError(message);
      })
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="bg-bg-card border border-border rounded-xl p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-semibold text-text-primary">Recent Requests</h2>
        <span className="text-text-muted text-xs">Last 50 requests</span>
      </div>

      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="h-10 bg-bg-hover rounded animate-pulse" />
          ))}
        </div>
      ) : error ? (
        <div className="text-center py-8 text-accent-red text-sm">{error}</div>
      ) : requests.length === 0 ? (
        <div className="text-center py-8 text-text-muted text-sm">No requests recorded yet.</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left py-3 px-2 text-text-muted font-medium text-xs uppercase tracking-wider">Time</th>
                <th className="text-left py-3 px-2 text-text-muted font-medium text-xs uppercase tracking-wider">Model</th>
                <th className="text-left py-3 px-2 text-text-muted font-medium text-xs uppercase tracking-wider">Provider</th>
                <th className="text-left py-3 px-2 text-text-muted font-medium text-xs uppercase tracking-wider">Team</th>
                <th className="text-right py-3 px-2 text-text-muted font-medium text-xs uppercase tracking-wider">Tokens</th>
                <th className="text-right py-3 px-2 text-text-muted font-medium text-xs uppercase tracking-wider">Cost</th>
                <th className="text-right py-3 px-2 text-text-muted font-medium text-xs uppercase tracking-wider">Latency</th>
                <th className="text-center py-3 px-2 text-text-muted font-medium text-xs uppercase tracking-wider">Status</th>
              </tr>
            </thead>
            <tbody>
              {requests.map((req) => (
                <tr key={req.id} className="border-b border-border/50 hover:bg-bg-hover transition-colors">
                  <td className="py-2.5 px-2 text-text-secondary text-xs whitespace-nowrap">
                    {new Date(req.timestamp).toLocaleString('en-US', {
                      month: 'short',
                      day: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </td>
                  <td className="py-2.5 px-2 text-text-primary font-medium">{req.model}</td>
                  <td className="py-2.5 px-2 text-text-secondary">{req.provider}</td>
                  <td className="py-2.5 px-2 text-text-secondary">{req.team}</td>
                  <td className="py-2.5 px-2 text-right text-text-secondary tabular-nums">
                    {(req.input_tokens + req.output_tokens).toLocaleString()}
                  </td>
                  <td className="py-2.5 px-2 text-right text-accent-blue font-medium tabular-nums">
                    ${req.cost_usd.toFixed(4)}
                  </td>
                  <td className="py-2.5 px-2 text-right text-text-secondary tabular-nums">
                    {req.latency_ms.toFixed(0)} ms
                  </td>
                  <td className="py-2.5 px-2 text-center">{statusBadge(req.status)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
