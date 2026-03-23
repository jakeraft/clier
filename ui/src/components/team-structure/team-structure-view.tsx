import { Panel, ReactFlow, ReactFlowProvider } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { nodeTypes } from "@/components/team-structure/member-node";
import type { TeamStructureProperties } from "@/components/team-structure/team-structure-types";

export function TeamStructure({
  nodes,
  edges,
  edgeTypes: customEdgeTypes,
  onNodeClick,
  children,
  className,
}: Readonly<TeamStructureProperties>) {
  return (
    <div className={cn("rounded-base bg-muted/50 h-[40vh] border", className)}>
      <ReactFlowProvider>
        <TeamStructureInner nodes={nodes} edges={edges} edgeTypes={customEdgeTypes} onNodeClick={onNodeClick}>
          {children}
        </TeamStructureInner>
      </ReactFlowProvider>
    </div>
  );
}

function TeamStructureInner({
  nodes,
  edges,
  edgeTypes: customEdgeTypes,
  onNodeClick,
  children,
}: Readonly<Omit<TeamStructureProperties, "className">>) {
  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      edgeTypes={customEdgeTypes}
      nodesDraggable={false}
      nodesConnectable={false}
      nodesFocusable={false}
      edgesFocusable={false}
      elementsSelectable={false}
      preventScrolling={false}
      zoomOnScroll={false}
      zoomOnPinch={false}
      zoomOnDoubleClick={false}
      panOnDrag={false}
      panOnScroll={false}
      onNodeClick={onNodeClick}
      fitView
      fitViewOptions={{ padding: 0.2 }}
      proOptions={{ hideAttribution: true }}
    >
      {children && (
        <Panel position="top-right" className={cn(flex.row, gap[1])}>
          {children}
        </Panel>
      )}
    </ReactFlow>
  );
}
