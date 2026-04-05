import { type ComponentType } from "react";
import { HashRouter, Routes, Route, Navigate, useParams } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AppErrorBoundary } from "@/components/app-error-boundary";
import { AppLayout } from "@/app-layout";
import { NotFound } from "@/pages/not-found";
import { SystemPrompts } from "@/pages/system-prompts";
import { SystemPromptDetail } from "@/pages/system-prompt-detail";
import { Teams } from "@/pages/teams";
import { TeamDetail } from "@/pages/team-detail";
import { Tasks } from "@/pages/tasks";
import { TaskDetail } from "@/pages/task-detail";
import { CliProfiles } from "@/pages/cli-profiles";
import { CliProfileDetail } from "@/pages/cli-profile-detail";
import { Members } from "@/pages/members";
import { MemberDetail } from "@/pages/member-detail";
import { GitRepos } from "@/pages/git-repos";
import { Envs } from "@/pages/envs";

// React Router reuses the same component instance when only the :id param changes
// (e.g. /members/A → /members/B). This preserves all hook state, causing stale data bugs.
// key={id} forces a full remount so every detail page starts with clean state.
function Keyed({ Component }: Readonly<{ Component: ComponentType }>) {
  const { id } = useParams();
  return <Component key={id} />;
}

export default function App() {
  return (
    <AppErrorBoundary>
      <TooltipProvider delayDuration={0}>
        <HashRouter>
          <Routes>
            <Route element={<AppLayout />}>
              <Route path="/" element={<Navigate to="/tasks" replace />} />
              <Route path="/system-prompts" element={<SystemPrompts />} />
              <Route path="/system-prompts/:id" element={<Keyed Component={SystemPromptDetail} />} />
              <Route path="/cli-profiles" element={<CliProfiles />} />
              <Route path="/cli-profiles/:id" element={<Keyed Component={CliProfileDetail} />} />
              <Route path="/members" element={<Members />} />
              <Route path="/members/:id" element={<Keyed Component={MemberDetail} />} />
              <Route path="/git-repos" element={<GitRepos />} />
              <Route path="/envs" element={<Envs />} />
              <Route path="/teams" element={<Teams />} />
              <Route path="/teams/:id" element={<Keyed Component={TeamDetail} />} />
              <Route path="/tasks" element={<Tasks />} />
              <Route path="/tasks/:id" element={<Keyed Component={TaskDetail} />} />
              <Route path="*" element={<NotFound />} />
            </Route>
          </Routes>
        </HashRouter>
      </TooltipProvider>
    </AppErrorBoundary>
  );
}
