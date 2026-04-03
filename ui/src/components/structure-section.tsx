import type { Node, Edge } from "@xyflow/react";
import { Network } from "lucide-react";
import { cn } from "@/lib/utilities";
import { flex } from "@/lib/layout";
import { Spinner } from "@/components/ui/spinner";
import { EmptyState } from "@/components/empty-state";
import { SectionCard as Section } from "@/components/section-card";
import { TeamStructure } from "@/components/team-structure/team-structure";

interface StructureSectionProperties {
  ready: boolean;
  empty: boolean;
  error: string | undefined;
  nodes: Node[];
  edges: Edge[];
  onNodeClick?: (event: React.MouseEvent, node: Node) => void;
}

/** Renders team structure with loading, error, and empty states. */
export function StructureSection({
  ready,
  empty,
  error,
  nodes,
  edges,
  onNodeClick,
}: Readonly<StructureSectionProperties>) {
  if (!ready) {
    return (
      <Section icon={Network} title="Structure">
        <div className={cn(flex.center, "h-[60vh]")}>
          <Spinner />
        </div>
      </Section>
    );
  }

  if (error) {
    return (
      <Section icon={Network} title="Structure">
        <EmptyState title="Failed to load structure" description={error} />
      </Section>
    );
  }

  if (empty) {
    return (
      <Section icon={Network} title="Structure">
        <EmptyState title="No structure data" />
      </Section>
    );
  }

  return (
    <Section icon={Network} title="Structure">
      <TeamStructure nodes={nodes} edges={edges} onNodeClick={onNodeClick} />
    </Section>
  );
}
