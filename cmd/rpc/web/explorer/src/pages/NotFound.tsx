import React from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'

const NotFoundPage: React.FC = () => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.3, ease: 'easeInOut' }}
      className="mx-auto px-4 sm:px-6 lg:px-8 py-16 max-w-[60rem]"
    >
      <div className="rounded-xl border border-gray-800/60 bg-card shadow-xl p-10 text-center">
        <p className="text-primary text-sm font-semibold tracking-widest mb-2">ERROR 404</p>
        <h1 className="text-4xl sm:text-5xl font-bold text-white mb-4">Page not found</h1>
        <p className="text-gray-400 mb-8">
          The page you requested does not exist or was moved.
        </p>
        <Link
          to="/"
          className="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-primary/20 hover:bg-primary/30 text-primary font-medium transition-colors duration-200"
        >
          <i className="fa-solid fa-house"></i>
          Go to dashboard
        </Link>
      </div>
    </motion.div>
  )
}

export default NotFoundPage
