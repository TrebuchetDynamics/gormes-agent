import { useEffect, useState } from 'react';
import { Clock, Pause, Play, RefreshCw, Zap } from 'lucide-react';

type CronJobSchedule = string | { display?: unknown; expr?: unknown } | null;

type CronJob = {
  id?: string;
  name?: unknown;
  prompt?: unknown;
  script?: unknown;
  schedule?: CronJobSchedule;
  schedule_display?: unknown;
  state?: unknown;
  enabled?: boolean | null;
  paused?: boolean | null;
  created_at?: number | null;
  last_run_unix?: number | null;
  last_status?: unknown;
  next_run_unix?: number | null;
  target?: unknown;
  provider?: unknown;
  model?: unknown;
  skills?: unknown;
  has_script?: boolean | null;
};

type CronJobsResponse = {
  jobs?: CronJob[];
};

type CronAction = 'pause' | 'resume' | 'trigger';

const cronActionEndpoints: Record<CronAction, (id: string) => string> = {
  pause: (id) => `/v1/admin/cron/jobs/${encodeURIComponent(id)}/pause`,
  resume: (id) => `/v1/admin/cron/jobs/${encodeURIComponent(id)}/resume`,
  trigger: (id) => `/v1/admin/cron/jobs/${encodeURIComponent(id)}/trigger`,
};

function asText(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function truncateText(value: string, maxLength: number): string {
  return value.length > maxLength ? `${value.slice(0, maxLength)}...` : value;
}

function getJobPrompt(job: CronJob): string {
  return asText(job.prompt);
}

function getJobName(job: CronJob): string {
  return asText(job.name).trim();
}

function getJobTitle(job: CronJob): string {
  const name = getJobName(job);
  if (name) return name;

  const prompt = getJobPrompt(job).trim();
  if (prompt) return truncateText(prompt, 60);

  const script = asText(job.script).trim();
  if (script) return truncateText(script, 60);

  return asText(job.id).trim() || 'Cron job';
}

function getJobScheduleDisplay(job: CronJob): string {
  const schedule = job.schedule && typeof job.schedule === 'object' ? job.schedule : null;
  return (
    asText(job.schedule_display).trim() ||
    asText(schedule?.display).trim() ||
    asText(schedule?.expr).trim() ||
    asText(job.schedule).trim() ||
    '?'
  );
}

function getJobState(job: CronJob): string {
  return asText(job.state).trim() || (job.paused === true ? 'paused' : job.enabled === false ? 'disabled' : 'scheduled');
}

function jobID(job: CronJob): string {
  return asText(job.id).trim();
}

function formatUnix(value?: number | null): string {
  if (!value || value <= 0) return '—';
  return new Date(value * 1000).toLocaleString();
}

function targetLabel(job: CronJob): string {
  return asText(job.target).trim() || asText(job.provider).trim() || 'local';
}

function skillLabel(job: CronJob): string {
  if (!Array.isArray(job.skills)) return '';
  return job.skills.filter((value): value is string => typeof value === 'string' && value.trim() !== '').join(', ');
}

export default function CronPage() {
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const [actioning, setActioning] = useState<string | null>(null);

  useEffect(() => {
    let ignore = false;

    async function loadJobs() {
      setLoading(true);
      setError(null);
      try {
        const response = await fetch('/v1/admin/cron/jobs');
        if (!response.ok) throw new Error(`cron jobs request failed (${response.status})`);
        const body = (await response.json()) as CronJobsResponse;
        if (!ignore) setJobs(Array.isArray(body.jobs) ? body.jobs : []);
      } catch (err) {
        if (!ignore) {
          setJobs([]);
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (!ignore) setLoading(false);
      }
    }

    loadJobs();
    return () => {
      ignore = true;
    };
  }, [reloadToken]);

  async function runJobAction(job: CronJob, action: CronAction) {
    const id = jobID(job);
    if (!id) return;

    const actionKey = `${id}:${action}`;
    setActioning(actionKey);
    setError(null);
    try {
      const response = await fetch(cronActionEndpoints[action](id), { method: 'POST' });
      if (!response.ok) throw new Error(`${action} failed (${response.status})`);
      setReloadToken((value) => value + 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setActioning(null);
    }
  }

  if (loading) {
    return (
      <article className="cron-page">
        <p className="eyebrow">Cron</p>
        <div className="cron-state-panel">
          <RefreshCw size={18} aria-hidden="true" />
          <span>Loading cron jobs...</span>
        </div>
      </article>
    );
  }

  return (
    <article className="cron-page">
      <div className="cron-header">
        <div>
          <p className="eyebrow">Cron</p>
          <h1>Scheduled Jobs</h1>
        </div>
        <button className="cron-icon-button" type="button" onClick={() => setReloadToken((value) => value + 1)}>
          <RefreshCw size={16} aria-hidden="true" />
          <span>Retry</span>
        </button>
      </div>

      {error && (
        <div className="cron-state-panel cron-state-panel-error" role="status">
          <span>Failed to load cron jobs: {error}</span>
        </div>
      )}

      {jobs.length === 0 ? (
        <div className="cron-state-panel">
          <Clock size={18} aria-hidden="true" />
          <span>No cron jobs scheduled.</span>
        </div>
      ) : (
        <div className="cron-job-list">
          {jobs.map((job, index) => {
            const id = jobID(job);
            const state = getJobState(job);
            const pauseAction: CronAction = getJobState(job) === 'paused' ? 'resume' : 'pause';
            const actionKey = `${id}:${pauseAction}`;
            const triggerKey = `${id}:trigger`;
            const skills = skillLabel(job);

            return (
              <section className="cron-job-row" key={id || `cron-job-${index}`}>
                <div className="cron-job-main">
                  <div className="cron-job-title-line">
                    <h2>{getJobTitle(job)}</h2>
                    <span className={`cron-status cron-status-${state}`}>{state}</span>
                  </div>
                  <div className="cron-job-meta">
                    <span>{getJobScheduleDisplay(job)}</span>
                    <span>{targetLabel(job)}</span>
                    {asText(job.model).trim() && <span>{asText(job.model).trim()}</span>}
                    {skills && <span>{skills}</span>}
                    {job.has_script === true && <span>script</span>}
                  </div>
                  <div className="cron-job-times">
                    <span>last {formatUnix(job.last_run_unix)}</span>
                    <span>next {formatUnix(job.next_run_unix)}</span>
                    {asText(job.last_status).trim() && <span>{asText(job.last_status).trim()}</span>}
                  </div>
                </div>
                <div className="cron-job-actions">
                  <button
                    className="cron-icon-button"
                    type="button"
                    disabled={!id || actioning === actionKey}
                    onClick={() => runJobAction(job, pauseAction)}
                  >
                    {pauseAction === 'resume' ? <Play size={15} aria-hidden="true" /> : <Pause size={15} aria-hidden="true" />}
                    <span>{pauseAction === 'resume' ? 'Resume' : 'Pause'}</span>
                  </button>
                  <button
                    className="cron-icon-button"
                    type="button"
                    disabled={!id || actioning === triggerKey}
                    onClick={() => runJobAction(job, 'trigger')}
                  >
                    <Zap size={15} aria-hidden="true" />
                    <span>Run</span>
                  </button>
                </div>
              </section>
            );
          })}
        </div>
      )}
    </article>
  );
}
