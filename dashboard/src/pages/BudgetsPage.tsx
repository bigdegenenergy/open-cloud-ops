import BudgetPanel from '../components/BudgetPanel';

export default function BudgetsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-text-primary mb-1">Budget Management</h1>
        <p className="text-text-muted text-sm">Set and monitor spending limits for teams, models, and providers.</p>
      </div>

      <BudgetPanel />
    </div>
  );
}
