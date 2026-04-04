import dagre from "@dagrejs/dagre";
import type { Node as DagreNode } from "@dagrejs/dagre";
import { MarkerType, Position, type Edge, type Node } from "@xyflow/react";
import type { MemberNodeData } from "@/components/team-structure/member-node";

/** Minimal member shape consumed by team-layout. */
export interface StructureMember {
  id: string;
  memberId: string;
  name: string;
  cliProfileId: string;
  systemPromptIds: string[];
}

export interface LayoutOptions {
  rootId?: string;
  cliProfileNames?: Map<string, string>;
  systemPromptNames?: Map<string, string>;
}

export const NODE_W = 200;
export const NODE_H = 120;

type Relation = { from: string; to: string; type: string };

export function buildFlowEdgeId(relation: Relation) {
  if (relation.type === "peer") {
    const [from, to] = [relation.from, relation.to].toSorted((a, b) => a.localeCompare(b));
    return `${relation.type}:${from}:${to}`;
  }
  return `${relation.type}:${relation.from}:${relation.to}`;
}

export function buildGraphShapeKey(rootId: string, members: Array<Pick<StructureMember, "id">>, relations: Relation[]) {
  return JSON.stringify({
    rootId,
    members: members.map((member) => member.id).toSorted((a, b) => a.localeCompare(b)),
    relations: relations.map((r) => buildFlowEdgeId(r)).toSorted((a, b) => a.localeCompare(b)),
  });
}

export function teamLayout(
  members: StructureMember[],
  relations: Relation[],
  options?: LayoutOptions,
): { nodes: Node<MemberNodeData>[]; edges: Edge[] } {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: "TB", nodesep: 40, ranksep: 80 });

  for (const member of members) {
    g.setNode(member.id, { width: NODE_W, height: NODE_H });
  }

  for (const relation of relations) {
    g.setEdge(relation.from, relation.to);
  }

  dagre.layout(g);

  const cliProfileNames: Map<string, string> = options?.cliProfileNames ?? new Map<string, string>();
  const spNames: Map<string, string> = options?.systemPromptNames ?? new Map<string, string>();

  const nodes: Node<MemberNodeData>[] = members.map((member) => {
    const pos = g.node(member.id) as DagreNode;
    return {
      id: member.id,
      type: "member",
      position: { x: pos.x - NODE_W / 2, y: pos.y - NODE_H / 2 },
      data: {
        name: member.name,
        memberId: member.memberId,
        isRoot: options?.rootId === member.id,
        cliProfileId: member.cliProfileId,
        cliProfileName: member.cliProfileId ? cliProfileNames.get(member.cliProfileId) : undefined,
        systemPrompts: member.systemPromptIds
          .map((id) => ({ id, name: spNames.get(id) }))
          .filter((index): index is { id: string; name: string } => index.name != undefined),
      },
      sourcePosition: Position.Bottom,
      targetPosition: Position.Top,
    };
  });

  const edgeStyle = { stroke: "var(--border)", strokeWidth: 1.5 };
  const flowEdges: Edge[] = relations.map((relation) => ({
    id: buildFlowEdgeId(relation),
    source: relation.from,
    target: relation.to,
    type: "smoothstep",
    markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16, color: "var(--border)" },
    style: relation.type === "peer" ? { ...edgeStyle, strokeDasharray: "6 4" } : edgeStyle,
    ...(relation.type === "peer" && {
      markerStart: { type: MarkerType.ArrowClosed, width: 16, height: 16, color: "var(--border)" },
    }),
  }));

  return { nodes, edges: flowEdges };
}
