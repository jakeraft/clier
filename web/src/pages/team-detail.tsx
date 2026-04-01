import { useParams } from "react-router";
import { ClipboardList } from "lucide-react";
import { api } from "@/api";
import { typography } from "@/lib/typography";
import { EMPTY_DATA } from "@/components/empty-cell";
import { EntityBadge } from "@/components/entity-badge";
import { EmptyEntityBadge } from "@/components/empty-entity-badge";
import { EntityBadgeList } from "@/components/entity-badge-list";
import { SectionCard as Section } from "@/components/section-card";
import { DetailLayout } from "@/components/detail-layout";
import { OverviewTable } from "@/components/overview-table";
import { StructureSection } from "@/components/structure-section";
import { useDetailPage } from "@/hooks/use-detail-page";
import { useTeamStructure } from "@/hooks/use-team-structure";

export function TeamDetail() {
  const { id: parameterId } = useParams<{ id: string }>();
  const { data: team, error, loading } = useDetailPage(parameterId, api.teams.get);

  const structure = useTeamStructure(parameterId);

  if (!team) return <DetailLayout error={error} loading={loading}>{undefined}</DetailLayout>;

  return (
    <DetailLayout error={error}>
      {/* Overview */}
      <Section icon={ClipboardList} title="Overview">
        <OverviewTable
          id={team.id}
          createdAt={team.createdAt}
          updatedAt={team.updatedAt}
          rows={[
            {
              label: "Name",
              children: <span className={typography[5]}>{team.name}</span>,
            },
            {
              label: "Root",
              children: team.rootMemberId ? (
                <EntityBadge to={`/members/${team.rootMemberId}`}>
                  {team.rootMemberName || EMPTY_DATA}
                </EntityBadge>
              ) : (
                <EmptyEntityBadge entity="member" />
              ),
            },
            {
              label: "Member",
              children: (
                <EntityBadgeList
                  entity="member"
                  items={team.memberIds.map((id, i) => ({
                    id,
                    name: team.memberNames[i] ?? EMPTY_DATA,
                    to: `/members/${id}`,
                  }))}
                />
              ),
            },
          ]}
        />
      </Section>

      <StructureSection {...structure} />
    </DetailLayout>
  );
}
