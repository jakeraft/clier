import { AlertCircle } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";

export function ErrorAlert({ message }: Readonly<{ message: string }>) {
  return (
    <Alert className="bg-destructive/10 text-destructive border-destructive/20">
      <AlertCircle className="size-4" />
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}
