import { useState, useEffect } from "react";
import { getReport } from "../api";
import type { ReportSummary } from "../api";

function formatUsd(value: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat("en-US").format(value);
}

interface CardProps {
  title: string;
  value: string;
  subtitle: string;
  accentClass: string;
  icon: string;
}

function Card({ title, value, subtitle, accentClass, icon }: CardProps) {
  return (
    <div className="bg-bg-card border border-border rounded-xl p-6 hover:bg-bg-hover transition-colors">
      <div className="flex items-center justify-between mb-4">
        <span className="text-text-secondary text-sm font-medium">{title}</span>
        <span className={`text-2xl ${accentClass}`}>{icon}</span>
      </div>
      <div className={`text-2xl font-bold mb-1 ${accentClass}`}>{value}</div>
      <div className="text-text-muted text-xs">{subtitle}</div>
    </div>
  );
}

export default function SummaryCards() {
  const [report, setReport] = useState<ReportSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    getReport()
      .then((data) => {
        if (!cancelled) setReport(data);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        const message =
          err instanceof Error ? err.message : "Failed to load report";
        setError(message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="bg-bg-card border border-border rounded-xl p-6 animate-pulse"
          >
            <div className="h-4 bg-bg-hover rounded w-24 mb-4" />
            <div className="h-8 bg-bg-hover rounded w-32 mb-2" />
            <div className="h-3 bg-bg-hover rounded w-20" />
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
        <Card
          title="Total Cost"
          value="--"
          subtitle="No data available"
          accentClass="text-accent-blue"
          icon="$"
        />
        <Card
          title="Total Requests"
          value="--"
          subtitle="No data available"
          accentClass="text-accent-green"
          icon="#"
        />
        <Card
          title="Avg Latency"
          value="--"
          subtitle="No data available"
          accentClass="text-accent-amber"
          icon="~"
        />
        <Card
          title="Savings"
          value="--"
          subtitle="No data available"
          accentClass="text-accent-purple"
          icon="%"
        />
      </div>
    );
  }

  if (!report) return null;

  const r = report;
  const periodLabel =
    r.from && r.to
      ? `${r.from.slice(0, 10)} to ${r.to.slice(0, 10)}`
      : "Current period";

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
      <Card
        title="Total Cost"
        value={formatUsd(r.total_cost_usd)}
        subtitle={periodLabel}
        accentClass="text-accent-blue"
        icon="$"
      />
      <Card
        title="Total Requests"
        value={formatNumber(r.total_requests)}
        subtitle={periodLabel}
        accentClass="text-accent-green"
        icon="#"
      />
      <Card
        title="Avg Latency"
        value={`${r.avg_latency_ms.toFixed(0)} ms`}
        subtitle="Across all models"
        accentClass="text-accent-amber"
        icon="~"
      />
      <Card
        title="Savings"
        value={formatUsd(r.total_savings_usd)}
        subtitle="Via caching & routing"
        accentClass="text-accent-purple"
        icon="%"
      />
    </div>
  );
}
