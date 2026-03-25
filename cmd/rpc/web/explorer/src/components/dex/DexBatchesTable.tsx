import React from "react";
import TableCard from "../Home/TableCard";

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
}

const truncate = (s: string, start = 10, end = 8) => {
  if (s.length <= start + end) return s;
  return `${s.slice(0, start)}…${s.slice(-end)}`;
};

const columns = [
  { label: "BatchType", width: "w-[9%]" },
  { label: "Committee", width: "w-[8%]" },
  { label: "ReceiptHash", width: "w-[18%]" },
  { label: "Orders", width: "w-[7%]" },
  { label: "Deposits", width: "w-[7%]" },
  { label: "Withdraws", width: "w-[7%]" },
  { label: "PoolSize", width: "w-[9%]" },
  { label: "TotalPoolPoints", width: "w-[11%]" },
  { label: "LockedHeight", width: "w-[9%]" },
  { label: "Receipts", width: "w-[7%]" },
];

const DexBatchesTable: React.FC<DexBatchesTableProps> = ({ rows, loading = false }) => {
  const tableRows = rows.map((row) => [
    <span className="text-white font-medium text-sm">{row.batchType}</span>,
    <span className="text-white font-medium">{row.committee}</span>,
    <span className="text-primary font-mono text-xs" title={row.receiptHash !== "N/A" ? row.receiptHash : undefined}>
      {row.receiptHash !== "N/A" ? truncate(row.receiptHash) : <span className="text-gray-500">N/A</span>}
    </span>,
    <span className="text-white font-medium">{row.orders}</span>,
    <span className="text-white font-medium">{row.deposits}</span>,
    <span className="text-white font-medium">{row.withdraws}</span>,
    <span className="text-white font-medium">{row.poolSize.toLocaleString()}</span>,
    <span className="text-white font-medium">{row.totalPoolPoints.toLocaleString()}</span>,
    <span className="text-white font-medium">
      {row.lockedHeight > 0 ? row.lockedHeight.toLocaleString() : <span className="text-gray-500">0</span>}
    </span>,
    <span className="text-white font-medium">{row.receipts}</span>,
  ]);

  return (
    <TableCard title="Dex Batches" live={false} columns={columns} rows={tableRows} loading={loading} spacing={4} />
  );
};

export default DexBatchesTable;
