"use client";

import { useRouter } from "next/navigation";
import * as THREE from "three";
import { useEffect, useMemo, useRef, useState, type CSSProperties } from "react";

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
  const stageRef = useRef<HTMLDivElement>(null);
  const [webglFallback, setWebglFallback] = useState(false);

  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) return;

    let frameId = 0;
    let renderer: THREE.WebGLRenderer | null = null;
    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(42, 1, 0.1, 100);
    const group = new THREE.Group();
    const core = new THREE.Mesh(
      new THREE.IcosahedronGeometry(0.58, 1),
      new THREE.MeshBasicMaterial({ color: 0x67f4d2, wireframe: true, transparent: true, opacity: 0.68 }),
    );

    const webglAvailable = typeof window !== "undefined" && ("WebGLRenderingContext" in window || "WebGL2RenderingContext" in window);
    if (!webglAvailable) {
      setWebglFallback(true);
      core.geometry.dispose();
      (core.material as THREE.Material).dispose();
      return;
    }

    try {
      renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true, powerPreference: "low-power" });
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      renderer.domElement.className = "network-canvas";
      renderer.domElement.setAttribute("aria-hidden", "true");
      stage.prepend(renderer.domElement);
      camera.position.z = 5.4;
      group.add(core);

      const pointById = new Map(nodes.map((node, index) => [node.id, new THREE.Vector3((node.x - 50) / 32, (50 - node.y) / 30, node.z / 38)]));
      const lineMaterial = new THREE.LineBasicMaterial({ color: 0x67f4d2, transparent: true, opacity: 0.32 });
      for (const edge of edges) {
        const start = pointById.get(edge.from);
        const end = pointById.get(edge.to);
        if (!start || !end) continue;
        group.add(new THREE.Line(new THREE.BufferGeometry().setFromPoints([start, end]), lineMaterial));
      }
      nodes.forEach((node) => {
        const material = new THREE.MeshBasicMaterial({ color: node.kind === "risk" ? 0xff966b : node.kind === "trace" ? 0x789bff : 0x67f4d2 });
        const mesh = new THREE.Mesh(new THREE.SphereGeometry(node.kind === "agent" ? 0.14 : 0.09, 12, 8), material);
        mesh.position.copy(pointById.get(node.id) ?? new THREE.Vector3());
        group.add(mesh);
      });
      scene.add(group);

      const resize = () => {
        if (!renderer) return;
        const width = stage.clientWidth || 640;
        const height = stage.clientHeight || 420;
        renderer.setSize(width, height, false);
        camera.aspect = width / height;
        camera.updateProjectionMatrix();
      };
      resize();
      const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
      const renderFrame = () => {
        if (!renderer) return;
        if (!reducedMotion) {
          group.rotation.y += 0.0018;
          core.rotation.z += 0.002;
        }
        renderer.render(scene, camera);
        if (!reducedMotion) frameId = window.requestAnimationFrame(renderFrame);
      };
      window.addEventListener("resize", resize);
      renderFrame();
      return () => {
        window.removeEventListener("resize", resize);
        window.cancelAnimationFrame(frameId);
        group.traverse((object) => {
          if (!(object instanceof THREE.Mesh) && !(object instanceof THREE.Line)) return;
          object.geometry.dispose();
          if (Array.isArray(object.material)) object.material.forEach((material) => material.dispose());
          else object.material.dispose();
        });
        renderer?.dispose();
        renderer?.forceContextLoss();
        renderer?.domElement.remove();
      };
    } catch {
      setWebglFallback(true);
      core.geometry.dispose();
      (core.material as THREE.Material).dispose();
      return () => { window.cancelAnimationFrame(frameId); };
    }
  }, []);

  return (
    <section className="network-scene" aria-labelledby="network-scene-title">
      <div className="network-scene-header">
        <div>
          <span className="eyebrow">Live topology</span>
          <h2 id="network-scene-title">Agent network</h2>
        </div>
        <span className="network-live"><i aria-hidden="true" /> Event flow active</span>
      </div>
      <div className={`network-stage${webglFallback ? " network-stage--fallback" : ""}`} ref={stageRef} role="img" aria-label="Agent network visualization">
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
