import React from 'react';
import DashboardPage from './pages/DashboardPage';
import ChatPage from './pages/ChatPage';
import ConfigPage from './pages/ConfigPage';
import EnvPage from './pages/EnvPage';
import SessionsPage from './pages/SessionsPage';
import LogsPage from './pages/LogsPage';
import CronPage from './pages/CronPage';
import SkillsPage from './pages/SkillsPage';
import DocsPage from './pages/DocsPage';
import AnalyticsPage from './pages/AnalyticsPage';

export type DashboardRoute = {
  id: string;
  path: string;
  label: string;
  endpoint: string;
  Component: React.ComponentType;
};

export const dashboardRoutes: DashboardRoute[] = [
  { id: 'dashboard', path: '/', label: 'Dashboard', endpoint: '/api/status', Component: DashboardPage },
  { id: 'chat', path: '/chat', label: 'Chat', endpoint: '/v1/chat/completions', Component: ChatPage },
  { id: 'config', path: '/config', label: 'Config', endpoint: '/api/model/options', Component: ConfigPage },
  { id: 'env', path: '/env', label: 'Keys', endpoint: '/api/providers/oauth', Component: EnvPage },
  { id: 'sessions', path: '/sessions', label: 'Sessions', endpoint: '/api/sessions', Component: SessionsPage },
  { id: 'logs', path: '/logs', label: 'Logs', endpoint: '/api/status', Component: LogsPage },
  { id: 'cron', path: '/cron', label: 'Cron', endpoint: '/v1/admin/cron/jobs', Component: CronPage },
  { id: 'skills', path: '/skills', label: 'Skills', endpoint: '/api/dashboard/plugins', Component: SkillsPage },
  { id: 'docs', path: '/docs', label: 'Docs', endpoint: '/api/status', Component: DocsPage },
  { id: 'analytics', path: '/analytics', label: 'Analytics', endpoint: '/api/status', Component: AnalyticsPage },
];

function currentPath(): string {
  const path = window.location.pathname.replace(/\/$/, '') || '/';
  return dashboardRoutes.some((route) => route.path === path) ? path : '/sessions';
}

export default function App() {
  const path = currentPath();
  const active = dashboardRoutes.find((route) => route.path === path) ?? dashboardRoutes[0];
  const ActivePage = active.Component;

  return (
    <main className="dashboard-shell" data-active-route={active.id}>
      <nav aria-label="Dashboard routes">
        {dashboardRoutes.map((route) => (
          <a key={route.id} href={route.path} aria-current={route.id === active.id ? 'page' : undefined}>
            {route.label}
          </a>
        ))}
      </nav>
      <section className="dashboard-page">
        <ActivePage />
      </section>
    </main>
  );
}
