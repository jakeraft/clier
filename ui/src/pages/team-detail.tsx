import { useParams } from "react-router";
import { ClipboardList, User, FileText, Terminal, FolderOpen } from "lucide-react";
import { api } from "@/api";
import type { MemberPlanView } from "@/api";
import { typography } from "@/lib/typography";
import { cn } from "@/lib/utilities";
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

  const tmMap = new Map(team.teamMembers.map((tm) => [tm.id, tm]));
  const rootTm = tmMap.get(team.rootTeamMemberId);

  return (
    <DetailLayout error={error}>
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
              children: rootTm ? (
                <EntityBadge to={`/members/${rootTm.memberId}`}>
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
                  items={team.teamMembers.map((tm, i) => ({
                    id: tm.id,
                    name: team.memberNames[i] ?? EMPTY_DATA,
                    to: `/members/${tm.memberId}`,
                  }))}
                />
              ),
            },
          ]}
        />
      </Section>

      <StructureSection {...structure} />

      {team.plan.length > 0 &&
        team.plan.map((member) => (
          <PlanMemberSection key={member.teamMemberId} member={member} />
        ))}
    </DetailLayout>
  );
}

function PlanMemberSection({ member }: Readonly<{ member: MemberPlanView }>) {
  return (
    <Section icon={User} title={member.memberName}>
        <Section icon={FolderOpen} title="Workspace">
          <OverviewTable
            rows={[
              { label: "Memberspace", children: <span className={typography[5]}>{member.memberspace}</span> },
              {
                label: "GitRepo",
                children: member.gitRepo ? (
                  <span className={typography[5]}>{member.gitRepo.url}</span>
                ) : (
                  <span className={typography[6]}>-</span>
                ),
              },
            ]}
          />

          {member.files.map((file) => (
            <Section key={file.path} icon={FileText} title={file.path}>
              <pre className={cn("rounded-base bg-muted/50 border p-3 whitespace-pre-wrap break-all", typography[5])}>
                {file.content}
              </pre>
            </Section>
          ))}
        </Section>

        <Section icon={Terminal} title="Terminal">
          <pre className={cn("rounded-base bg-muted/50 border p-3 whitespace-pre-wrap break-all", typography[5])}>
            {member.command}
          </pre>
        </Section>
    </Section>
  );
}
