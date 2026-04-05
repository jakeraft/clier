import { useLocation, useNavigate } from "react-router";
import { cn } from "@/lib/utilities";
import { typography } from "@/lib/typography";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

export function NotFound() {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  return (
    <div className="p-6">
      <Alert variant="destructive">
        <AlertTitle>Page not found</AlertTitle>
        <AlertDescription>No page matches "{pathname}".</AlertDescription>
      </Alert>
      <div className="mt-4">
        <button
          type="button"
          className={cn("bg-primary text-primary-foreground rounded-md px-4 py-2", typography[5])}
          onClick={() => {
            void navigate("/tasks");
          }}
        >
          Go to home
        </button>
      </div>
    </div>
  );
}
