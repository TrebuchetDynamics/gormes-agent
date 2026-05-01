import { UnavailablePanel } from './UnavailablePanel';

export default function ChatPage() {
  return (
    <UnavailablePanel title="Chat" endpoint="/v1/chat/completions" />
  );
}
