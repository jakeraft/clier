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
              label: "Git Repo",
              children: member.gitRepoId ? (
                <EntityBadge to="/git-repos">{member.gitRepoName || EMPTY_DATA}</EntityBadge>
              ) : (
                <EmptyEntityBadge entity="git-repo" />
              ),
            },
            {
              label: "CLI Profile",
              children: member.cliProfileId ? (
                <EntityBadge to={`/cli-profiles/${member.cliProfileId}`}>{member.cliProfileName || EMPTY_DATA}</EntityBadge>
              ) : (
                <EmptyEntityBadge entity="cli-profile" />
              ),
            },
            {
              label: "System Prompt",
              children: (
                <EntityBadgeList
                  entity="system-prompt"
                  items={member.systemPromptIds.map((id, i) => ({
                    id,
                    name: member.systemPromptNames[i] ?? EMPTY_DATA,
                    to: `/system-prompts/${id}`,
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
