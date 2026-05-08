import { useState, useEffect } from 'react';
import { UnavailablePanel } from './UnavailablePanel';

type KanbanLane = {
  status: string;
  count: number;
  label: string;
};

type KanbanTask = {
  id: string;
  title: string;
  body?: string;
  status: string;
  priority?: number;
  assignee?: string;
  created_at: string;
};

type KanbanResponse = {
  lanes: KanbanLane[];
  dispatcher: { available: boolean; reason?: string };
  total_tasks: number;
};

type StatusFilter = '' | 'triage' | 'todo' | 'ready' | 'running' | 'blocked' | 'done' | 'archived';

const STATUS_COLORS: Record<string, string> = {
  triage: '#f7c873',
  todo: '#9ad',
  ready: '#b7f7d1',
  running: '#7eb8da',
  blocked: '#e8836d',
  done: '#7d9f7a',
  archived: '#666',
};

function statusBadge(status: string) {
  return (
    <span className="kanban-status-badge" style={{ background: STATUS_COLORS[status] ?? '#444', color: '#050505', padding: '0.1em 0.5em', borderRadius: '0.3em', fontSize: '0.82em', fontWeight: 600 }}>
      {status}
    </span>
  );
}

export default function DashboardPage() {
  const [kanban, setKanban] = useState<KanbanResponse | null>(null);
  const [tasks, setTasks] = useState<KanbanTask[]>([]);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/status')
      .then((res) => res.json())
      .then((data) => {
        const panel = data?.panels?.kanban;
        if (!panel || panel.state !== 'enabled') {
          setLoading(false);
          return;
        }
        return fetch('/api/kanban').then((res) => res.json());
      })
      .then((kanbanData) => {
        if (kanbanData) setKanban(kanbanData);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!kanban) return;
    const params = new URLSearchParams();
    if (statusFilter) params.set('status', statusFilter);
    fetch(`/api/kanban/tasks?${params.toString()}`)
      .then((res) => {
        if (!res.ok) throw new Error('failed to load tasks');
        return res.json();
      })
      .then((data) => setTasks(data.tasks ?? []))
      .catch((err) => setError(err.message));
  }, [statusFilter, kanban]);

  if (loading) {
    return (
      <section className="dashboard-page">
        <p className="eyebrow">Dashboard</p>
        <p>Loading kanban board...</p>
      </section>
    );
  }

  if (!kanban) {
    return (
      <UnavailablePanel title="Dashboard" endpoint="/api/kanban">
        <p style={{ marginTop: '1rem' }}>The kanban dashboard panel is not configured. Ensure the kanban store is wired when starting the gateway.</p>
      </UnavailablePanel>
    );
  }

  return (
    <section className="dashboard-page">
      <p className="eyebrow">Dashboard</p>
      <h1>Kanban Board</h1>

      <div className="kanban-lanes" style={{ display: 'flex', flexWrap: 'wrap', gap: '1rem', marginBottom: '2rem' }}>
        {kanban.lanes.map((lane) => (
          <button
            key={lane.status}
            onClick={() => setStatusFilter(statusFilter === lane.status ? '' : (lane.status as StatusFilter))}
            className="kanban-lane-card"
            style={{
              flex: '1 0 8rem',
              border: `1px solid ${STATUS_COLORS[lane.status] ?? '#444'}`,
              borderRadius: '0.75rem',
              padding: '1rem',
              background: '#111',
              cursor: 'pointer',
              textAlign: 'center',
              opacity: statusFilter && statusFilter !== lane.status ? 0.4 : 1,
            }}
          >
            <div style={{ fontSize: '2rem', fontWeight: 700, color: STATUS_COLORS[lane.status] ?? '#fff' }}>{lane.count}</div>
            <div style={{ color: '#aaa', marginTop: '0.25rem' }}>{lane.label}</div>
          </button>
        ))}
      </div>

      {error && <p style={{ color: '#e8836d' }}>Error: {error}</p>}

      <div className="kanban-task-list" style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
        {tasks.length === 0 && <p style={{ color: '#888' }}>No tasks match the current filter.</p>}
        {tasks.map((task) => (
          <div
            key={task.id}
            className="kanban-task-row"
            style={{
              border: '1px solid #333',
              borderRadius: '0.5rem',
              padding: '0.75rem 1rem',
              background: '#0a0a0a',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: '1rem',
            }}
          >
            <div style={{ flex: 1 }}>
              <div style={{ fontWeight: 600 }}>{task.title}</div>
              <div style={{ fontSize: '0.82em', color: '#888', marginTop: '0.25rem' }}>
                <code>{task.id}</code>
                {task.assignee && <span> · {task.assignee}</span>}
                {task.priority !== undefined && task.priority > 0 && <span> · P{task.priority}</span>}
              </div>
            </div>
            {statusBadge(task.status)}
          </div>
        ))}
      </div>
    </section>
  );
}
