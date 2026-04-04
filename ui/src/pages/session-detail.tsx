import { useState } from "react";
import { useParams } from "react-router";
import { ClipboardList, Map, ScrollText, MessageSquare, User, FileText, Terminal, FolderOpen } from "lucide-react";
import { api } from "@/api";
import type { MemberPlanView, LogView, MessageView } from "@/api";
import { typography, typographyIcon } from "@/lib/typography";
import { cn } from "@/lib/utilities";
import { gap } from "@/lib/layout";
import { formatDateTime } from "@/lib/format-date";
import { EntityBadge } from "@/components/entity-badge";
import { SectionCard as Section } from "@/components/section-card";
import { DetailLayout } from "@/components/detail-layout";
import { OverviewTable } from "@/components/overview-table";
import { useDetailPage } from "@/hooks/use-detail-page";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

type Tab = "overview" | "plan";

export function SessionDetail() {
  const { id: parameterId } = useParams<{ id: string }>();
  const { data: session, error, loading } = useDetailPage(parameterId, api.sessions.get);
  const [tab, setTab] = useState<Tab>("overview");

  if (!session) return <DetailLayout error={error} loading={loading}>{undefined}</DetailLayout>;

  return (
    <DetailLayout error={error}>
      <ToggleGroup
        type="single"
        size="sm"
        value={tab}
        onValueChange={(v) => { if (v) setTab(v as Tab); }}
      >
        {([
          { value: "overview" as const, label: "Overview", icon: ClipboardList },
          { value: "plan" as const, label: "Plan", icon: Map },
        ]).map(({ value, label, icon: Icon }) => (
          <ToggleGroupItem
            key={value}
            value={value}
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

      {tab === "overview" && <OverviewTab session={session} />}
      {tab === "plan" && <PlanTab plan={session.plan} />}
    </DetailLayout>
  );
}

function OverviewTab({ session }: Readonly<{ session: { id: string; status: string; teamId: string; teamName: string; createdAt: string; updatedAt: string; logs: LogView[]; messages: MessageView[] } }>) {
  return (
    <>
      <Section icon={ClipboardList} title="Overview">
        <OverviewTable
          id={session.id}
          createdAt={session.createdAt}
          updatedAt={session.updatedAt}
          rows={[
            {
              label: "Status",
              children: <span className={typography[5]}>{session.status}</span>,
            },
            {
              label: "Team",
              children: <EntityBadge to={`/teams/${session.teamId}`}>{session.teamName}</EntityBadge>,
            },
          ]}
        />
      </Section>

      <Section
        icon={ScrollText}
        title="Logs"
        empty={session.logs.length === 0 ? { title: "No logs yet", description: "Logs will appear when members record entries" } : undefined}
      >
        {session.logs.length > 0 && <LogTable logs={session.logs} />}
      </Section>

      <Section
        icon={MessageSquare}
        title="Messages"
        empty={session.messages.length === 0 ? { title: "No messages yet", description: "Messages will appear when members communicate" } : undefined}
      >
        {session.messages.length > 0 && <MessageTable messages={session.messages} />}
      </Section>
    </>
  );
}

function LogTable({ logs }: Readonly<{ logs: LogView[] }>) {
  return (
    <div className="rounded-base overflow-hidden border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[15%]">Member</TableHead>
            <TableHead>Content</TableHead>
            <TableHead className="w-[20%]">Created</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {logs.map((log) => (
            <TableRow key={log.id}>
              <TableCell className={typography[5]}>{log.memberName}</TableCell>
              <TableCell className={cn(typography[5], "whitespace-pre-wrap break-all")}>{log.content}</TableCell>
              <TableCell className={typography[6]}>{formatDateTime(log.createdAt)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function MessageTable({ messages }: Readonly<{ messages: MessageView[] }>) {
  return (
    <div className="rounded-base overflow-hidden border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[12%]">From</TableHead>
            <TableHead className="w-[12%]">To</TableHead>
            <TableHead>Content</TableHead>
            <TableHead className="w-[20%]">Created</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {messages.map((msg) => (
            <TableRow key={msg.id}>
              <TableCell className={typography[5]}>{msg.fromMemberName || "-"}</TableCell>
              <TableCell className={typography[5]}>{msg.toMemberName}</TableCell>
              <TableCell className={cn(typography[5], "whitespace-pre-wrap break-all")}>{msg.content}</TableCell>
              <TableCell className={typography[6]}>{formatDateTime(msg.createdAt)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function PlanTab({ plan }: Readonly<{ plan: MemberPlanView[] }>) {
  if (plan.length === 0) {
    return (
      <Section
        icon={ClipboardList}
        title="Plan"
        empty={{ title: "No plan", description: "Plan is built when the session starts" }}
      />
    );
  }
  return (
    <>
      {plan.map((member) => (
        <PlanMemberSection key={member.teamMemberId} member={member} />
      ))}
    </>
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
