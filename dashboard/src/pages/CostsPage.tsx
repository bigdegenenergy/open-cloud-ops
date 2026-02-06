import CostChart from '../components/CostChart';
import SummaryCards from '../components/SummaryCards';

export default function CostsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-text-primary mb-1">Cost Analysis</h1>
        <p className="text-text-muted text-sm">Break down costs by model, provider, or team.</p>
      </div>

      <SummaryCards />
      <CostChart />
    </div>
  );
}
