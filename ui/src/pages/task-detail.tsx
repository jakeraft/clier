import { useState } from "react";
import { useParams } from "react-router";
import { ClipboardList, Map, ScrollText, MessageSquare, User, FileText, Terminal, FolderOpen } from "lucide-react";
import { api } from "@/api";
import type { MemberPlanView, NoteView, MessageView } from "@/api";
import { typography, typographyIcon } from "@/lib/typography";
import { cn } from "@/lib/utilities";
import { gap } from "@/lib/layout";
import { formatDateTime } from "@/lib/format-date";
import { EntityBadge } from "@/components/entity-badge";
import { StatusBadge } from "@/components/status-badge";
import { SectionCard as Section } from "@/components/section-card";
import { DetailLayout } from "@/components/detail-layout";
import { OverviewTable } from "@/components/overview-table";
import { useDetailPage } from "@/hooks/use-detail-page";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

type Tab = "overview" | "plan";

export function TaskDetail() {
  const { id: parameterId } = useParams<{ id: string }>();
  const { data: task, error, loading } = useDetailPage(parameterId, api.tasks.get);
  const [tab, setTab] = useState<Tab>("overview");

  if (!task) return <DetailLayout error={error} loading={loading}>{undefined}</DetailLayout>;

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

      {tab === "overview" && <OverviewTab task={task} />}
      {tab === "plan" && <PlanTab plan={task.plan} />}
    </DetailLayout>
  );
}

function OverviewTab({ task }: Readonly<{ task: { id: string; status: string; teamId: string; teamName: string; createdAt: string; updatedAt: string; notes: NoteView[]; messages: MessageView[] } }>) {
  return (
    <>
      <Section icon={ClipboardList} title="Overview">
        <OverviewTable
          id={task.id}
          createdAt={task.createdAt}
          updatedAt={task.updatedAt}
          rows={[
            {
              label: "Status",
              children: <StatusBadge status={task.status} />,
            },
            {
              label: "Team",
              children: <EntityBadge to={`/teams/${task.teamId}`}>{task.teamName}</EntityBadge>,
            },
          ]}
        />
      </Section>

      <Section
        icon={ScrollText}
        title="Notes"
        empty={task.notes.length === 0 ? { title: "No notes yet", description: "Notes will appear when members post progress" } : undefined}
      >
        {task.notes.length > 0 && <NoteTable notes={task.notes} />}
      </Section>

      <Section
        icon={MessageSquare}
        title="Messages"
        empty={task.messages.length === 0 ? { title: "No messages yet", description: "Messages will appear when members communicate" } : undefined}
      >
        {task.messages.length > 0 && <MessageTable messages={task.messages} />}
      </Section>
    </>
  );
}

function NoteTable({ notes }: Readonly<{ notes: NoteView[] }>) {
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
          {notes.map((n) => (
            <TableRow key={n.id}>
              <TableCell className={typography[5]}>{n.memberName}</TableCell>
              <TableCell className={cn(typography[5], "whitespace-pre-wrap break-all")}>{n.content}</TableCell>
              <TableCell className={typography[6]}>{formatDateTime(n.createdAt)}</TableCell>
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
        empty={{ title: "No plan", description: "Plan is built when the task starts" }}
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
