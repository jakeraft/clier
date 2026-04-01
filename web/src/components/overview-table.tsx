import { formatDateTime } from "@/lib/format-date";
import { Table, TableBody, TableRow, TableCell, TableHead } from "@/components/ui/table";

export interface OverviewRow {
  label: string;
  children: React.ReactNode;
}

interface OverviewTableProperties {
  id?: string;
  rows: OverviewRow[];
  createdAt?: string;
  updatedAt?: string;
}

export function OverviewTable({ id, rows, createdAt, updatedAt }: Readonly<OverviewTableProperties>) {
  return (
    // overflow-hidden: required for border-radius to clip table corners
    <div className="rounded-base overflow-hidden border">
      <Table>
        <TableBody>
          {id !== undefined && (
            <TableRow>
              <TableHead className="bg-muted/50 w-[10%]">ID</TableHead>
              <TableCell>{id}</TableCell>
            </TableRow>
          )}
          {rows.map((row) => (
            <TableRow key={row.label}>
              <TableHead className="bg-muted/50 w-[10%]">{row.label}</TableHead>
              <TableCell>{row.children}</TableCell>
            </TableRow>
          ))}
          {updatedAt !== undefined && (
            <TableRow>
              <TableHead className="bg-muted/50 w-[10%]">Updated</TableHead>
              <TableCell>{formatDateTime(updatedAt)}</TableCell>
            </TableRow>
          )}
          {createdAt !== undefined && (
            <TableRow>
              <TableHead className="bg-muted/50 w-[10%]">Created</TableHead>
              <TableCell>{formatDateTime(createdAt)}</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
