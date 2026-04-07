import { useState } from "react";
import { FileText, BookOpen } from "lucide-react";
import { api } from "@/api";
import type { AgentDotMdView, SkillView } from "@/api";
import { typography, typographyIcon } from "@/lib/typography";
import { cn } from "@/lib/utilities";
import { gap } from "@/lib/layout";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

type Tab = "agent-dot-md" | "skill";

const agentDotMdColumns: EntityTableColumn<AgentDotMdView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

const skillColumns: EntityTableColumn<SkillView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

export function Prompts() {
  const [tab, setTab] = useState<Tab>("agent-dot-md");

  return (
    <>
      <ToggleGroup
        type="single"
        size="sm"
        value={tab}
        onValueChange={(v) => { if (v) setTab(v as Tab); }}
      >
        <ToggleGroupItem
          value="agent-dot-md"
          className={cn(
            gap[1],
            typography[3],
            "text-muted-foreground hover:text-foreground data-[state=on]:text-accent-foreground",
          )}
        >
          <FileText className={typographyIcon[3]} />
          CLAUDE.md
        </ToggleGroupItem>
        <ToggleGroupItem
          value="skill"
          className={cn(
            gap[1],
            typography[3],
            "text-muted-foreground hover:text-foreground data-[state=on]:text-accent-foreground",
          )}
        >
          <BookOpen className={typographyIcon[3]} />
          SKILL.md
        </ToggleGroupItem>
      </ToggleGroup>

      {tab === "agent-dot-md" && (
        <EntityListPage<AgentDotMdView>
          entityType="agent-dot-md"
          apiList={api.agentDotMds.list}
          columns={agentDotMdColumns}
          empty={{ title: "No CLAUDE.md files yet", description: "CLAUDE.md project instructions for team members" }}
          routeBase="/prompts/agent-dot-mds"
        />
      )}
      {tab === "skill" && (
        <EntityListPage<SkillView>
          entityType="skill"
          apiList={api.skills.list}
          columns={skillColumns}
          empty={{ title: "No SKILL.md files yet", description: "Claude Code skills for team members" }}
          routeBase="/prompts/skills"
        />
      )}
    </>
  );
}
