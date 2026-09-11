"use client";

import * as THREE from "three";
import { useEffect, useMemo, useRef, type CSSProperties } from "react";
import type { SectorNode, SectorRoute } from "./control-tower-model";

type Props = {
  nodes: SectorNode[];
  routes: SectorRoute[];
  selectedId?: string;
  onSelect?: (id: string) => void;
};

const tones = { healthy: "#67f4d2", degraded: "#f6ba63", critical: "#ff846d" };

function gridPosition(nodes: SectorNode[], node: SectorNode): CSSProperties {
  const xs = nodes.map(({ x }) => x);
  const ys = nodes.map(({ y }) => y);
  const xRange = Math.max(...xs) - Math.min(...xs) || 1;
  const yRange = Math.max(...ys) - Math.min(...ys) || 1;
  const column = 2 + Math.round(((node.x - Math.min(...xs)) / xRange) * 9);
  const row = 2 + Math.round(((Math.max(...ys) - node.y) / yRange) * 5);
  return { gridColumn: column, gridRow: row, zIndex: Math.round((node.z + 2) * 10) };
}

export function SectorTopology({ nodes, routes, selectedId, onSelect }: Props) {
  const stageRef = useRef<HTMLDivElement>(null);
  const positions = useMemo(
    () => new Map(nodes.map((node) => [node.id, new THREE.Vector3(node.x, node.y, node.z)])),
    [nodes],
  );

  useEffect(() => {
    const stage = stageRef.current;
    if (!stage || typeof window === "undefined") return;

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(44, 1, 0.1, 100);
    let renderer: THREE.WebGLRenderer | undefined;
    let frame = 0;
    let observer: ResizeObserver | undefined;
    let resize: (() => void) | undefined;
    let usesWindowResize = false;
    const reducedMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;

    try {
      if (!("WebGLRenderingContext" in window || "WebGL2RenderingContext" in window)) return;
      renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true, powerPreference: "low-power" });
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      renderer.domElement.setAttribute("aria-hidden", "true");
      renderer.domElement.className = "sector-topology-canvas";
      stage.prepend(renderer.domElement);
      camera.position.set(0, 0, 8);

      scene.add(new THREE.GridHelper(8, 8, 0x284150, 0x1a2d38));
      const criticalLines: THREE.LineBasicMaterial[] = [];
      for (const route of routes) {
        const from = positions.get(route.fromId);
        const to = positions.get(route.toId);
        if (!from || !to) continue;
        const material = new THREE.LineBasicMaterial({ color: tones[route.tone], transparent: true, opacity: route.tone === "critical" ? 0.9 : 0.45 });
        scene.add(new THREE.Line(new THREE.BufferGeometry().setFromPoints([from, to]), material));
        if (route.tone === "critical") criticalLines.push(material);
      }
      for (const node of nodes) {
        const mesh = new THREE.Mesh(
          new THREE.SphereGeometry(node.tone === "critical" ? 0.16 : 0.12, 16, 12),
          new THREE.MeshBasicMaterial({ color: tones[node.tone] }),
        );
        mesh.position.copy(positions.get(node.id)!);
        scene.add(mesh);
      }

      resize = () => {
        const width = stage.clientWidth || 1;
        const height = stage.clientHeight || 1;
        renderer?.setSize(width, height, false);
        camera.aspect = width / height;
        camera.updateProjectionMatrix();
      };
      const render = (time = 0) => {
        if (!renderer) return;
        if (!reducedMotion) criticalLines.forEach((material) => { material.opacity = 0.55 + Math.sin(time / 240) * 0.35; });
        renderer.render(scene, camera);
        if (!reducedMotion && criticalLines.length) frame = window.requestAnimationFrame(render);
      };
      resize();
      if (typeof ResizeObserver !== "undefined") {
        observer = new ResizeObserver(resize);
        observer.observe(stage);
      } else {
        usesWindowResize = true;
        window.addEventListener("resize", resize);
      }
      render();
    } catch {
      renderer?.domElement.remove();
      renderer?.dispose();
    }

    return () => {
      observer?.disconnect();
      if (usesWindowResize && resize) window.removeEventListener("resize", resize);
      window.cancelAnimationFrame(frame);
      scene.traverse((object) => {
        if (!(object instanceof THREE.Mesh) && !(object instanceof THREE.Line) && !(object instanceof THREE.GridHelper)) return;
        object.geometry.dispose();
        const materials = Array.isArray(object.material) ? object.material : [object.material];
        materials.forEach((material) => material.dispose());
      });
      renderer?.dispose();
      renderer?.domElement.remove();
    };
  }, [nodes, positions, routes]);

  return (
    <div className="sector-topology">
      <div ref={stageRef} role="img" aria-label="Live Agent sector topology">
        <div aria-hidden="true" className="sector-topology-grid" />
      </div>
      <div className="sector-topology-nodes" style={{ display: "grid", gridTemplateColumns: "repeat(12, minmax(0, 1fr))", gridTemplateRows: "repeat(8, minmax(0, 1fr))" }}>
        {nodes.map((node) => (
          <button key={node.id} type="button" aria-label={`Inspect ${node.callsign} ${node.label}`} aria-pressed={selectedId === node.id} onClick={() => onSelect?.(node.id)} style={{ ...gridPosition(nodes, node), color: tones[node.tone] }}>
            <span>{node.callsign}</span><span>{node.label}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
