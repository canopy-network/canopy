import { useState, useEffect } from 'react'
import { useTxByHash } from './useApi'
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

    const searchInData = async (term: string) => {
        if (!term.trim()) {
            setResults(null)
            return
        }

        setLoading(true)
        setError(null)

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

            // 1. If it looks like a transaction hash (32+ hexadecimal characters)
            if (term.length >= 32 && /^[a-fA-F0-9]+$/.test(term)) {
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

                // It could also be a block hash
                searchPromises.push(
                    BlockByHash(term)
                        .then(block => {
                            if (block && block.blockHeader) {
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

            // 2. If it is an address (40 hexadecimal characters)
            if (term.length === 40) {
                searchPromises.push(
                    getModalData(term, 1)
                        .then(result => {
                            if (result && result !== "no result found") {
                                // Si es una cuenta
                                if (result.account) {
                                    searchResults.addresses.push({
                                        type: 'address' as const,
                                        id: result.account.address,
                                        title: 'Account',
                                        subtitle: `Balance: ${(result.account.amount / 1000000).toLocaleString()} CNPY`,
                                        data: result.account
                                    })
                                }

                                // Si es un validador
                                if (result.validator) {
                                    searchResults.validators.push({
                                        type: 'validator' as const,
                                        id: result.validator.address,
                                        title: result.validator.name || 'Validator',
                                        subtitle: `Address: ${result.validator.address.slice(0, 16)}...`,
                                        data: result.validator
                                    })
                                }
                            }
                        })
                        .catch(err => console.log('Address search error:', err))
                )

                // Direct search as validator and as account
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
                        })
                        .catch(err => console.log('Validator search error:', err))
                )

                searchPromises.push(
                    Account(0, term)
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
                )
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
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [searchTerm, hashSearchData, isHashSearch])

    return {
        results,
        loading,
        error,
        search: searchInData
    }
}