import InsightsPanel from '../components/InsightsPanel';

export default function InsightsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-text-primary mb-1">Insights & Alerts</h1>
        <p className="text-text-muted text-sm">Automated recommendations and alerts for cost optimization.</p>
      </div>

      <InsightsPanel />
    </div>
  );
}
