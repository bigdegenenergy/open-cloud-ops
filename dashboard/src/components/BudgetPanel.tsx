import { useState, useEffect } from "react";
import { getBudgets, createBudget } from "../api";
import type { Budget } from "../api";

function utilizationColor(pct: number): string {
  if (pct >= 90) return "bg-accent-red";
  if (pct >= 70) return "bg-accent-amber";
  return "bg-accent-green";
}

function utilizationTextColor(pct: number): string {
  if (pct >= 90) return "text-accent-red";
  if (pct >= 70) return "text-accent-amber";
  return "text-accent-green";
}

export default function BudgetPanel() {
  const [budgets, setBudgets] = useState<Budget[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [scope, setScope] = useState("team");
  const [entityId, setEntityId] = useState("");
  const [limitUsd, setLimitUsd] = useState("");
  const [periodDays, setPeriodDays] = useState("30");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    getBudgets()
      .then((data) => {
        if (!cancelled) setBudgets(data);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        const message =
          err instanceof Error ? err.message : "Failed to load budgets";
        setError(message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  function loadBudgets() {
    setLoading(true);
    getBudgets()
      .then(setBudgets)
      .catch((err: unknown) => {
        const message =
          err instanceof Error ? err.message : "Failed to load budgets";
        setError(message);
      })
      .finally(() => setLoading(false));
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);

    const limit = parseFloat(limitUsd);
    const period = parseInt(periodDays, 10);

    if (!entityId.trim()) {
      setFormError("Entity ID is required");
      return;
    }
    if (isNaN(limit) || limit <= 0) {
      setFormError("Budget limit must be a positive number");
      return;
    }
    if (isNaN(period) || period <= 0) {
      setFormError("Period must be a positive number of days");
      return;
    }

    setSubmitting(true);
    createBudget({
      scope,
      entity_id: entityId.trim(),
      limit_usd: limit,
      period_days: period,
    })
      .then(() => {
        setShowForm(false);
        setEntityId("");
        setLimitUsd("");
        setPeriodDays("30");
        loadBudgets();
      })
      .catch((err: unknown) => {
        const message =
          err instanceof Error ? err.message : "Failed to create budget";
        setFormError(message);
      })
      .finally(() => setSubmitting(false));
  }

  return (
    <div className="bg-bg-card border border-border rounded-xl p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-semibold text-text-primary">
          Budget Management
        </h2>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-3 py-1.5 bg-accent-blue text-white text-xs font-medium rounded-lg hover:bg-accent-blue/80 transition-colors cursor-pointer"
        >
          {showForm ? "Cancel" : "+ New Budget"}
        </button>
      </div>

      {showForm && (
        <form
          onSubmit={handleSubmit}
          className="mb-6 p-4 bg-bg-secondary rounded-lg border border-border space-y-4"
        >
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-text-secondary text-xs font-medium mb-1">
                Scope
              </label>
              <select
                value={scope}
                onChange={(e) => setScope(e.target.value)}
                className="w-full bg-bg-card border border-border rounded-lg px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-accent-blue"
              >
                <option value="team">Team</option>
                <option value="model">Model</option>
                <option value="provider">Provider</option>
                <option value="project">Project</option>
              </select>
            </div>
            <div>
              <label className="block text-text-secondary text-xs font-medium mb-1">
                Entity ID
              </label>
              <input
                type="text"
                value={entityId}
                onChange={(e) => setEntityId(e.target.value)}
                placeholder="e.g. platform-team"
                className="w-full bg-bg-card border border-border rounded-lg px-3 py-2 text-sm text-text-primary placeholder-text-muted focus:outline-none focus:ring-1 focus:ring-accent-blue"
              />
            </div>
            <div>
              <label className="block text-text-secondary text-xs font-medium mb-1">
                Limit (USD)
              </label>
              <input
                type="number"
                step="0.01"
                min="0"
                value={limitUsd}
                onChange={(e) => setLimitUsd(e.target.value)}
                placeholder="1000.00"
                className="w-full bg-bg-card border border-border rounded-lg px-3 py-2 text-sm text-text-primary placeholder-text-muted focus:outline-none focus:ring-1 focus:ring-accent-blue"
              />
            </div>
            <div>
              <label className="block text-text-secondary text-xs font-medium mb-1">
                Period (days)
              </label>
              <input
                type="number"
                min="1"
                value={periodDays}
                onChange={(e) => setPeriodDays(e.target.value)}
                className="w-full bg-bg-card border border-border rounded-lg px-3 py-2 text-sm text-text-primary placeholder-text-muted focus:outline-none focus:ring-1 focus:ring-accent-blue"
              />
            </div>
          </div>
          {formError && (
            <div className="text-accent-red text-xs">{formError}</div>
          )}
          <button
            type="submit"
            disabled={submitting}
            className="px-4 py-2 bg-accent-green text-white text-sm font-medium rounded-lg hover:bg-accent-green/80 transition-colors disabled:opacity-50 cursor-pointer"
          >
            {submitting ? "Creating..." : "Create Budget"}
          </button>
        </form>
      )}

      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-16 bg-bg-hover rounded-lg animate-pulse"
            />
          ))}
        </div>
      ) : error ? (
        <div className="text-center py-8 text-accent-red text-sm">{error}</div>
      ) : budgets.length === 0 ? (
        <div className="text-center py-8 text-text-muted text-sm">
          No budgets configured. Create one to start tracking spend limits.
        </div>
      ) : (
        <div className="space-y-3">
          {budgets.map((b) => {
            const safePct = b.limit_usd > 0 ? b.utilization_pct : 0;
            const pct = Math.min(safePct, 100);
            return (
              <div
                key={`${b.scope}-${b.entity_id}`}
                className="p-4 bg-bg-secondary rounded-lg border border-border"
              >
                <div className="flex items-center justify-between mb-2">
                  <div>
                    <span className="text-text-primary font-medium text-sm">
                      {b.entity_id}
                    </span>
                    <span className="text-text-muted text-xs ml-2 px-1.5 py-0.5 bg-bg-card rounded">
                      {b.scope}
                    </span>
                  </div>
                  <span
                    className={`text-sm font-medium ${utilizationTextColor(safePct)}`}
                  >
                    {safePct.toFixed(1)}%
                  </span>
                </div>
                <div className="w-full bg-bg-card rounded-full h-2 mb-2">
                  <div
                    className={`h-2 rounded-full transition-all ${utilizationColor(safePct)}`}
                    style={{ width: `${pct}%` }}
                  />
                </div>
                <div className="flex justify-between text-xs text-text-muted">
                  <span>${b.spent_usd.toFixed(2)} spent</span>
                  <span>
                    ${b.limit_usd.toFixed(2)} limit / {b.period_days}d
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
