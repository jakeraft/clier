import { useParams } from "react-router";
import { ClipboardList, Settings2 } from "lucide-react";
import { api } from "@/api";
import { typography } from "@/lib/typography";
import { SectionCard as Section } from "@/components/section-card";
import { OverviewTable } from "@/components/overview-table";
import { DetailLayout } from "@/components/detail-layout";
import { useDetailPage } from "@/hooks/use-detail-page";

export function ClaudeSettingsDetail() {
  const { id: parameterId } = useParams<{ id: string }>();
  const { data, error, loading } = useDetailPage(parameterId, api.claudeSettings.get);

  if (!data) return <DetailLayout error={error} loading={loading}>{undefined}</DetailLayout>;

  return (
    <DetailLayout error={error}>
      <Section icon={ClipboardList} title="Overview">
        <OverviewTable
          id={data.id}
          createdAt={data.createdAt}
          updatedAt={data.updatedAt}
          rows={[
            {
              label: "Name",
              children: <span className={typography[5]}>{data.name}</span>,
            },
          ]}
        />
      </Section>

      <Section icon={Settings2} title="Content">
        <pre className={`rounded-base bg-muted/50 border p-3 whitespace-pre-wrap ${typography[5]}`}>
          {data.content}
        </pre>
      </Section>
    </DetailLayout>
  );
}
