import { useState, useEffect } from 'react';

export interface Account {
    id: string;
    address: string;
    nickname: string;
    publicKey: string;
    isActive: boolean;
}

export interface KeystoreResponse {
    addressMap: Record<string, {
        publicKey: string;
        salt: string;
        encrypted: string;
        keyAddress: string;
        keyNickname: string;
    }>;
    nicknameMap: Record<string, string>;
}

export const useAccounts = () => {
    const [accounts, setAccounts] = useState<Account[]>([]);
    const [activeAccount, setActiveAccount] = useState<Account | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const API_BASE_URL = 'http://localhost:50003/v1/admin';

    const fetchAccounts = async () => {
        try {
            setLoading(true);
            setError(null);
            
            const response = await fetch(`${API_BASE_URL}/keystore`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            if (!response.ok) {
                throw new Error(`Error ${response.status}: ${response.statusText}`);
            }

            const data: KeystoreResponse = await response.json();
            
            // Convert keystore response to our account format
            const accountsList: Account[] = Object.entries(data.addressMap).map(([address, keystoreEntry]) => ({
                id: address,
                address: address,
                nickname: keystoreEntry.keyNickname || `Account ${address.slice(0, 8)}...`,
                publicKey: keystoreEntry.publicKey,
                isActive: false, // Will be set based on active state
            }));

            setAccounts(accountsList);
            
            // If no active account, set the first one as active
            if (accountsList.length > 0 && !activeAccount) {
                const firstAccount = accountsList[0];
                setActiveAccount({ ...firstAccount, isActive: true });
                setAccounts(prev => prev.map(acc => 
                    acc.id === firstAccount.id 
                        ? { ...acc, isActive: true }
                        : { ...acc, isActive: false }
                ));
            }
            
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Unknown error');
            console.error('Error fetching accounts:', err);
        } finally {
            setLoading(false);
        }
    };

    const switchAccount = (accountId: string) => {
        const newActiveAccount = accounts.find(acc => acc.id === accountId);
        if (newActiveAccount) {
            setActiveAccount({ ...newActiveAccount, isActive: true });
            setAccounts(prev => prev.map(acc => ({
                ...acc,
                isActive: acc.id === accountId
            })));
        }
    };

    const createNewAccount = async (nickname: string, password: string) => {
        try {
            const response = await fetch(`${API_BASE_URL}/keystore-new-key`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    nickname,
                    password
                }),
            });

            if (!response.ok) {
                throw new Error(`Error ${response.status}: ${response.statusText}`);
            }

            const newAddress = await response.text();
            
            // Reload accounts after creating a new one
            await fetchAccounts();
            
            return newAddress.replace(/"/g, ''); // Remove quotes from response
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Error creating account');
            throw err;
        }
    };

    const deleteAccount = async (accountId: string) => {
        try {
            const account = accounts.find(acc => acc.id === accountId);
            if (!account) return;

            const response = await fetch(`${API_BASE_URL}/keystore-delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    nickname: account.nickname
                }),
            });

            if (!response.ok) {
                throw new Error(`Error ${response.status}: ${response.statusText}`);
            }

            // Reload accounts after deleting
            await fetchAccounts();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Error deleting account');
            throw err;
        }
    };

    useEffect(() => {
        fetchAccounts();
    }, []);

    return {
        accounts,
        activeAccount,
        loading,
        error,
        switchAccount,
        createNewAccount,
        deleteAccount,
        refetch: fetchAccounts
    };
};
