import { useState, useEffect } from 'react'
import { useTxByHash, useAllValidators } from './useApi'
import {
    getModalData,
    BlockByHeight,
    BlockByHash,
    TxByHash,
    Validator,
    Account
} from '../lib/api'

interface SearchResult {
    type: 'block' | 'transaction' | 'address' | 'validator'
    id: string
    title: string
    subtitle?: string
    data: any
}

interface SearchResults {
    total: number
    blocks: SearchResult[]
    transactions: SearchResult[]
    addresses: SearchResult[]
    validators: SearchResult[]
}

export const useSearch = (searchTerm: string) => {
    const [results, setResults] = useState<SearchResults | null>(null)
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    // Detect if search term is a transaction hash
    const isHashSearch = searchTerm && searchTerm.length >= 32 && /^[a-fA-F0-9]+$/.test(searchTerm)

    // Solo usa el hook para hash exactos
    const { data: hashSearchData } = useTxByHash(isHashSearch ? searchTerm : '')

    // Get all validators for partial address search
    const { data: allValidatorsData } = useAllValidators()

    const searchInData = async (term: string) => {
        if (!term.trim()) {
            setResults(null)
            return
        }

        setLoading(true)
        setError(null)

        // Clear previous results immediately
        setResults(null)

        try {
            const searchResults: SearchResults = {
                total: 0,
                blocks: [],
                transactions: [],
                addresses: [],
                validators: []
            }

            // DIRECT SEARCH FOR BLOCKS, TRANSACTIONS, ACCOUNTS, AND VALIDATORS
            const searchPromises: Promise<any>[] = []

            // Check if term is hexadecimal (could be address, hash, etc.)
            const isHex = /^[a-fA-F0-9]+$/.test(term)

            // Check if it's likely an address (8-40 hex chars) - addresses take priority
            const isLikelyAddress = isHex && term.length >= 8 && term.length <= 40

            // 1. If it looks like a transaction/block hash (32+ hexadecimal characters, but NOT an address)
            // Only search for blocks/transactions if it's NOT a partial address
            if (isHex && term.length >= 32 && !isLikelyAddress) {
                searchPromises.push(
                    TxByHash(term)
                        .then(tx => {
                            if (tx && (tx.sender || tx.txHash || tx.hash)) {
                                searchResults.transactions.push({
                                    type: 'transaction' as const,
                                    id: tx.txHash || tx.hash || term,
                                    title: 'Transaction',
                                    subtitle: `Hash: ${term.slice(0, 16)}...`,
                                    data: tx
                                })
                            }
                        })
                        .catch(err => console.log('Transaction search error:', err))
                )

                // It could also be a block hash (only if NOT an address)
                searchPromises.push(
                    BlockByHash(term)
                        .then(block => {
                            if (block && block.blockHeader && block.blockHeader.hash) {
                                searchResults.blocks.push({
                                    type: 'block' as const,
                                    id: block.blockHeader.hash || term,
                                    title: `Block #${block.blockHeader.height}`,
                                    subtitle: `Hash: ${(block.blockHeader.hash || term).slice(0, 16)}...`,
                                    data: block
                                })
                            }
                        })
                        .catch(err => console.log('Block hash search error:', err))
                )
            }

            // 2. If it is an address (40 hexadecimal characters) OR a partial address (8-39 hex chars)
            // PRIORITY: Validators first, then addresses
            if (isLikelyAddress) {
                // If it's exactly 40 chars, try as complete address
                if (term.length === 40) {
                    // First check if it's a validator AND also search for account
                    // Both can exist for the same address
                    searchPromises.push(
                        Validator(0, term)
                            .then(validator => {
                                if (validator && validator.address) {
                                    const validatorResult = {
                                        type: 'validator' as const,
                                        id: validator.address,
                                        title: validator.name || 'Validator',
                                        subtitle: `Address: ${validator.address.slice(0, 16)}...`,
                                        data: validator
                                    }

                                    // Verificar duplicados
                                    if (!searchResults.validators.some(v => v.id === validator.address)) {
                                        searchResults.validators.push(validatorResult)
                                    }
                                }
                                
                                // Always try to search as account too (even if it's a validator)
                                // A validator can also have an account
                                return Account(0, term)
                                    .then(account => {
                                        if (account && account.address) {
                                            const accountResult = {
                                                type: 'address' as const,
                                                id: account.address,
                                                title: 'Account',
                                                subtitle: `Balance: ${(account.amount / 1000000).toLocaleString()} CNPY`,
                                                data: account
                                            }

                                            // Verificar duplicados
                                            if (!searchResults.addresses.some(a => a.id === account.address)) {
                                                searchResults.addresses.push(accountResult)
                                            }
                                        }
                                    })
                                    .catch(err => console.log('Account search error:', err))
                            })
                            .catch(() => {
                                // If validator search fails, try as account
                                return Account(0, term)
                                    .then(account => {
                                        if (account && account.address) {
                                            const accountResult = {
                                                type: 'address' as const,
                                                id: account.address,
                                                title: 'Account',
                                                subtitle: `Balance: ${(account.amount / 1000000).toLocaleString()} CNPY`,
                                                data: account
                                            }

                                            // Verificar duplicados
                                            if (!searchResults.addresses.some(a => a.id === account.address)) {
                                                searchResults.addresses.push(accountResult)
                                            }
                                        }
                                    })
                                    .catch(err => console.log('Account search error:', err))
                            })
                    )

                    // Also try getModalData as fallback
                    searchPromises.push(
                        getModalData(term, 1)
                            .then(result => {
                                if (result && result !== "no result found") {
                                    // Si es un validador, agregarlo como validador
                                    if (result.validator) {
                                        const validatorId = result.validator.address
                                        // Solo agregar si no existe ya como validador
                                        if (!searchResults.validators.some(v => v.id === validatorId)) {
                                            searchResults.validators.push({
                                                type: 'validator' as const,
                                                id: validatorId,
                                                title: result.validator.name || 'Validator',
                                                subtitle: `Address: ${validatorId.slice(0, 16)}...`,
                                                data: result.validator
                                            })
                                        }
                                    }
                                    
                                    // También agregar como cuenta si existe (incluso si es validador)
                                    if (result.account) {
                                        const accountId = result.account.address
                                        // Agregar como address incluso si es validador (ambos pueden existir)
                                        if (!searchResults.addresses.some(a => a.id === accountId)) {
                                            searchResults.addresses.push({
                                                type: 'address' as const,
                                                id: accountId,
                                                title: 'Account',
                                                subtitle: `Balance: ${(result.account.amount / 1000000).toLocaleString()} CNPY`,
                                                data: result.account
                                            })
                                        }
                                    }
                                }
                            })
                            .catch(err => console.log('Address search error:', err))
                    )
                } else {
                    // For partial addresses (8-39 chars), search in loaded validators first
                    const termLower = term.toLowerCase()

                    // Search in validators list if available (validators take priority)
                    const foundValidatorAddresses = new Set<string>()

                    if (allValidatorsData?.results) {
                        const matchingValidators = allValidatorsData.results.filter((v: any) => {
                            const address = (v.address || '').toLowerCase()
                            return address.startsWith(termLower)
                        })

                        matchingValidators.forEach((validator: any) => {
                            if (validator.address) {
                                foundValidatorAddresses.add(validator.address.toLowerCase())
                                const validatorResult = {
                                    type: 'validator' as const,
                                    id: validator.address,
                                    title: validator.name || validator.netAddress || 'Validator',
                                    subtitle: `Address: ${validator.address.slice(0, 16)}...`,
                                    data: validator
                                }
                                if (!searchResults.validators.some(v => v.id === validator.address)) {
                                    searchResults.validators.push(validatorResult)
                                }
                            }
                        })
                    }

                    // Also try API calls for exact matches (in case the validator list doesn't have it)
                    // Try with padded address (might work for some cases)
                    const paddedAddress = term.padEnd(40, '0')

                    searchPromises.push(
                        getModalData(paddedAddress, 1)
                            .then(result => {
                                if (result && result !== "no result found") {
                                    // Si es un validador, agregarlo como validador
                                    if (result.validator && result.validator.address && result.validator.address.toLowerCase().startsWith(termLower)) {
                                        const validatorId = result.validator.address
                                        foundValidatorAddresses.add(validatorId.toLowerCase())
                                        if (!searchResults.validators.some(v => v.id === validatorId)) {
                                            searchResults.validators.push({
                                                type: 'validator' as const,
                                                id: validatorId,
                                                title: result.validator.name || 'Validator',
                                                subtitle: `Address: ${validatorId.slice(0, 16)}...`,
                                                data: result.validator
                                            })
                                        }
                                    }
                                    
                                    // También agregar como cuenta si existe (incluso si es validador)
                                    if (result.account && result.account.address && result.account.address.toLowerCase().startsWith(termLower)) {
                                        const accountId = result.account.address
                                        // Agregar como address incluso si es validador (ambos pueden existir)
                                        if (!searchResults.addresses.some(a => a.id === accountId)) {
                                            searchResults.addresses.push({
                                                type: 'address' as const,
                                                id: accountId,
                                                title: 'Account',
                                                subtitle: `Balance: ${(result.account.amount / 1000000).toLocaleString()} CNPY`,
                                                data: result.account
                                            })
                                        }
                                    }
                                }
                            })
                            .catch(() => { }) // Silently fail
                    )

                    // Try direct API calls (these might fail for partial addresses, but worth trying)
                    searchPromises.push(
                        Validator(0, term)
                            .then(validator => {
                                if (validator && validator.address && validator.address.toLowerCase().startsWith(termLower)) {
                                    foundValidatorAddresses.add(validator.address.toLowerCase())
                                    if (!searchResults.validators.some(v => v.id === validator.address)) {
                                        searchResults.validators.push({
                                            type: 'validator' as const,
                                            id: validator.address,
                                            title: validator.name || 'Validator',
                                            subtitle: `Address: ${validator.address.slice(0, 16)}...`,
                                            data: validator
                                        })
                                    }
                                }
                            })
                            .catch(() => { }) // Silently fail for partial searches
                    )

                    searchPromises.push(
                        Account(0, term)
                            .then(account => {
                                if (account && account.address && account.address.toLowerCase().startsWith(termLower)) {
                                    const accountId = account.address
                                    // Agregar como address incluso si es validador (ambos pueden existir)
                                    if (!searchResults.addresses.some(a => a.id === accountId)) {
                                        searchResults.addresses.push({
                                            type: 'address' as const,
                                            id: accountId,
                                            title: 'Account',
                                            subtitle: `Balance: ${(account.amount / 1000000).toLocaleString()} CNPY`,
                                            data: account
                                        })
                                    }
                                }
                            })
                            .catch(() => { }) // Silently fail for partial searches
                    )
                }
            }

            // 3. If it is a number (block height)
            if (/^\d+$/.test(term)) {
                const blockHeight = parseInt(term)
                searchPromises.push(
                    BlockByHeight(blockHeight)
                        .then(block => {
                            if (block && block.blockHeader) {
                                searchResults.blocks.push({
                                    type: 'block' as const,
                                    id: block.blockHeader.hash || '',
                                    title: `Block #${block.blockHeader.height}`,
                                    subtitle: `Hash: ${(block.blockHeader.hash || '').slice(0, 16)}...`,
                                    data: block
                                })
                            }
                        })
                        .catch(err => console.log('Block height search error:', err))
                )
            }

            // Esperar a que todas las promesas se completen
            await Promise.all(searchPromises)

            // Calcular total
            const total = searchResults.blocks.length +
                searchResults.transactions.length +
                searchResults.addresses.length +
                searchResults.validators.length

            setResults({
                ...searchResults,
                total
            })
        } catch (err) {
            setError('Error searching data')
            console.error('Search error:', err)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        const timeoutId = setTimeout(() => {
            searchInData(searchTerm)
        }, 300) // 300ms debounce

        return () => clearTimeout(timeoutId)
    }, [searchTerm, hashSearchData, isHashSearch, allValidatorsData])

    return {
        results,
        loading,
        error,
        search: searchInData
    }
}