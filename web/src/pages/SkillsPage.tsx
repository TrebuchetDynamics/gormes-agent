import { UnavailablePanel } from './UnavailablePanel';

export default function SkillsPage() {
  return (
    <UnavailablePanel title="Skills" endpoint="/api/dashboard/plugins" />
  );
}
