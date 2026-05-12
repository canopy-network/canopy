import { motion } from 'framer-motion'
import Stages from '../components/Home/Stages'
import OverviewCards from '../components/Home/OverviewCards'
import ExtraTables from '../components/Home/ExtraTables'

const HomePage = () => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.3, ease: "easeInOut" }}
      className='w-full flex flex-col gap-8'
    >
      <div className="flex flex-col gap-2">
        <h1 className="explorer-page-title">
          Explorer
        </h1>
        <p className="explorer-page-subtitle max-w-2xl">
          Live network overview, blocks, transactions, and validators.
        </p>
      </div>
      <Stages />
      <OverviewCards />
      <ExtraTables />
    </motion.div>
  )
}

export default HomePage
