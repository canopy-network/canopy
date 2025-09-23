import { useState, useEffect } from 'react'
import { useBlocks, useTransactions, useValidators } from './useApi'
import { getModalData } from '../lib/api'

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

    const { data: blocksData } = useBlocks(1)
    const { data: transactionsData } = useTransactions(1, 0)
    const { data: validatorsData } = useValidators(1)

    const searchInData = async (term: string) => {
        if (!term.trim()) {
            setResults(null)
            return
        }

        setLoading(true)
        setError(null)

        try {
            const lowerTerm = term.toLowerCase()
            const searchResults: SearchResults = {
                total: 0,
                blocks: [],
                transactions: [],
                addresses: [],
                validators: []
            }

            // First try direct API search for specific addresses
            if (term.length === 40) { // 40-character address
                try {
                    const apiResult = await getModalData(term, 1)
                    if (apiResult && apiResult !== "no result found") {
                        // If it's an account
                        if (apiResult.account) {
                            searchResults.addresses.push({
                                type: 'address' as const,
                                id: apiResult.account.address,
                                title: 'Account',
                                subtitle: `Balance: ${(apiResult.account.amount / 1000000).toLocaleString()} CNPY`,
                                data: apiResult.account
                            })
                        }
                        // If it's a validator
                        if (apiResult.validator) {
                            searchResults.validators.push({
                                type: 'validator' as const,
                                id: apiResult.validator.address,
                                title: apiResult.validator.name || 'Validator',
                                subtitle: `Address: ${apiResult.validator.address.slice(0, 16)}...`,
                                data: apiResult.validator
                            })
                        }
                        // If it's a block
                        if (apiResult.block) {
                            searchResults.blocks.push({
                                type: 'block' as const,
                                id: apiResult.block.blockHeader?.hash || apiResult.block.hash || '',
                                title: `Block #${apiResult.block.blockHeader?.height ?? apiResult.block.height}`,
                                subtitle: `Hash: ${(apiResult.block.blockHeader?.hash || apiResult.block.hash || '').slice(0, 16)}...`,
                                data: apiResult.block
                            })
                        }
                        // If it's a transaction
                        if (apiResult.sender || apiResult.txHash) {
                            searchResults.transactions.push({
                                type: 'transaction' as const,
                                id: apiResult.txHash || apiResult.hash || '',
                                title: 'Transaction',
                                subtitle: `Hash: ${(apiResult.txHash || apiResult.hash || '').slice(0, 16)}...`,
                                data: apiResult
                            })
                        }
                    }
                } catch (apiError) {
                    console.log('API search failed, falling back to local search:', apiError)
                }
            }

            // Local search in loaded data (as fallback)
            // Search in blocks
            if (blocksData?.results) {
                const blocks = blocksData.results.filter((block: any) => {
                    const height = block.blockHeader?.height ?? block.height
                    const hash = block.blockHeader?.hash || block.hash || ''
                    return (
                        height?.toString().includes(term) ||
                        hash.toLowerCase().includes(lowerTerm)
                    )
                })

                const newBlocks = blocks.slice(0, 5).map((block: any) => ({
                    type: 'block' as const,
                    id: block.blockHeader?.hash || block.hash || '',
                    title: `Block #${block.blockHeader?.height ?? block.height}`,
                    subtitle: `Hash: ${(block.blockHeader?.hash || block.hash || '').slice(0, 16)}...`,
                    data: block
                }))

                // Avoid duplicates
                newBlocks.forEach((block: any) => {
                    if (!searchResults.blocks.some(b => b.id === block.id)) {
                        searchResults.blocks.push(block)
                    }
                })
            }

            // Search in transactions
            if (transactionsData?.results) {
                const transactions = transactionsData.results.filter((tx: any) => {
                    const hash = tx.txHash || tx.hash || ''
                    const from = tx.sender || tx.from || ''
                    const to = tx.recipient || tx.to || ''
                    return (
                        hash.toLowerCase().includes(lowerTerm) ||
                        from.toLowerCase().includes(lowerTerm) ||
                        to.toLowerCase().includes(lowerTerm)
                    )
                })

                const newTransactions = transactions.slice(0, 5).map((tx: any) => ({
                    type: 'transaction' as const,
                    id: tx.txHash || tx.hash || '',
                    title: 'Transaction',
                    subtitle: `Hash: ${(tx.txHash || tx.hash || '').slice(0, 16)}...`,
                    data: tx
                }))

                // Avoid duplicates
                newTransactions.forEach((tx: any) => {
                    if (!searchResults.transactions.some(t => t.id === tx.id)) {
                        searchResults.transactions.push(tx)
                    }
                })
            }

            // Search in validators
            if (validatorsData?.results) {
                const validators = validatorsData.results.filter((validator: any) => {
                    const address = validator.address || ''
                    const name = validator.name || ''
                    return (
                        address.toLowerCase().includes(lowerTerm) ||
                        name.toLowerCase().includes(lowerTerm)
                    )
                })

                const newValidators = validators.slice(0, 5).map((validator: any) => ({
                    type: 'validator' as const,
                    id: validator.address || '',
                    title: validator.name || 'Validator',
                    subtitle: `Address: ${(validator.address || '').slice(0, 16)}...`,
                    data: validator
                }))

                // Avoid duplicates
                newValidators.forEach((validator: any) => {
                    if (!searchResults.validators.some(v => v.id === validator.id)) {
                        searchResults.validators.push(validator)
                    }
                })
            }

            // Search addresses in transactions (sender/recipient) only if we didn't find anything with the API
            if (searchResults.addresses.length === 0 && transactionsData?.results) {
                const addressSet = new Set<string>()
                
                transactionsData.results.forEach((tx: any) => {
                    const from = tx.sender || tx.from || ''
                    const to = tx.recipient || tx.to || ''
                    
                    if (from && from.toLowerCase().includes(lowerTerm)) {
                        addressSet.add(from)
                    }
                    if (to && to.toLowerCase().includes(lowerTerm)) {
                        addressSet.add(to)
                    }
                })
                
                // Convert Set to Array and create address results
                const addresses = Array.from(addressSet).slice(0, 5)
                searchResults.addresses = addresses.map((address: string) => ({
                    type: 'address' as const,
                    id: address,
                    title: 'Address',
                    subtitle: `Address: ${address.slice(0, 16)}...`,
                    data: {
                        address: address,
                        balance: 0, // This should come from a real API
                        transactionCount: 0 // This should come from a real API
                    }
                }))
            }

            // Remove duplicates and prioritize results by type
            const deduplicatedResults = {
                ...searchResults,
                addresses: searchResults.addresses.filter((addr: any) => {
                    // Don't show addresses that already appear as transactions
                    const isInTransactions = searchResults.transactions.some((tx: any) => 
                        tx.data?.sender === addr.id || tx.data?.from === addr.id ||
                        tx.data?.recipient === addr.id || tx.data?.to === addr.id
                    )
                    return !isInTransactions
                })
            }

            deduplicatedResults.total = 
                deduplicatedResults.blocks.length +
                deduplicatedResults.transactions.length +
                deduplicatedResults.addresses.length +
                deduplicatedResults.validators.length

            setResults(deduplicatedResults)
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
    }, [searchTerm, blocksData, transactionsData, validatorsData])

    return {
        results,
        loading,
        error,
        search: searchInData
    }
}
