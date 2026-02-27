import { useQuery } from "@tanstack/react-query";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useDSFetcher } from "@/core/dsFetch";

interface Transaction {
  hash: string;
  height: number;
  time: number;
  transaction: {
    type: string;
    from?: string;
    to?: string;
    amount?: number;
  };
  fee: number;
  memo?: string;
  status?: string;
}

interface TransactionResponse {
  results: Transaction[];
  total: number;
  pageNumber: number;
  perPage: number;
}

export function useTransactions() {
  const { accounts, loading: accountsLoading } = useAccounts();
  const dsFetch = useDSFetcher();

  return useQuery({
    queryKey: ["transactions", accounts.map((acc: any) => acc.address)],
    enabled: !accountsLoading && accounts.length > 0,
    queryFn: async () => {
      if (accounts.length === 0) return [];

      try {
        // Fetch transactions for all accounts
        const allTransactions: Transaction[] = [];

        for (const account of accounts) {
          const [sentTxsData, receivedTxsData, failedTxsData] =
            await Promise.all([
              dsFetch("txs.sent", {
                account: { address: account.address },
                page: 1,
                perPage: 20,
              }).catch((error) => {
                console.error(
                  `Error fetching sent transactions for address ${account.address}:`,
                  error,
                );
                return { results: [] };
              }),
              dsFetch("txs.received", {
                account: { address: account.address },
                page: 1,
                perPage: 20,
              }).catch((error) => {
                console.error(
                  `Error fetching received transactions for address ${account.address}:`,
                  error,
                );
                return { results: [] };
              }),
              dsFetch("txs.failed", {
                account: { address: account.address },
                page: 1,
                perPage: 20,
              }).catch((error) => {
                console.error(
                  `Error fetching failed transactions for address ${account.address}:`,
                  error,
                );
                return { results: [] };
              }),
            ]);

          const sentTxs = sentTxsData.results || [];
          const receivedTxs = receivedTxsData.results || [];
          const failedTxs = failedTxsData.results || [];

          // Add status to transactions
          sentTxs.forEach((tx: Transaction) => (tx.status = "included"));
          receivedTxs.forEach((tx: Transaction) => (tx.status = "included"));
          failedTxs.forEach((tx: Transaction) => (tx.status = "failed"));

          allTransactions.push(...sentTxs, ...receivedTxs, ...failedTxs);
        }

        // Sort by time (most recent first) and remove duplicates
        const uniqueTransactions = allTransactions
          .filter(
            (tx, index, self) =>
              index === self.findIndex((t) => t.hash === tx.hash),
          )
          .sort((a, b) => b.time - a.time)
          .slice(0, 10); // Get latest 10 transactions

        return uniqueTransactions;
      } catch (error) {
        console.error("Error fetching transactions:", error);
        return [];
      }
    },
    staleTime: 10000,
    retry: 2,
    retryDelay: 1000,
  });
}
