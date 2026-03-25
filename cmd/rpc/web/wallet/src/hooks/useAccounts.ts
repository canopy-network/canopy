import { useDS } from "@/core/useDs";

export interface Account {
  address: string;
  nickname?: string;
  balance?: number;
  stakedAmount?: number;
  publicKey?: string;
  type?: "local" | "imported";
}

export interface AccountsState {
  accounts: Account[];
  selectedAccount: Account | null;
  isLoading: boolean;
}

export const useAccounts = () => {
  const { data: accountsData, isLoading } = useDS<any>(
    "account",
    {},
    {
      staleTimeMs: 10000,
      refetchIntervalMs: 30000,
      refetchOnMount: true,
      refetchOnWindowFocus: false,
      select: (data) => {
        if (!data) return { accounts: [], selectedAccount: null };

        // Handle single account case
        if (data.address) {
          const account: Account = {
            address: data.address,
            nickname: data.nickname || "Account 1",
            balance: data.amount || 0,
            stakedAmount: data.stakedAmount || 0,
            publicKey: data.publicKey,
            type: "local",
          };
          return {
            accounts: [account],
            selectedAccount: account,
          };
        }

        // Handle multiple accounts case
        if (Array.isArray(data)) {
          const accounts = data.map((acc, index) => ({
            address: acc.address,
            nickname: acc.nickname || `Account ${index + 1}`,
            balance: acc.amount || 0,
            stakedAmount: acc.stakedAmount || 0,
            publicKey: acc.publicKey,
            type: acc.type || "local",
          }));
          return {
            accounts,
            selectedAccount: accounts[0] || null,
          };
        }

        return { accounts: [], selectedAccount: null };
      },
    },
  );

  const accounts = accountsData?.accounts || [];
  const selectedAccount = accountsData?.selectedAccount || null;

  return {
    accounts,
    selectedAccount,
    isLoading,
    // Helper methods
    getAccount: (address: string) =>
      accounts.find((acc: Account) => acc.address === address),
    hasAccount: (address: string) =>
      accounts.some((acc: Account) => acc.address === address),
    totalBalance: accounts.reduce(
      (sum: number, acc: Account) => sum + (acc.balance || 0),
      0,
    ),
    totalStaked: accounts.reduce(
      (sum: number, acc: Account) => sum + (acc.stakedAmount || 0),
      0,
    ),
  };
};
