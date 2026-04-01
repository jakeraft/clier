import { memo } from "react";
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { EntityBadge } from "@/components/entity-badge";
import { EmptyEntityBadge } from "@/components/empty-entity-badge";
import { EntityBadgeList } from "@/components/entity-badge-list";
import { NODE_H, NODE_W } from "@/components/team-structure/team-layout";

export interface MemberNodeData extends Record<string, unknown> {
  name: string;
  isRoot: boolean;
  cliProfileId?: string;
  cliProfileName?: string;
  systemPrompts: { id: string; name: string }[];
}

type MemberNodeType = Node<MemberNodeData, "member">;

const MemberNode = memo(function MemberNode({ id, data }: NodeProps<MemberNodeType>) {
  return (
    <>
      <Handle type="target" position={Position.Left} className={"!size-0 !opacity-0"} />
      <div
        className={cn(
          flex.col,
          flex.center,
          gap[1],
          "rounded-base bg-card relative h-full w-full border px-3 py-2 shadow-sm",
          "cursor-pointer",
        )}
        style={{ width: NODE_W, height: NODE_H }}
        title={data.name}
      >
        {/* CLI Profile */}
        <div className={cn(flex.center, "w-full")}>
          {data.cliProfileId && data.cliProfileName ? (
            <EntityBadge to={`/cli-profiles/${data.cliProfileId}`}>{data.cliProfileName}</EntityBadge>
          ) : (
            <EmptyEntityBadge entity="cli-profile" />
          )}
        </div>

        {/* System Prompts */}
        <div className={cn(flex.center, "w-full")}>
          <EntityBadgeList
            entity="system-prompt"
            items={data.systemPrompts.map((sp) => ({
              id: sp.id,
              name: sp.name,
              to: `/system-prompts/${sp.id}`,
            }))}
          />
        </div>

        {/* Member name */}
        <div className={cn(flex.center, "w-full")}>
          <EntityBadge to={`/members/${id}`}>{data.name}</EntityBadge>
        </div>
      </div>
      <Handle type="source" position={Position.Right} className={"!size-0 !opacity-0"} />
    </>
  );
});

export const nodeTypes = { member: MemberNode };
