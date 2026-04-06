import { useParams } from "react-router";
import { ClipboardList } from "lucide-react";
import { api } from "@/api";
import { typography } from "@/lib/typography";
import { SectionCard as Section } from "@/components/section-card";
import { OverviewTable } from "@/components/overview-table";
import { DetailLayout } from "@/components/detail-layout";
import { useDetailPage } from "@/hooks/use-detail-page";

export function SkillDetail() {
  const { id: parameterId } = useParams<{ id: string }>();
  const { data, error, loading } = useDetailPage(parameterId, api.skills.get);

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
            {
              label: "Content",
              children: <pre className="whitespace-pre-wrap text-sm">{data.content}</pre>,
            },
          ]}
        />
      </Section>
    </DetailLayout>
  );
}
