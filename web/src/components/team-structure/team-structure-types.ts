import type { Edge, EdgeTypes, Node } from "@xyflow/react";
import type { MouseEvent, ReactNode } from "react";

export interface TeamStructureProperties {
  nodes: Node[];
  edges: Edge[];
  edgeTypes?: EdgeTypes;
  onNodeClick?: (event: MouseEvent, node: Node) => void;
  children?: ReactNode;
  className?: string;
}
