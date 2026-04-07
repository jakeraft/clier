import { useParams } from "react-router";
import { ClipboardList } from "lucide-react";
import { api } from "@/api";
import { typography } from "@/lib/typography";
import { EMPTY_DATA } from "@/components/empty-cell";
import { EntityBadge } from "@/components/entity-badge";
import { EmptyEntityBadge } from "@/components/empty-entity-badge";
import { EntityBadgeList } from "@/components/entity-badge-list";
import { SectionCard as Section } from "@/components/section-card";
import { OverviewTable } from "@/components/overview-table";
import { DetailLayout } from "@/components/detail-layout";
import { useDetailPage } from "@/hooks/use-detail-page";

export function MemberDetail() {
  const { id: parameterId } = useParams<{ id: string }>();
  const { data: member, error, loading } = useDetailPage(parameterId, api.members.get);

  if (!member) return <DetailLayout error={error} loading={loading}>{undefined}</DetailLayout>;

  return (
    <DetailLayout error={error}>
      <Section icon={ClipboardList} title="Overview">
        <OverviewTable
          id={member.id}
          createdAt={member.createdAt}
          updatedAt={member.updatedAt}
          rows={[
            {
              label: "Name",
              children: <span className={typography[5]}>{member.name}</span>,
            },
            {
              label: "Model",
              children: <span className={typography[5]}>{member.model || EMPTY_DATA}</span>,
            },
            {
              label: "Args",
              children: member.args.length > 0 ? (
                <span className={typography[5]}>{member.args.join(" ")}</span>
              ) : (
                EMPTY_DATA
              ),
            },
            {
              label: "Git Repo",
              children: member.gitRepoId ? (
                <EntityBadge to="/git-repos">{member.gitRepoName || EMPTY_DATA}</EntityBadge>
              ) : (
                <EmptyEntityBadge entity="git-repo" />
              ),
            },
            {
              label: "CLAUDE.md",
              children: member.agentDotMdId ? (
                <EntityBadge to={`/prompts/agent-dot-mds/${member.agentDotMdId}`}>{member.agentDotMdName || EMPTY_DATA}</EntityBadge>
              ) : (
                <EmptyEntityBadge entity="agent-dot-md" />
              ),
            },
            {
              label: "SKILL.md",
              children: (
                <EntityBadgeList
                  entity="skill"
                  items={member.skillIds.map((id, i) => ({
                    id,
                    name: member.skillNames[i] ?? EMPTY_DATA,
                    to: `/prompts/skills/${id}`,
                  }))}
                />
              ),
            },
            {
              label: "settings.json",
              children: member.claudeSettingsId ? (
                <EntityBadge to={`/claude/claude-settings/${member.claudeSettingsId}`}>{member.claudeSettingsName || EMPTY_DATA}</EntityBadge>
              ) : (
                <EmptyEntityBadge entity="claude-settings" />
              ),
            },
            {
              label: ".claude.json",
              children: member.claudeJsonId ? (
                <EntityBadge to={`/claude/claude-jsons/${member.claudeJsonId}`}>{member.claudeJsonName || EMPTY_DATA}</EntityBadge>
              ) : (
                <EmptyEntityBadge entity="claude-json" />
              ),
            },
            {
              label: "Env",
              children: (
                <EntityBadgeList
                  entity="env"
                  items={member.envIds.map((id, i) => ({
                    id,
                    name: member.envNames[i] ?? EMPTY_DATA,
                    to: `/envs/${id}`,
                  }))}
                />
              ),
            },
          ]}
        />
      </Section>
    </DetailLayout>
  );
}
