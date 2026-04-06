import { memo } from "react";
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { typography } from "@/lib/typography";
import { EntityBadge } from "@/components/entity-badge";
import { NODE_H, NODE_W } from "@/components/team-structure/team-layout";

export interface MemberNodeData extends Record<string, unknown> {
  name: string;
  memberId: string;
  isRoot: boolean;
  model: string;
  skillCount: number;
}

type MemberNodeType = Node<MemberNodeData, "member">;

const MemberNode = memo(function MemberNode({ id, data }: NodeProps<MemberNodeType>) {
  return (
    <>
      <Handle type="target" position={Position.Top} className={"!size-0 !opacity-0"} />
      <div
        className={cn(
          flex.col,
          flex.center,
          gap[1],
          "rounded-base bg-card relative h-full w-full border px-3 py-2 shadow-sm",
          "cursor-pointer pointer-events-auto",
        )}
        style={{ width: NODE_W, height: NODE_H }}
        title={data.name}
      >
        {/* Model */}
        <div className={cn(flex.center, "w-full")}>
          <span className={cn(typography[5], "text-muted-foreground truncate")}>{data.model || "no model"}</span>
        </div>

        {/* Member name */}
        <div className={cn(flex.center, "w-full")}>
          <EntityBadge to={`/members/${data.memberId}`}>{data.name}</EntityBadge>
        </div>
      </div>
      <Handle type="source" position={Position.Bottom} className={"!size-0 !opacity-0"} />
    </>
  );
});

export const nodeTypes = { member: MemberNode };
