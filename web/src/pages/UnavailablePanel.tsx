type UnavailablePanelProps = {
  title: string;
  endpoint: string;
  children?: React.ReactNode;
};

export function UnavailablePanel({ title, endpoint, children }: UnavailablePanelProps) {
  return (
    <article className="unavailable-panel" data-endpoint={endpoint}>
      <p className="eyebrow">Gormes dashboard scaffold</p>
      <h1>{title}</h1>
      <p>
        This route is present for Hermes dashboard parity. The detailed page behavior will land in a later
        source-backed slice after the native API endpoint is bound.
      </p>
      <p>
        Required endpoint: <code>{endpoint}</code>
      </p>
      {children}
    </article>
  );
}
