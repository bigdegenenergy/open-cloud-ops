import { useState, useEffect, useCallback } from "react";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  ArcElement,
  Tooltip,
  Legend,
} from "chart.js";
import type { TooltipItem } from "chart.js";
import { Bar, Doughnut } from "react-chartjs-2";
import { getCostSummary } from "../api";
import type { CostSummaryItem } from "../api";

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  ArcElement,
  Tooltip,
  Legend,
);

const PALETTE = [
  "#3b82f6",
  "#10b981",
  "#f59e0b",
  "#ef4444",
  "#8b5cf6",
  "#06b6d4",
  "#ec4899",
  "#f97316",
  "#14b8a6",
  "#a855f7",
];

type Dimension = "model" | "provider" | "team";

export default function CostChart() {
  const [dimension, setDimension] = useState<Dimension>("model");
  const [data, setData] = useState<CostSummaryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback((dim: Dimension) => {
    setLoading(true);
    setError(null);
    const now = new Date();
    const from = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
    getCostSummary(dim, from.toISOString(), now.toISOString())
      .then(setData)
      .catch((err: unknown) => {
        const message =
          err instanceof Error ? err.message : "Failed to load cost data";
        setError(message);
        setData([]);
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchData(dimension);
  }, [dimension, fetchData]);

  const labels = data.map((d) => d.dimension_name);
  const costs = data.map((d) => d.total_cost_usd);
  const counts = data.map((d) => d.total_requests);
  const colors = data.map((_, i) => PALETTE[i % PALETTE.length]);

  const barData = {
    labels,
    datasets: [
      {
        label: "Cost (USD)",
        data: costs,
        backgroundColor: colors,
        borderRadius: 6,
        borderSkipped: false as const,
      },
    ],
  };

  const barOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: "#1e2235",
        titleColor: "#e2e8f0",
        bodyColor: "#94a3b8",
        borderColor: "#2d3250",
        borderWidth: 1,
        callbacks: {
          label: (ctx: TooltipItem<"bar">) =>
            ` $${(ctx.parsed.y ?? 0).toFixed(4)}`,
        },
      },
    },
    scales: {
      x: {
        ticks: { color: "#64748b", font: { size: 11 } },
        grid: { display: false },
      },
      y: {
        ticks: {
          color: "#64748b",
          font: { size: 11 },
          callback: (val: string | number) => `$${val}`,
        },
        grid: { color: "#2d325033" },
      },
    },
  };

  const doughnutData = {
    labels,
    datasets: [
      {
        data: counts,
        backgroundColor: colors,
        borderColor: "#1e2235",
        borderWidth: 2,
      },
    ],
  };

  const doughnutOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: "right" as const,
        labels: { color: "#94a3b8", padding: 12, font: { size: 11 } },
      },
      tooltip: {
        backgroundColor: "#1e2235",
        titleColor: "#e2e8f0",
        bodyColor: "#94a3b8",
        borderColor: "#2d3250",
        borderWidth: 1,
      },
    },
  };

  const tabs: { key: Dimension; label: string }[] = [
    { key: "model", label: "By Model" },
    { key: "provider", label: "By Provider" },
    { key: "team", label: "By Team" },
  ];

  return (
    <div className="bg-bg-card border border-border rounded-xl p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-semibold text-text-primary">
          Cost Breakdown
        </h2>
        <div className="flex gap-1 bg-bg-secondary rounded-lg p-1">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setDimension(tab.key)}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors cursor-pointer ${
                dimension === tab.key
                  ? "bg-accent-blue text-white"
                  : "text-text-secondary hover:text-text-primary"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-text-muted text-sm">Loading cost data...</div>
        </div>
      ) : error ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-accent-red text-sm">{error}</div>
        </div>
      ) : data.length === 0 ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-text-muted text-sm">
            No cost data available for this period.
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="h-64">
            <Bar data={barData} options={barOptions} />
          </div>
          <div className="h-64">
            <Doughnut data={doughnutData} options={doughnutOptions} />
          </div>
        </div>
      )}
    </div>
  );
}
