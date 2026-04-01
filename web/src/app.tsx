import { type ComponentType, useEffect, useState } from "react";
import { BrowserRouter, Routes, Route, Navigate, useParams } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AppErrorBoundary } from "@/components/app-error-boundary";
import { AppLayout } from "@/app-layout";
import { api } from "@/api";
import type { BrandInfo } from "@/api";
import { BrandContext } from "@/hooks/use-brand";
import { SystemPrompts } from "@/pages/system-prompts";
import { SystemPromptDetail } from "@/pages/system-prompt-detail";
import { Teams } from "@/pages/teams";
import { TeamDetail } from "@/pages/team-detail";
import { Sprints } from "@/pages/sprints";
import { SprintDetail } from "@/pages/sprint-detail";
import { CliProfiles } from "@/pages/cli-profiles";
import { CliProfileDetail } from "@/pages/cli-profile-detail";
import { Members } from "@/pages/members";
import { MemberDetail } from "@/pages/member-detail";
import { Environments } from "@/pages/environments";
import { GitRepos } from "@/pages/git-repos";

// React Router reuses the same component instance when only the :id param changes
// (e.g. /sprints/A → /sprints/B). This preserves all hook state, causing stale data bugs.
// key={id} forces a full remount so every detail page starts with clean state.
function Keyed({ Component }: Readonly<{ Component: ComponentType }>) {
  const { id } = useParams();
  return <Component key={id} />;
}

export default function App() {
  const [brand, setBrand] = useState<BrandInfo | null>(null);

  useEffect(() => {
    api.system.info().then((info) => {
      setBrand(info);
      document.title = info.displayName;
    }, () => {});
  }, []);

  if (!brand) return null;

  return (
    <AppErrorBoundary>
      <BrandContext value={brand}>
      <TooltipProvider delayDuration={0}>
        <BrowserRouter>
          <Routes>
            <Route element={<AppLayout />}>
              <Route path="/" element={<Navigate to="/teams" replace />} />
              <Route path="/system-prompts" element={<SystemPrompts />} />
              <Route path="/system-prompts/:id" element={<Keyed Component={SystemPromptDetail} />} />
              <Route path="/cli-profiles" element={<CliProfiles />} />
              <Route path="/cli-profiles/:id" element={<Keyed Component={CliProfileDetail} />} />
              <Route path="/members" element={<Members />} />
              <Route path="/members/:id" element={<Keyed Component={MemberDetail} />} />
              <Route path="/environments" element={<Environments />} />
              <Route path="/git-repos" element={<GitRepos />} />
              <Route path="/teams" element={<Teams />} />
              <Route path="/teams/:id" element={<Keyed Component={TeamDetail} />} />
              <Route path="/sprints" element={<Sprints />} />
              <Route path="/sprints/:id" element={<Keyed Component={SprintDetail} />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </TooltipProvider>
      </BrandContext>
    </AppErrorBoundary>
  );
}
