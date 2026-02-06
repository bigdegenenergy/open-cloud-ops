import { useState, useEffect } from "react";
import { getInsights } from "../api";
import type { Insight } from "../api";

function severityStyles(severity: Insight["severity"]) {
  switch (severity) {
    case "critical":
      return {
        border: "border-accent-red/30",
        bg: "bg-accent-red/5",
        badge: "bg-accent-red/15 text-accent-red",
        icon: "!",
      };
    case "warning":
      return {
        border: "border-accent-amber/30",
        bg: "bg-accent-amber/5",
        badge: "bg-accent-amber/15 text-accent-amber",
        icon: "!",
      };
    default:
      return {
        border: "border-accent-blue/30",
        bg: "bg-accent-blue/5",
        badge: "bg-accent-blue/15 text-accent-blue",
        icon: "i",
      };
  }
}

export default function InsightsPanel() {
  const [insights, setInsights] = useState<Insight[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    getInsights()
      .then((data) => {
        if (!cancelled) setInsights(data);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        const message =
          err instanceof Error ? err.message : "Failed to load insights";
        setError(message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="bg-bg-card border border-border rounded-xl p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-semibold text-text-primary">
          Insights & Alerts
        </h2>
        <span className="text-text-muted text-xs">
          {insights.length} active {insights.length === 1 ? "alert" : "alerts"}
        </span>
      </div>

      {loading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-20 bg-bg-hover rounded-lg animate-pulse"
            />
          ))}
        </div>
      ) : error ? (
        <div className="text-center py-8 text-accent-red text-sm">{error}</div>
      ) : insights.length === 0 ? (
        <div className="text-center py-8">
          <div className="text-accent-green text-2xl mb-2">[ok]</div>
          <div className="text-text-muted text-sm">
            All clear. No alerts or recommendations at this time.
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          {insights.map((insight) => {
            const styles = severityStyles(insight.severity);
            return (
              <div
                key={insight.id}
                className={`p-4 rounded-lg border ${styles.border} ${styles.bg}`}
              >
                <div className="flex items-start gap-3">
                  <div
                    className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0 mt-0.5 ${styles.badge}`}
                  >
                    {styles.icon}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-text-primary font-medium text-sm">
                        {insight.title}
                      </span>
                      <span
                        className={`px-1.5 py-0.5 rounded text-xs font-medium ${styles.badge}`}
                      >
                        {insight.severity}
                      </span>
                      <span className="text-text-muted text-xs px-1.5 py-0.5 bg-bg-card rounded">
                        {insight.category}
                      </span>
                    </div>
                    <p className="text-text-secondary text-sm mb-2">
                      {insight.description}
                    </p>
                    {insight.recommendation && (
                      <div className="text-xs text-text-muted bg-bg-card rounded px-3 py-2">
                        Recommendation: {insight.recommendation}
                      </div>
                    )}
                    <div className="text-xs text-text-muted mt-2">
                      {new Date(insight.created_at).toLocaleString()}
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
