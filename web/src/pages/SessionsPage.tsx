import { UnavailablePanel } from './UnavailablePanel';

export default function SessionsPage() {
  return (
    <UnavailablePanel title="Sessions" endpoint="/api/sessions" />
  );
}
