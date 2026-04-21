import React from "react";
import AnimatedNumber from "../AnimatedNumber";
import { formatPaginationRange } from "../../lib/utils";
import PageSizeSelect from "../shared/PageSizeSelect";

export interface DexBatchRow {
  batchType: "Locked" | "Next";
  committee: number;
  receiptHash: string;
  orders: number;
  deposits: number;
  withdraws: number;
  poolSize: number;
  totalPoolPoints: number;
  lockedHeight: number;
  receipts: number;
}

interface DexBatchesTableProps {
  rows: DexBatchRow[];
  loading?: boolean;
  title?: string;
}

const desktopHeaderClass =
  "px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4";
const desktopRowCellClass =
  "bg-[#1a1a1a] px-2 py-2 align-middle transition-colors group-hover:bg-[#272729] sm:px-3 lg:px-4";

const truncate = (s: string, start = 10, end = 8) => {
  if (s.length <= start + end) return s;
  return `${s.slice(0, start)}…${s.slice(-end)}`;
};

const columns = [
  { label: "Committee" },
  { label: "ReceiptHash" },
  { label: "Orders" },
  { label: "Deposits" },
  { label: "Withdraws" },
  { label: "PoolSize" },
  { label: "TotalPoolPoints" },
  { label: "LockedHeight" },
  { label: "Receipts" },
];

const DexBatchesTable: React.FC<DexBatchesTableProps> = ({ rows, loading = false, title }) => {
  const [currentPage, setCurrentPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(10);

  React.useEffect(() => {
    setCurrentPage(1);
  }, [rows.length, pageSize]);

  const totalCount = rows.length;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const startIdx = totalCount === 0 ? 0 : (currentPage - 1) * pageSize + 1;
  const endIdx = Math.min(currentPage * pageSize, totalCount);
  const paginatedRows = React.useMemo(() => {
    const start = (currentPage - 1) * pageSize;
    return rows.slice(start, start + pageSize);
  }, [currentPage, pageSize, rows]);

  const visiblePages = React.useMemo(() => {
    if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1);
    const pageSet = new Set([1, totalPages, currentPage - 1, currentPage, currentPage + 1]);
    return Array.from(pageSet).filter((page) => page >= 1 && page <= totalPages).sort((a, b) => a - b);
  }, [currentPage, totalPages]);

  const goToPage = (page: number) => {
    setCurrentPage(Math.min(Math.max(1, page), totalPages));
  };

  return (
    <div className="rounded-xl border border-white/10 bg-card p-5">
      {title ? (
        <div className="mb-4">
          <h2 className="wallet-card-title tracking-tight">{title}</h2>
        </div>
      ) : null}

      <div className="overflow-x-auto">
        <table
          className="w-full min-w-[1080px]"
          style={{ tableLayout: "auto", borderCollapse: "separate", borderSpacing: "0 4px" }}
        >
          <thead>
            <tr>
              {columns.map((column) => (
                <th key={column.label} className={desktopHeaderClass}>
                  {column.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              Array.from({ length: pageSize }).map((_, index) => (
                <tr key={`skeleton-${index}`} className="group animate-pulse">
                  {columns.map((_, columnIndex) => (
                    <td
                      key={`${index}-${columnIndex}`}
                      className={desktopRowCellClass}
                      style={{
                        borderTopLeftRadius: columnIndex === 0 ? "10px" : undefined,
                        borderBottomLeftRadius: columnIndex === 0 ? "10px" : undefined,
                        borderTopRightRadius: columnIndex === columns.length - 1 ? "10px" : undefined,
                        borderBottomRightRadius: columnIndex === columns.length - 1 ? "10px" : undefined,
                      }}
                    >
                      <div className={`h-4 rounded bg-white/6 ${columnIndex === 2 ? "w-36" : "w-20"}`} />
                    </td>
                  ))}
                </tr>
              ))
            ) : paginatedRows.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                  No dex batches found
                </td>
              </tr>
            ) : (
              paginatedRows.map((row, rowIndex) => (
                <tr key={`${row.batchType}-${row.committee}-${row.receiptHash}-${rowIndex}`} className="group">
                  <td
                    className={desktopRowCellClass}
                    style={{ borderTopLeftRadius: "10px", borderBottomLeftRadius: "10px" }}
                  >
                    <span className="text-sm text-white tabular-nums">{row.committee}</span>
                  </td>
                  <td className={desktopRowCellClass}>
                    <span className="block max-w-[14rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm text-white" title={row.receiptHash !== "N/A" ? row.receiptHash : undefined}>
                      {row.receiptHash !== "N/A" ? truncate(row.receiptHash) : "N/A"}
                    </span>
                  </td>
                  <td className={desktopRowCellClass}>
                    <span className="text-sm text-white tabular-nums">{row.orders}</span>
                  </td>
                  <td className={desktopRowCellClass}>
                    <span className="text-sm text-white tabular-nums">{row.deposits}</span>
                  </td>
                  <td className={desktopRowCellClass}>
                    <span className="text-sm text-white tabular-nums">{row.withdraws}</span>
                  </td>
                  <td className={desktopRowCellClass}>
                    <span className="text-sm text-white tabular-nums">{row.poolSize.toLocaleString()}</span>
                  </td>
                  <td className={desktopRowCellClass}>
                    <span className="text-sm text-white tabular-nums">{row.totalPoolPoints.toLocaleString()}</span>
                  </td>
                  <td className={desktopRowCellClass}>
                    <span className="text-sm text-white tabular-nums">{row.lockedHeight > 0 ? row.lockedHeight.toLocaleString() : "0"}</span>
                  </td>
                  <td
                    className={desktopRowCellClass}
                    style={{ borderTopRightRadius: "10px", borderBottomRightRadius: "10px" }}
                  >
                    <span className="text-sm text-white tabular-nums">{row.receipts}</span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {!loading && totalCount > 0 && (
        <div className="mt-4 flex flex-col gap-3 text-sm text-white/60 md:flex-row md:items-center md:justify-between">
          <div className="flex items-center gap-3">
            {formatPaginationRange(startIdx, endIdx)} of <AnimatedNumber value={totalCount} />
            <PageSizeSelect value={pageSize} onChange={setPageSize} />
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => goToPage(currentPage - 1)}
              disabled={currentPage === 1}
              className="explorer-pagination-button px-3 py-1.5"
              aria-label="Previous page"
            >
              <i className="fa-solid fa-angle-left" />
            </button>

            {visiblePages.map((page, index, arr) => {
              const prevPage = arr[index - 1];
              const showDots = index > 0 && page - (prevPage || 0) > 1;

              return (
                <React.Fragment key={page}>
                  {showDots && <span className="px-1 text-white/40">…</span>}
                  <button
                    type="button"
                    onClick={() => goToPage(page)}
                    className={`explorer-pagination-button explorer-pagination-page px-3 py-1.5 ${
                      currentPage === page ? "explorer-pagination-page-active" : ""
                    }`}
                  >
                    {page}
                  </button>
                </React.Fragment>
              );
            })}

            <button
              type="button"
              onClick={() => goToPage(currentPage + 1)}
              disabled={currentPage === totalPages}
              className="explorer-pagination-button px-3 py-1.5"
              aria-label="Next page"
            >
              <i className="fa-solid fa-angle-right" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default DexBatchesTable;
