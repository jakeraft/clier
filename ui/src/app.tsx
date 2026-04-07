import { type ComponentType } from "react";
import { HashRouter, Routes, Route, Navigate, useParams } from "react-router";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AppErrorBoundary } from "@/components/app-error-boundary";
import { AppLayout } from "@/app-layout";
import { NotFound } from "@/pages/not-found";
import { ClaudeMdDetail } from "@/pages/claude-md-detail";
import { SkillDetail } from "@/pages/skill-detail";
import { SettingsDetail } from "@/pages/settings-detail";
import { ClaudeJsonDetail } from "@/pages/claude-json-detail";
import { Prompts } from "@/pages/prompts";
import { ClaudeConfig } from "@/pages/claude-config";
import { Teams } from "@/pages/teams";
import { TeamDetail } from "@/pages/team-detail";
import { Tasks } from "@/pages/tasks";
import { TaskDetail } from "@/pages/task-detail";
import { Members } from "@/pages/members";
import { MemberDetail } from "@/pages/member-detail";

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
              <Route path="/prompts" element={<Prompts />} />
              <Route path="/prompts/claude-mds/:id" element={<Keyed Component={ClaudeMdDetail} />} />
              <Route path="/prompts/skills/:id" element={<Keyed Component={SkillDetail} />} />
              <Route path="/claude" element={<ClaudeConfig />} />
              <Route path="/claude/settings/:id" element={<Keyed Component={SettingsDetail} />} />
              <Route path="/claude/claude-jsons/:id" element={<Keyed Component={ClaudeJsonDetail} />} />
              <Route path="/members" element={<Members />} />
              <Route path="/members/:id" element={<Keyed Component={MemberDetail} />} />
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
