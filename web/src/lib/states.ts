import { LoaderCircle, CheckCircle, XCircle, type LucideIcon } from "lucide-react";
type State = "running" | "completed" | "errored";

const STATE_STYLE = new Map<State, string>([
  ["running", "bg-info/10 text-info"],
  ["completed", "bg-success/10 text-success"],
  ["errored", "bg-destructive/10 text-destructive"],
]);

const STATE_ICON = new Map<State, LucideIcon>([
  ["running", LoaderCircle],
  ["completed", CheckCircle],
  ["errored", XCircle],
]);

export { STATE_STYLE, STATE_ICON, type State };
