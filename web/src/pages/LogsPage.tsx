import { UnavailablePanel } from './UnavailablePanel';

export default function LogsPage() {
  return (
    <UnavailablePanel title="Logs" endpoint="/api/status" />
  );
}
