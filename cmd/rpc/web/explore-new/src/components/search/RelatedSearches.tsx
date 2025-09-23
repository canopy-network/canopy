import React from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'

const RelatedSearches: React.FC = () => {
    const relatedSearches = [
        {
            title: 'Recent Blocks',
            description: 'Explore the latest blocks on the network',
            icon: 'fa-solid fa-cube',
            link: '/blocks',
            color: 'text-primary bg-green-600/20 py-2.5 pr-7 pl-2.5 rounded-full'
        },
        {
            title: 'Latest Transactions',
            description: 'View recent transaction activity',
            icon: 'fa-solid fa-arrow-right-arrow-left',
            link: '/transactions',
            color: 'text-blue-500 bg-blue-600/20 py-2.5 pr-7 pl-2.5 rounded-full'
        },
        {
            title: 'Top Validators',
            description: 'See the most active validators',
            icon: 'fa-solid fa-chart-pie',
            link: '/validators',
            color: 'text-primary bg-green-600/20 py-2.5 pr-7.5 pl-[0.610rem] rounded-full'
        }
    ]

    return (
        <div>
            <h2 className="text-xl font-semibold text-white mb-6">Related Searches</h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {relatedSearches.map((search, index) => (
                    <motion.div
                        key={search.title}
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: index * 0.1 }}
                        className="bg-input border border-gray-800/60 rounded-xl p-6  hover:border-gray-800/80 transition-colors group"
                    >
                        <Link to={search.link} className="block">
                            <div className="flex items-start gap-4">
                                <div className={`w-12 h-12 bg-input rounded-lg flex items-center justify-center group-hover:bg-gray-500 transition-colors`}>
                                    <i className={`${search.icon} ${search.color} text-lg`}></i>
                                </div>
                                <div className="flex-1">
                                    <h3 className="text-white font-medium mb-2 group-hover:text-primary transition-colors">
                                        {search.title}
                                    </h3>
                                    <p className="text-gray-400 text-sm">
                                        {search.description}
                                    </p>
                                </div>
                            </div>
                        </Link>
                    </motion.div>
                ))}
            </div>
        </div>
    )
}

export default RelatedSearches
