import { useState, useEffect } from 'react';
import { getHealth } from '../api';
import type { HealthStatus } from '../api';

export type Page = 'dashboard' | 'costs' | 'requests' | 'budgets' | 'insights';

interface NavItem {
  key: Page;
  label: string;
  icon: string;
}

const NAV_ITEMS: NavItem[] = [
  { key: 'dashboard', label: 'Overview', icon: '[=]' },
  { key: 'costs', label: 'Cost Analysis', icon: '[$]' },
  { key: 'requests', label: 'Requests', icon: '[>]' },
  { key: 'budgets', label: 'Budgets', icon: '[B]' },
  { key: 'insights', label: 'Insights', icon: '[!]' },
];

interface SidebarProps {
  activePage: Page;
  onNavigate: (page: Page) => void;
}

export default function Sidebar({ activePage, onNavigate }: SidebarProps) {
  const [health, setHealth] = useState<HealthStatus | null>(null);

  useEffect(() => {
    getHealth()
      .then(setHealth)
      .catch(() => setHealth(null));
  }, []);

  return (
    <aside className="w-60 bg-bg-secondary border-r border-border flex flex-col h-screen fixed left-0 top-0">
      {/* Logo / Brand */}
      <div className="p-5 border-b border-border">
        <h1 className="text-lg font-bold text-text-primary tracking-tight">Cerebra</h1>
        <p className="text-xs text-text-muted mt-0.5">Open Cloud Ops Dashboard</p>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-4 px-3 space-y-1">
        {NAV_ITEMS.map((item) => (
          <button
            key={item.key}
            onClick={() => onNavigate(item.key)}
            className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors cursor-pointer ${
              activePage === item.key
                ? 'bg-accent-blue/10 text-accent-blue'
                : 'text-text-secondary hover:text-text-primary hover:bg-bg-hover'
            }`}
          >
            <span className="text-xs font-mono w-6 text-center">{item.icon}</span>
            {item.label}
          </button>
        ))}
      </nav>

      {/* Health Status Footer */}
      <div className="p-4 border-t border-border">
        <div className="flex items-center gap-2">
          <div
            className={`w-2 h-2 rounded-full ${
              health?.status === 'ok' || health?.status === 'healthy'
                ? 'bg-accent-green'
                : 'bg-accent-red'
            }`}
          />
          <span className="text-xs text-text-muted">
            {health
              ? `Cerebra ${health.version || 'v?'}`
              : 'Disconnected'}
          </span>
        </div>
        {health && health.uptime_seconds > 0 && (
          <div className="text-xs text-text-muted mt-1 ml-4">
            Uptime: {Math.floor(health.uptime_seconds / 3600)}h {Math.floor((health.uptime_seconds % 3600) / 60)}m
          </div>
        )}
      </div>
    </aside>
  );
}
