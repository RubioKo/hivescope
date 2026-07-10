import { useEffect, useRef } from "react";
import ForceGraph2D from "react-force-graph-2d";
import { useGraphStore } from "../store/graphStore";
import type { GraphNode } from "../types";

const NODE_COLORS: Record<string, string> = {
  agent: "#58A6FF",
  tool: "#3ECF8E",
  user: "#D29922",
  gateway: "#F85149",
};

const STATUS_COLORS: Record<string, string> = {
  running: "#58A6FF",
  success: "#3ECF8E",
  error: "#F85149",
  idle: "#8B949E",
};

function getNodeColor(node: GraphNode): string {
  if (node.status === "running") return STATUS_COLORS.running;
  if (node.status === "error") return STATUS_COLORS.error;
  if (node.status === "success") return STATUS_COLORS.success;
  return NODE_COLORS[node.type] || STATUS_COLORS.idle;
}

export function GraphCanvas() {
  const fgRef = useRef<any>(null);
  const nodes = useGraphStore((s) => s.nodes);
  const links = useGraphStore((s) => s.links);
  const selectNode = useGraphStore((s) => s.selectNode);

  const graphData = {
    nodes: Array.from(nodes.values()),
    links: links,
  };

  useEffect(() => {
    if (fgRef.current) {
      fgRef.current.d3Force("charge")?.strength(-300);
      fgRef.current.d3Force("link")?.distance(150);
    }
  }, []);

  return (
    <div className="w-full h-full">
      <ForceGraph2D
        ref={fgRef}
        graphData={graphData}
        nodeLabel="label"
        nodeColor={(node: any) => getNodeColor(node as GraphNode)}
        nodeCanvasObject={(node: any, ctx: CanvasRenderingContext2D, globalScale: number) => {
          const label = node.label as string;
          const size = node.type === "agent" ? 8 : 6;
          const color = getNodeColor(node as GraphNode);

          ctx.beginPath();
          ctx.arc(node.x!, node.y!, size, 0, 2 * Math.PI);
          ctx.fillStyle = color;
          ctx.fill();

          if (node.status === "running") {
            ctx.beginPath();
            ctx.arc(node.x!, node.y!, size + 3, 0, 2 * Math.PI);
            ctx.strokeStyle = color;
            ctx.globalAlpha = 0.3;
            ctx.stroke();
            ctx.globalAlpha = 1;
          }

          if (globalScale > 0.8) {
            ctx.font = `${Math.max(10, 12 / globalScale)}px monospace`;
            ctx.fillStyle = "#E6EDF3";
            ctx.textAlign = "center";
            ctx.fillText(label, node.x!, node.y! + size + 12 / globalScale);
          }
        }}
        linkColor={() => "#30363D"}
        linkWidth={1}
        linkDirectionalParticles={2}
        linkDirectionalParticleWidth={2}
        linkDirectionalParticleColor={() => "#58A6FF"}
        onNodeClick={(node: any) => selectNode(node.id as string)}
        backgroundColor="#0D1117"
        width={window.innerWidth}
        height={window.innerHeight}
      />
    </div>
  );
}
