import { Outlet, useLocation, useNavigate } from "react-router";
import { Moon, Sun, Users, Play, User, BookOpen, Settings2, FolderGit2, KeyRound } from "lucide-react";
import { typography, typographyIcon } from "@/lib/typography";
import { cn } from "@/lib/utilities";
import { flex, gap } from "@/lib/layout";
import { PageErrorBoundary } from "@/components/page-error-boundary";
import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useTheme } from "@/hooks/use-theme";

const NAV_ITEMS = [
  { to: "/tasks", label: "Task", icon: Play },
  { to: "/teams", label: "Team", icon: Users },
  { to: "/members", label: "Member", icon: User },
  { to: "/prompts", label: "Prompt", icon: BookOpen },
  { to: "/claude", label: "Claude", icon: Settings2 },
  { to: "/git-repos", label: "Repo", icon: FolderGit2 },
  { to: "/envs", label: "Env", icon: KeyRound },
];

export function AppLayout() {
  const navigate = useNavigate();
  const { theme, toggle } = useTheme();
  return (
    <div className={cn(flex.col, "h-screen")}>
      <header className="shrink-0 border-b">
        <div className={cn(flex.row, gap[5], "mx-auto h-13 max-w-screen-xl overflow-x-auto px-5")}>
          {/* gap-0.5: tighten icon-text spacing (Button default is gap-2) */}
          <Button
            variant="ghost"
            className={cn("shrink-0", gap[1], typography[1])}
            onClick={() => {
              void navigate("/tasks");
            }}
          >
            <Logo />
            Clier
          </Button>
          <Nav />
          <div className={cn("ml-auto", flex.row, gap[1])}>
            <Toggle size="sm" pressed={theme === "dark"} onPressedChange={toggle} aria-label="Toggle theme">
              {theme === "dark" ? <Moon /> : <Sun />}
            </Toggle>
          </div>
        </div>
      </header>

      <div className={cn(flex.colFill, "mx-auto w-full max-w-screen-xl overflow-x-auto px-5")}>
        <main className={cn(flex.colFill, "overflow-auto py-4")}>
          <PageErrorBoundary>
            <Outlet />
          </PageErrorBoundary>
        </main>
      </div>
    </div>
  );
}

function Logo() {
  const terminal = (
    <>
      <rect width="18" height="18" x="3" y="3" rx="2" ry="2" className="fill-background" />
      <path d="m7 11 2-2-2-2" />
      <path d="M11 13h4" />
    </>
  );
  return (
    <svg
      viewBox="0 0 32 32"
      fill="none"
      className="size-7"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <g transform="translate(8,8)" opacity="0.4">
        {terminal}
      </g>
      <g transform="translate(4,4)" opacity="0.7">
        {terminal}
      </g>
      <g>{terminal}</g>
    </svg>
  );
}

function Nav() {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const current = NAV_ITEMS.find(({ to }) => pathname.startsWith(to))?.to ?? "";

  return (
    <ToggleGroup
      type="single"
      size="sm"
      value={current}
      onValueChange={(value) => {
        void navigate(value || current);
      }}
    >
      {/* gap-0.5: tighten icon-text spacing (toggleVariants default is gap-2) */}
      {/* text-muted-foreground + hover:text-foreground: override toggleVariants hover:text-muted-foreground */}
      {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
        <ToggleGroupItem
          key={to}
          value={to}
          aria-label={label}
          className={cn(
            gap[1],
            typography[3],
            "text-muted-foreground hover:text-foreground data-[state=on]:text-accent-foreground",
          )}
        >
          <Icon className={typographyIcon[3]} />
          {label}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  );
}
