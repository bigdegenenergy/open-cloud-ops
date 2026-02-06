import RequestsTable from '../components/RequestsTable';

export default function RequestsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-text-primary mb-1">Recent Requests</h1>
        <p className="text-text-muted text-sm">View the latest LLM API requests processed through Cerebra.</p>
      </div>

      <RequestsTable />
    </div>
  );
}
