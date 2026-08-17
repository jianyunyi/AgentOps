import type { Span } from "../../lib/api/types";

function SpanNode({ span, children }: { span: Span; children: Span[] }) {
  return (
    <li>
      <div className="span-node">
        <strong>{span.name}</strong>
        <span>{span.spanType}</span>
        <span>{span.durationMs} ms</span>
        <span>{span.status}</span>
      </div>
      {children.length > 0 && <SpanTree spans={children} parentId={span.spanId} />}
    </li>
  );
}

export function SpanTree({ spans, parentId = null }: { spans: Span[]; parentId?: string | null }) {
  const children = spans.filter((span) => span.parentSpanId === parentId);
  if (children.length === 0) return null;
  return (
    <ul className="span-tree">
      {children.map((span) => (
        <SpanNode key={span.spanId} span={span} children={spans.filter((candidate) => candidate.parentSpanId === span.spanId)} />
      ))}
    </ul>
  );
}
