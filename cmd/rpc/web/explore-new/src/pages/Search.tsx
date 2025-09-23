import React, { useState, useEffect } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import SearchFilters from '../components/search/SearchFilters'
import { useSearch } from '../hooks/useSearch'
import SearchResults from '../components/search/SearchResults'
import RelatedSearches from '../components/search/RelatedSearches'

const SearchPage: React.FC = () => {
    const [searchParams, setSearchParams] = useSearchParams()
    const navigate = useNavigate()
    const [searchTerm, setSearchTerm] = useState('')
    const [filters, setFilters] = useState({
        type: 'all',
        date: 'all',
        sort: 'newest'
    })

    const { results: searchResults, loading } = useSearch(searchTerm)

    // Get search term from URL
    useEffect(() => {
        const query = searchParams.get('q')
        if (query) {
            setSearchTerm(query)
        }
    }, [searchParams])

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault()
        if (searchTerm.trim()) {
            setSearchParams({ q: searchTerm.trim() })
            navigate(`/search?q=${encodeURIComponent(searchTerm.trim())}`)
        }
    }

    const handleFilterChange = (newFilters: any) => {
        setFilters(newFilters)
        // Here you could refilter the results
    }

    const clearSearch = () => {
        setSearchTerm('')
        setSearchParams({})
        navigate('/search')
    }

    return (
        <div className="min-h-screen bg-background">
            <div className="container mx-auto p-8 md:my-8 bg-card md:rounded-xl">
                {/* Header */}
                <motion.div
                    initial={{ opacity: 0, y: -20 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="mb-8"
                >
                    <h1 className="text-3xl font-bold text-white mb-2">Search Results</h1>
                    <p className="text-gray-400">Find blocks, transactions, addresses, and validators</p>
                </motion.div>

                {/* Search Input */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.1 }}
                    className="mb-6"
                >
                    <form onSubmit={handleSearch} className="relative">
                        <input
                            type="text"
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                            placeholder="Search blocks, transactions, addresses..."
                            className="w-full bg-input border border-gray-800/50 rounded-lg px-4 py-3 pl-12 pr-3 text-white placeholder-gray-500 focus:outline-none focus:ring focus:ring-primary/50 focus:border-primary"
                        />
                        {searchTerm && (
                            <button
                                type="button"
                                onClick={clearSearch}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
                            >
                                <i className="fa-solid fa-times"></i>
                            </button>
                        )}
                        <button
                            type="submit"
                            className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-primary transition-colors"
                        >
                            <i className="fa-solid fa-magnifying-glass"></i>
                        </button>
                    </form>
                </motion.div>

                {/* Filters */}
                {searchTerm && (
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: 0.2 }}
                        className="mb-6"
                    >
                        <SearchFilters filters={filters} onFilterChange={handleFilterChange} />
                    </motion.div>
                )}

                {/* Results */}
                <AnimatePresence mode="wait">
                    {loading ? (
                        <motion.div
                            key="loading"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            className="flex items-center justify-center py-12"
                        >
                            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                            <span className="ml-3 text-gray-400">Searching...</span>
                        </motion.div>
                    ) : searchResults ? (
                        <motion.div
                            key="results"
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            exit={{ opacity: 0, y: -20 }}
                            transition={{ delay: 0.3 }}
                        >
                            <SearchResults
                                results={searchResults}
                                searchTerm={searchTerm}
                                filters={filters}
                            />
                        </motion.div>
                    ) : searchTerm ? (
                        <motion.div
                            key="no-results"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            className="text-center py-12"
                        >
                            <i className="fa-solid fa-search text-4xl text-gray-600 mb-4"></i>
                            <h3 className="text-xl font-semibold text-white mb-2">No results found</h3>
                            <p className="text-gray-400">Try searching for a different term</p>
                        </motion.div>
                    ) : (
                        <motion.div
                            key="empty"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            className="text-center py-12"
                        >
                            <i className="fa-solid fa-search text-4xl text-gray-600 mb-4"></i>
                            <h3 className="text-xl font-semibold text-white mb-2">Start searching</h3>
                            <p className="text-gray-400">Enter a block height, transaction hash, address, or validator name</p>
                        </motion.div>
                    )}
                </AnimatePresence>


            </div>
            <AnimatePresence mode="wait">
                {/* Related Searches */}
                {searchTerm && (
                    <div
                        className="container mx-auto p-8 py-4 mb-4 bg-card md:rounded-xl">
                        <motion.div
                            initial={{ opacity: 0, y: 20 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.4 }}
                        >
                            <RelatedSearches />
                        </motion.div>

                    </div>
                )}
            </AnimatePresence>
        </div>
    )
}

export default SearchPage
