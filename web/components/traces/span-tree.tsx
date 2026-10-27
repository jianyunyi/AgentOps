import type { Span } from "../../lib/api/types";
import { buildSpanTree, type SpanTreeNode } from "./span-tree-model";

function SpanNode({ node, ancestors }: { node: SpanTreeNode; ancestors: string[] }) {
  const { span } = node;
  const path = [...ancestors, span.name];

  return (
    <li>
      <article className="span-node">
        <div className="span-node__heading">
          <strong>{span.name}</strong>
          <span className="span-node__type">{span.spanType}</span>
          <span>#{span.sequence}</span>
          <span>{span.durationMs} ms</span>
          <span>{span.status}</span>
          {node.detachedRoot && <span className="span-node__warning">Parent link detached</span>}
        </div>
        <div className="span-node__path">
          <span>Path from displayed root</span>
          <code>{path.join(" → ")}</code>
        </div>
        <code className="span-node__ids">
          Span {span.spanId}{span.parentSpanId ? ` · parent ${span.parentSpanId}` : " · root"}
        </code>
        <time dateTime={span.startedAt}>{new Date(span.startedAt).toLocaleString()}</time>
      </article>
      {node.children.length > 0 && (
        <ul className="span-tree__children">
          {node.children.map((child) => (
            <SpanNode key={child.span.spanId} node={child} ancestors={path} />
          ))}
        </ul>
      )}
    </li>
  );
}

export function SpanTree({ spans }: { spans: Span[] }) {
  const roots = buildSpanTree(spans);
  if (roots.length === 0) return <p className="empty-state">No spans recorded for this trace.</p>;

  return (
    <ul className="span-tree" aria-label="Execution spans from trace root">
      {roots.map((root) => <SpanNode key={root.span.spanId} node={root} ancestors={[]} />)}
    </ul>
  );
}
