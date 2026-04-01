import { useParams } from "react-router";
import { ClipboardList, AlertTriangle } from "lucide-react";
import { api } from "@/api";
import { cn } from "@/lib/utilities";
import { typography } from "@/lib/typography";
import { StateBadge } from "@/components/state-badge";
import { EntityBadge } from "@/components/entity-badge";
import { SectionCard as Section } from "@/components/section-card";
import { OverviewTable } from "@/components/overview-table";
import { DetailLayout } from "@/components/detail-layout";
import { useDetailPage } from "@/hooks/use-detail-page";

export function SprintDetail() {
  const { id: parameterId } = useParams<{ id: string }>();
  const { data: sprint, error, loading } = useDetailPage(parameterId, api.sprints.get);

  if (!sprint) return <DetailLayout error={error} loading={loading}>{undefined}</DetailLayout>;

  return (
    <DetailLayout error={error}>
      {sprint.error && (
        <Section icon={AlertTriangle} title="Error" className="text-destructive">
          <pre className={cn("rounded-base border p-3", typography[8])}>{sprint.error}</pre>
        </Section>
      )}

      <Section icon={ClipboardList} title="Overview">
        <OverviewTable
          id={sprint.id}
          createdAt={sprint.createdAt}
          rows={[
            {
              label: "Name",
              children: <span className={typography[5]}>{sprint.name}</span>,
            },
            { label: "State", children: <StateBadge state={sprint.state} /> },
            {
              label: "Team",
              children: <EntityBadge to="/teams">{sprint.teamName}</EntityBadge>,
            },
          ]}
        />
      </Section>
    </DetailLayout>
  );
}
