import { UnavailablePanel } from './UnavailablePanel';

export default function EnvPage() {
  return (
    <UnavailablePanel title="Keys" endpoint="/api/providers/oauth" />
  );
}
