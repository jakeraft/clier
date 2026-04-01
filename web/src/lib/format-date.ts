type DateInput = string | Date | null | undefined;

const LOCALE = "sv-SE";

function toDate(value: DateInput): Date | undefined {
  if (!value) return undefined;
  return value instanceof Date ? value : new Date(value);
}

export function formatDateTime(value: DateInput): string {
  const d = toDate(value);
  if (!d) return "-";
  return d.toLocaleString(LOCALE, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

