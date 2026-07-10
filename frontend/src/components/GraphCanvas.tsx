import { useEffect, useRef, useMemo } from "react";
import ForceGraph2D from "react-force-graph-2d";
import { useGraphStore } from "../store/graphStore";
import type { GraphNode, GraphLink } from "../types";

const NODE_COLORS: Record<string, string> = {
  agent: "#58A6FF",
  tool: "#3ECF8E",
  user: "#D29922",
  gateway: "#F85149",
};

const PHASE_COLORS: Record<string, string> = {
  planning: "#D29922",
  coding: "#58A6FF",
  reviewing: "#BC8CFF",
  testing: "#3ECF8E",
};

const STATUS_GLOW: Record<string, string> = {
  running: "#58A6FF",
  success: "#3ECF8E",
  error: "#F85149",
};

function drawShape(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  r: number,
  type: string,
) {
  ctx.beginPath();
  switch (type) {
    case "tool":
      ctx.roundRect(x - r, y - r * 0.8, r * 2, r * 1.6, 3);
      break;
    case "gateway":
      for (let i = 0; i < 6; i++) {
        const a = (Math.PI / 3) * i - Math.PI / 6;
        const px = x + r * Math.cos(a);
        const py = y + r * Math.sin(a);
        i === 0 ? ctx.moveTo(px, py) : ctx.lineTo(px, py);
      }
      ctx.closePath();
      break;
    case "user":
      ctx.moveTo(x, y - r);
      ctx.lineTo(x + r * 0.7, y);
      ctx.lineTo(x, y + r);
      ctx.lineTo(x - r * 0.7, y);
      ctx.closePath();
      break;
    default:
      ctx.arc(x, y, r, 0, 2 * Math.PI);
  }
}

export function GraphCanvas() {
  const fgRef = useRef<any>(null);
  const nodes = useGraphStore((s) => s.nodes);
  const links = useGraphStore((s) => s.links);
  const selectNode = useGraphStore((s) => s.selectNode);

  const graphData = useMemo(
    () => ({
      nodes: Array.from(nodes.values()),
      links: links,
    }),
    [nodes, links],
  );

  useEffect(() => {
    const fg = fgRef.current;
    if (!fg) return;
    fg.d3Force("charge")?.strength(-250);
    fg.d3Force("link")?.distance(120);
    fg.d3Force("center", null);
    fg.d3ReheatSimulation();
  }, []);

  useEffect(() => {
    if (graphData.nodes.length === 0 || !fgRef.current) return;
    fgRef.current.d3ReheatSimulation();
  }, [graphData.nodes.length]);

  return (
    <div className="w-full h-full">
      <ForceGraph2D
        ref={fgRef}
        graphData={graphData}
        nodeLabel=""
        nodeCanvasObject={(node: any, ctx: CanvasRenderingContext2D, globalScale: number) => {
          const n = node as GraphNode;
          const x = node.x!;
          const y = node.y!;
          const r = n.type === "agent" ? 7 : 6;
          const color = NODE_COLORS[n.type] || "#8B949E";
          const now = performance.now();

          ctx.save();

          if (n.status === "running") {
            ctx.beginPath();
            const pulseR = r + 3 + Math.sin(now * 0.005) * 3;
            ctx.arc(x, y, pulseR, 0, 2 * Math.PI);
            ctx.fillStyle = color;
            ctx.globalAlpha = 0.15 + Math.sin(now * 0.005) * 0.1;
            ctx.fill();
            ctx.globalAlpha = 1;
          }

          if (n.status === "error") {
            ctx.shadowColor = STATUS_GLOW.error;
            ctx.shadowBlur = 12;
          }

          drawShape(ctx, x, y, r, n.type);
          ctx.fillStyle = color;
          ctx.fill();

          if (n.status === "running") {
            drawShape(ctx, x, y, r, n.type);
            ctx.strokeStyle = color;
            ctx.lineWidth = 2;
            ctx.globalAlpha = 0.6 + Math.sin(now * 0.004) * 0.4;
            ctx.stroke();
            ctx.globalAlpha = 1;
          }

          ctx.shadowBlur = 0;

          if (globalScale > 0.6) {
            ctx.font = `${Math.max(10, 11 / globalScale)}px monospace`;
            ctx.fillStyle = "#E6EDF3";
            ctx.textAlign = "center";
            ctx.fillText(n.label, x, y + r + 14 / globalScale);
          }

          ctx.restore();
        }}
        linkColor={(link: any) => {
          const l = link as GraphLink;
          return PHASE_COLORS[l.phase] || "#30363D";
        }}
        linkWidth={(link: any) => {
          const l = link as GraphLink;
          const age = Date.now() - l.createdAt;
          return age < 5000 ? 2.5 : 1;
        }}
        linkDirectionalParticles={2}
        linkDirectionalParticleWidth={2.5}
        linkDirectionalParticleSpeed={(link: any) => {
          const l = link as GraphLink;
          const age = Date.now() - l.createdAt;
          return age < 5000 ? 0.02 : 0.005;
        }}
        linkDirectionalParticleColor={(link: any) => {
          const l = link as GraphLink;
          return PHASE_COLORS[l.phase] || "#58A6FF";
        }}
        onNodeClick={(node: any) => selectNode(node.id as string)}
        onBackgroundClick={() => selectNode(null)}
        backgroundColor="#0D1117"
        width={window.innerWidth}
        height={window.innerHeight}
      />
    </div>
  );
}
