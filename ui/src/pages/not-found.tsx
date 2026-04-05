import { useLocation } from "react-router";
import { ErrorAlert } from "@/components/error-alert";

export function NotFound() {
  const { pathname } = useLocation();
  return (
    <div className="p-6">
      <ErrorAlert message={`No page matches "${pathname}".`} />
    </div>
  );
}
