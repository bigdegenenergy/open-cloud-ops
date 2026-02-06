import { useState } from 'react';
import Sidebar from './components/Sidebar';
import type { Page } from './components/Sidebar';
import DashboardPage from './pages/DashboardPage';
import CostsPage from './pages/CostsPage';
import RequestsPage from './pages/RequestsPage';
import BudgetsPage from './pages/BudgetsPage';
import InsightsPage from './pages/InsightsPage';

function renderPage(page: Page) {
  switch (page) {
    case 'dashboard':
      return <DashboardPage />;
    case 'costs':
      return <CostsPage />;
    case 'requests':
      return <RequestsPage />;
    case 'budgets':
      return <BudgetsPage />;
    case 'insights':
      return <InsightsPage />;
  }
}

export default function App() {
  const [activePage, setActivePage] = useState<Page>('dashboard');

  return (
    <div className="flex min-h-screen bg-bg-primary">
      <Sidebar activePage={activePage} onNavigate={setActivePage} />
      <main className="flex-1 ml-60 p-8">
        {renderPage(activePage)}
      </main>
    </div>
  );
}
