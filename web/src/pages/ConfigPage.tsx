import { UnavailablePanel } from './UnavailablePanel';

export default function ConfigPage() {
  return (
    <UnavailablePanel title="Config" endpoint="/api/model/options" />
  );
}
