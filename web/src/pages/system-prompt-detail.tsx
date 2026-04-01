import { useParams } from "react-router";
import { BookOpen, ClipboardList, Lock } from "lucide-react";
import { api } from "@/api";
import { typography } from "@/lib/typography";
import { SectionCard as Section } from "@/components/section-card";
import { DetailLayout } from "@/components/detail-layout";
import { OverviewTable } from "@/components/overview-table";
import { useDetailPage } from "@/hooks/use-detail-page";

export function SystemPromptDetail() {
  const { id: parameterId } = useParams<{ id: string }>();
  const { data: systemPrompt, error, loading } = useDetailPage(parameterId, api.systemPrompts.get);

  if (!systemPrompt) return <DetailLayout error={error} loading={loading}>{undefined}</DetailLayout>;

  return (
    <DetailLayout error={error}>
      <Section icon={ClipboardList} title="Overview">
        <OverviewTable
          id={systemPrompt.id}
          createdAt={systemPrompt.createdAt}
          updatedAt={systemPrompt.updatedAt}
          rows={[
            {
              label: "Name",
              children: (
                <span className="flex items-center gap-1.5">
                  {systemPrompt.bundled && <Lock className="size-3.5 shrink-0" />}
                  <span className={typography[5]}>{systemPrompt.name}</span>
                </span>
              ),
            },
          ]}
        />
      </Section>

      <Section icon={BookOpen} title="Prompt">
        <pre className={`rounded-base bg-muted/50 border p-3 whitespace-pre-wrap ${typography[5]}`}>
          {systemPrompt.prompt}
        </pre>
      </Section>
    </DetailLayout>
  );
}
