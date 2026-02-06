import SummaryCards from '../components/SummaryCards';
import CostChart from '../components/CostChart';
import RequestsTable from '../components/RequestsTable';
import InsightsPanel from '../components/InsightsPanel';

export default function DashboardPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-text-primary mb-1">Dashboard Overview</h1>
        <p className="text-text-muted text-sm">Monitor your LLM gateway costs, usage, and performance.</p>
      </div>

      <SummaryCards />
      <CostChart />

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <InsightsPanel />
        <div className="xl:col-span-1">
          <RequestsTable />
        </div>
      </div>
    </div>
  );
}
