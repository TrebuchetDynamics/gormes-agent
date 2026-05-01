import { UnavailablePanel } from './UnavailablePanel';

export default function DashboardPage() {
  return (
    <UnavailablePanel title="Dashboard" endpoint="/api/status" />
  );
}
