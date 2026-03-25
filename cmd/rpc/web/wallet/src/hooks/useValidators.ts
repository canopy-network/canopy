import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { useDSFetcher } from "@/core/dsFetch";
import { useAccountsList } from "@/app/providers/AccountsProvider";
import { useMemo } from "react";

interface Validator {
  address: string;
  publicKey: string;
  stakedAmount: number;
  unstakingAmount: number;
  unstakingHeight: number;
  pausedHeight: number;
  unstaking: boolean;
  paused: boolean;
  delegate: boolean;
  blocksProduced: number;
  rewards24h: number;
  stakeWeight: number;
  weightChange: number;
  nickname?: string;
}

export function useValidators() {
  // Use granular hook - only re-renders when accounts list changes
  const { accounts, loading: accountsLoading } = useAccountsList();
  const dsFetch = useDSFetcher();

  // Create stable query key from addresses
  const addressesKey = useMemo(
    () => accounts.map((a) => a.address).sort().join(","),
    [accounts]
  );

  return useQuery({
    queryKey: ["validators.byAccounts", addressesKey],
    enabled: !accountsLoading && accounts.length > 0,
    queryFn: async (): Promise<Validator[]> => {
      try {
        // Get all validators from the network using DS pattern
        const allValidatorsResponse = await dsFetch("validators");
        const allValidators = allValidatorsResponse || [];

        // Filter validators that belong to our accounts
        const accountAddresses = accounts.map((acc) => acc.address);
        const ourValidators = allValidators.filter((validator: any) =>
          accountAddresses.includes(validator.address),
        );

        // Map to our interface
        const validators: Validator[] = ourValidators.map((validator: any) => {
          const account = accounts.find(
            (acc) => acc.address === validator.address,
          );
          return {
            address: validator.address,
            publicKey: validator.publicKey || "",
            stakedAmount: validator.stakedAmount || 0,
            unstakingAmount: validator.unstakingAmount || 0,
            unstakingHeight: validator.unstakingHeight || 0,
            pausedHeight: validator.maxPausedHeight || 0,
            unstaking: validator.unstakingHeight > 0,
            paused: validator.maxPausedHeight > 0,
            delegate: validator.delegate || false,
            blocksProduced: 0, // This would need to be calculated separately
            rewards24h: 0, // This would need to be calculated separately
            stakeWeight: 0, // This would need to be calculated separately
            weightChange: 0, // This would need to be calculated separately
            nickname: account?.nickname,
            // Include all raw validator data to preserve committees, netAddress, etc.
            ...validator,
          };
        });

        return validators;
      } catch (error) {
        console.error("Error fetching validators:", error);
        return [];
      }
    },
    staleTime: 10000,
    retry: 2,
    retryDelay: 1000,
    placeholderData: keepPreviousData,
  });
}
