"use client";

import { useRouter } from "next/navigation";
import { useMemo, type CSSProperties } from "react";

type NetworkNode = {
  id: string;
  label: string;
  kind: "agent" | "trace" | "policy" | "risk" | "audit";
  detail: string;
  href: string;
  x: number;
  y: number;
  z: number;
};

const nodes: NetworkNode[] = [
  { id: "agent", label: "Agent Registry", kind: "agent", detail: "Identity and credential health", href: "/settings/agents", x: 50, y: 48, z: 32 },
  { id: "trace", label: "Trace Explorer", kind: "trace", detail: "Execution chains and cost", href: "/dashboard/traces", x: 22, y: 28, z: 12 },
  { id: "policy", label: "Policy Guardrails", kind: "policy", detail: "Versioned controls", href: "/settings/policies", x: 76, y: 26, z: 20 },
  { id: "risk", label: "Risk Review", kind: "risk", detail: "Prompt injection and safety signals", href: "/dashboard/risk", x: 77, y: 72, z: 8 },
  { id: "audit", label: "Audit Evidence", kind: "audit", detail: "Immutable activity records", href: "/settings/audit", x: 22, y: 73, z: 16 },
];

const edges = [
  { from: "agent", to: "trace", angle: -155, length: 31 },
  { from: "agent", to: "policy", angle: -24, length: 30 },
  { from: "agent", to: "risk", angle: 25, length: 31 },
  { from: "agent", to: "audit", angle: 155, length: 31 },
];

const edgeStyle = (angle: number, length: number): CSSProperties => ({ "--edge-angle": `${angle}deg`, "--edge-length": `${length}%` } as CSSProperties);

export function AgentNetworkScene() {
  const router = useRouter();
  const nodeMap = useMemo(() => new Map(nodes.map((node) => [node.id, node])), []);

  return (
    <section className="network-scene" aria-labelledby="network-scene-title">
      <div className="network-scene-header">
        <div>
          <span className="eyebrow">Live topology</span>
          <h2 id="network-scene-title">Agent network</h2>
        </div>
        <span className="network-live"><i aria-hidden="true" /> Event flow active</span>
      </div>
      <div className="network-stage" role="img" aria-label="Agent network visualization">
        <div className="network-halo" aria-hidden="true" />
        <div className="network-orbit network-orbit--one" aria-hidden="true" />
        <div className="network-orbit network-orbit--two" aria-hidden="true" />
        {edges.map((edge) => <span className="network-edge" key={`${edge.from}-${edge.to}`} style={edgeStyle(edge.angle, edge.length)} aria-hidden="true"><i /></span>)}
        <div className="network-core" aria-hidden="true"><span>OPS</span></div>
        {nodes.map((node) => {
          const style: CSSProperties = { left: `${node.x}%`, top: `${node.y}%`, zIndex: node.z, "--node-depth": `${node.z}px` } as CSSProperties;
          return (
            <button className={`network-node network-node--${node.kind}`} key={node.id} style={style} type="button" onClick={() => router.push(node.href)} aria-label={node.label}>
              <span className="network-node-point" aria-hidden="true" />
              <span className="network-node-copy"><strong>{node.label}</strong><small>{node.detail}</small></span>
            </button>
          );
        })}
      </div>
      <div className="network-legend" aria-label="Network node legend">
        {Array.from(nodeMap.values()).map((node) => <span key={node.id}><i className={`legend-dot legend-dot--${node.kind}`} aria-hidden="true" />{node.kind}</span>)}
      </div>
    </section>
  );
}
