import type { Span } from "../../lib/api/types";

export interface SpanTreeNode {
  span: Span;
  children: SpanTreeNode[];
  detachedRoot: boolean;
}

function compareSpans(left: Span, right: Span): number {
  return left.sequence - right.sequence
    || left.startedAt.localeCompare(right.startedAt)
    || left.spanId.localeCompare(right.spanId);
}

/** Build a deterministic, root-first tree without silently dropping malformed spans. */
export function buildSpanTree(spans: Span[]): SpanTreeNode[] {
  const ordered: Span[] = [];
  const spanById = new Map<string, Span>();

  for (const span of [...spans].sort(compareSpans)) {
    if (spanById.has(span.spanId)) continue;
    spanById.set(span.spanId, span);
    ordered.push(span);
  }

  const parentById = new Map<string, string>();
  const detached = new Set<string>();

  for (const span of ordered) {
    const parentId = span.parentSpanId?.trim();
    if (!parentId) continue;
    if (parentId === span.spanId || !spanById.has(parentId)) {
      detached.add(span.spanId);
      continue;
    }
    parentById.set(span.spanId, parentId);
  }

  // Break one edge in each malformed parent cycle, preferring to preserve earlier events.
  const inspected = new Set<string>();
  for (const span of ordered) {
    const path: string[] = [];
    const positionById = new Map<string, number>();
    let currentId: string | undefined = span.spanId;

    while (currentId && !inspected.has(currentId)) {
      const cycleStart = positionById.get(currentId);
      if (cycleStart !== undefined) {
        const cycleIds = path.slice(cycleStart);
        const breakId = cycleIds
          .map((id) => spanById.get(id)!)
          .sort(compareSpans)[0]?.spanId;
        if (breakId) {
          parentById.delete(breakId);
          detached.add(breakId);
        }
        break;
      }
      positionById.set(currentId, path.length);
      path.push(currentId);
      currentId = parentById.get(currentId);
    }
    for (const id of path) inspected.add(id);
  }

  const nodeById = new Map<string, SpanTreeNode>();
  for (const span of ordered) {
    nodeById.set(span.spanId, { span, children: [], detachedRoot: detached.has(span.spanId) });
  }

  const roots: SpanTreeNode[] = [];
  for (const span of ordered) {
    const node = nodeById.get(span.spanId)!;
    const parentId = parentById.get(span.spanId);
    const parent = parentId ? nodeById.get(parentId) : undefined;
    if (parent) parent.children.push(node);
    else roots.push(node);
  }

  return roots;
}
