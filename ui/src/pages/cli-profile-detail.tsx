import { useParams } from "react-router";
import { ClipboardList, Cog, Settings, SlidersHorizontal } from "lucide-react";
import { api } from "@/api";
import { typography } from "@/lib/typography";
import { EMPTY_DATA } from "@/components/empty-cell";
import { SectionCard as Section } from "@/components/section-card";
import { ArgumentsBadge } from "@/components/arguments-badge";
import { OverviewTable } from "@/components/overview-table";
import { DetailLayout } from "@/components/detail-layout";
import { useDetailPage } from "@/hooks/use-detail-page";

export function CliProfileDetail() {
  const { id: parameterId } = useParams<{ id: string }>();
  const { data: profile, error, loading } = useDetailPage(parameterId, api.cliProfiles.get);

  if (!profile) return <DetailLayout error={error} loading={loading}>{undefined}</DetailLayout>;

  return (
    <DetailLayout error={error}>
      <Section icon={ClipboardList} title="Overview">
        <OverviewTable
          id={profile.id}
          rows={[
            {
              label: "Name",
              children: <span className={typography[5]}>{profile.name}</span>,
            },
            {
              label: "Binary",
              children: profile.binary ? <span className={typography[5]}>{profile.binary}</span> : EMPTY_DATA,
            },
            {
              label: "Model",
              children: profile.model ? <span className={typography[5]}>{profile.model}</span> : EMPTY_DATA,
            },
          ]}
          createdAt={profile.createdAt}
          updatedAt={profile.updatedAt}
        />
      </Section>
      <Section icon={Cog} title="Config">
        <ArgumentsBadge
          args={Object.entries(profile.dotConfig ?? {}).map(([k, v]) => `${k}: ${typeof v === "object" ? JSON.stringify(v) : String(v)}`)}
          variant="secondary"
        />
      </Section>

      <Section icon={Settings} title="System Args">
        <ArgumentsBadge args={profile.systemArgs} variant="secondary" />
      </Section>

      <Section icon={SlidersHorizontal} title="Custom Args">
        <ArgumentsBadge args={profile.customArgs} variant="outline" />
      </Section>
    </DetailLayout>
  );
}
