import { UnavailablePanel } from './UnavailablePanel';

export default function AnalyticsPage() {
  return (
    <UnavailablePanel title="Analytics" endpoint="/api/status" />
  );
}
