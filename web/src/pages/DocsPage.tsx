import { UnavailablePanel } from './UnavailablePanel';

export default function DocsPage() {
  return (
    <UnavailablePanel title="Docs" endpoint="/api/status" />
  );
}
