import { type ColumnDef, type ColumnMeta, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { formatDateTime } from "@/lib/format-date";
import type { Entity } from "@/lib/entities";
import { typography } from "@/lib/typography";
import { cn } from "@/lib/utilities";
import { flex } from "@/lib/layout";
import { EmptyState } from "@/components/empty-state";
import { Spinner } from "@/components/ui/spinner";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

// ─── Types ───

interface CellMeta extends ColumnMeta<unknown, unknown> {
  className?: string;
  width?: number | string;
}

export interface EntityTableColumn<T> {
  header: string;
  cell: (item: T) => React.ReactNode;
  // variant: plain text → "muted"; badge/component columns → omit (unstyled)
  variant?: "muted";
  className?: string;
  // flex weight for width distribution among flex columns (default: 1)
  flex?: number;
}

export type EntityRow = { id: string; name: string; createdAt: string; updatedAt: string };

export interface EntityTableProperties<T extends EntityRow> {
  entityType: Entity;
  items: T[];
  ready?: boolean;
  columns: EntityTableColumn<T>[];
  empty: { title: string; description: string };
  onRowClick?: (item: T) => void;
  renderName?: (item: T) => React.ReactNode;
}

// ─── Column builder ───

function buildColumns<T extends EntityRow>(
  columns: EntityTableColumn<T>[],
  renderName: ((item: T) => React.ReactNode) | undefined,
): ColumnDef<T, unknown>[] {
  const cols: ColumnDef<T, unknown>[] = [];

  const FIXED: Record<string, number> = { name: 200, updatedAt: 200, createdAt: 200 };

  cols.push({
    id: "name",
    header: "Name",
    cell: ({ row }) => (renderName ? renderName(row.original) : row.original.name),
    meta: { className: typography[4] } satisfies CellMeta,
  });

  const VARIANT_CLASS = { muted: typography[6] } as const;
  const flexWeights = new Map<string, number>();
  for (const [index, col] of columns.entries()) {
    const id = `custom-${index}`;
    const variantClass = col.variant ? VARIANT_CLASS[col.variant] : undefined;
    const merged = [variantClass, col.className].filter(Boolean).join(" ") || undefined;
    cols.push({
      id,
      header: col.header,
      cell: ({ row }) => col.cell(row.original),
      meta: { className: merged } satisfies CellMeta,
    });
    if (col.flex != undefined) flexWeights.set(id, col.flex);
  }

  cols.push(
    {
      id: "updatedAt",
      header: "Updated",
      cell: ({ row }) => formatDateTime(row.original.updatedAt),
      meta: { className: typography[6] } satisfies CellMeta,
    },
    {
      id: "createdAt",
      header: "Created",
      cell: ({ row }) => formatDateTime(row.original.createdAt),
      meta: { className: typography[6] } satisfies CellMeta,
    },
  );

  // Fixed columns get px; flex columns share the remainder by weight
  const totalFlexWeight = cols
    .filter((col) => FIXED[col.id!] === undefined)
    .reduce((sum, col) => sum + (flexWeights.get(col.id!) ?? 1), 0);

  for (const col of cols) {
    const meta = (col.meta ?? {}) as CellMeta;
    const fixed = FIXED[col.id!];
    if (fixed === undefined) {
      const weight = flexWeights.get(col.id!) ?? 1;
      col.meta = { ...meta, width: `${(weight / totalFlexWeight) * 100}%` };
    } else {
      col.meta = { ...meta, width: `${fixed}px` };
    }
  }

  return cols;
}

// ─── Component ───

export function EntityTable<T extends EntityRow>({
  items,
  ready = true,
  columns,
  empty,
  onRowClick,
  renderName,
}: Readonly<EntityTableProperties<T>>) {
  const tanstackColumns = buildColumns(columns, renderName);

  const table = useReactTable({
    data: items,
    columns: tanstackColumns,
    getCoreRowModel: getCoreRowModel(),
    getRowId: (row) => row.id,
  });

  const headerGroups = table.getHeaderGroups();
  const rows = table.getRowModel().rows;
  const isEmpty = items.length === 0;

  return (
    <div className={flex.colFill}>
      <div className={cn(flex.colFill, "rounded-base min-h-[120px] overflow-auto border")}>
        <Table className="table-fixed">
          <TableHeader>
            {headerGroups.map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead
                    key={header.id}
                    className="truncate px-4"
                    style={{ width: (header.column.columnDef.meta as CellMeta | undefined)?.width }}
                  >
                    {header.isPlaceholder ? undefined : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow
                key={row.id}
                className={onRowClick ? "cursor-pointer" : undefined}
                onClick={onRowClick ? () => onRowClick(row.original) : undefined}
              >
                {row.getVisibleCells().map((cell) => (
                  <TableCell
                    key={cell.id}
                    className={cn("px-4", (cell.column.columnDef.meta as CellMeta | undefined)?.className)}
                  >
                    <div className="truncate">{flexRender(cell.column.columnDef.cell, cell.getContext())}</div>
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
        {!ready && (
          <div className={cn(flex.center, "flex-1")}>
            <Spinner />
          </div>
        )}
        {ready && isEmpty && (
          <div className={cn(flex.center, "flex-1")}>
            <EmptyState title={empty.title} description={empty.description} />
          </div>
        )}
      </div>
    </div>
  );
}
