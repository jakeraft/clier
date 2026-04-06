import { useState } from "react";
import { Settings2, FileJson } from "lucide-react";
import { api } from "@/api";
import type { SettingsView, ClaudeJsonView } from "@/api";
import { typography, typographyIcon } from "@/lib/typography";
import { cn } from "@/lib/utilities";
import { gap } from "@/lib/layout";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

type Tab = "settings" | "claude-json";

const settingsColumns: EntityTableColumn<SettingsView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

const claudeJsonColumns: EntityTableColumn<ClaudeJsonView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

export function ClaudeConfig() {
  const [tab, setTab] = useState<Tab>("settings");

  return (
    <>
      <ToggleGroup
        type="single"
        size="sm"
        value={tab}
        onValueChange={(v) => { if (v) setTab(v as Tab); }}
      >
        <ToggleGroupItem
          value="settings"
          className={cn(
            gap[1],
            typography[3],
            "text-muted-foreground hover:text-foreground data-[state=on]:text-accent-foreground",
          )}
        >
          <Settings2 className={typographyIcon[3]} />
          settings.json
        </ToggleGroupItem>
        <ToggleGroupItem
          value="claude-json"
          className={cn(
            gap[1],
            typography[3],
            "text-muted-foreground hover:text-foreground data-[state=on]:text-accent-foreground",
          )}
        >
          <FileJson className={typographyIcon[3]} />
          .claude.json
        </ToggleGroupItem>
      </ToggleGroup>

      {tab === "settings" && (
        <EntityListPage<SettingsView>
          entityType="claude-settings"
          apiList={api.settings.list}
          columns={settingsColumns}
          empty={{ title: "No settings.json yet", description: "Claude Code settings.json configurations" }}
          routeBase="/claude/settings"
        />
      )}
      {tab === "claude-json" && (
        <EntityListPage<ClaudeJsonView>
          entityType="claude-json"
          apiList={api.claudeJsons.list}
          columns={claudeJsonColumns}
          empty={{ title: "No .claude.json yet", description: "Claude Code .claude.json project configurations" }}
          routeBase="/claude/claude-jsons"
        />
      )}
    </>
  );
}
