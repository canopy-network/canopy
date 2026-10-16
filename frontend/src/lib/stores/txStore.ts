import { create } from "zustand";

export type TxPhase =
  | "idle"
  | "signing"
  | "submitting"
  | "waiting"
  | "confirmed"
  | "failed";

interface TxState {
  phase: TxPhase;
  txHash: string | null;
  error: string | null;
  blockHeight: number | null;
  start: () => void;
  setPhase: (phase: TxPhase) => void;
  setTxHash: (hash: string) => void;
  setError: (error: string) => void;
  setConfirmed: (height: number) => void;
  reset: () => void;
}

export const useTxStore = create<TxState>((set) => ({
  phase: "idle",
  txHash: null,
  error: null,
  blockHeight: null,
  start: () =>
    set({
      phase: "signing",
      txHash: null,
      error: null,
      blockHeight: null,
    }),
  setPhase: (phase) => set({ phase }),
  setTxHash: (txHash) => set({ txHash, phase: "waiting" }),
  setError: (error) => set({ error, phase: "failed" }),
  setConfirmed: (blockHeight) => set({ blockHeight, phase: "confirmed" }),
  reset: () =>
    set({
      phase: "idle",
      txHash: null,
      error: null,
      blockHeight: null,
    }),
}));
