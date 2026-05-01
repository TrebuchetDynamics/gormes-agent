import { UnavailablePanel } from './UnavailablePanel';

export default function CronPage() {
  return (
    <UnavailablePanel title="Cron" endpoint="/v1/admin/cron/jobs" />
  );
}
