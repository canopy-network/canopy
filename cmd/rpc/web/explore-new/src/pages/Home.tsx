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
      className='mx-auto px-4 sm:px-6 lg:px-8 py-10 flex flex-col gap-8'
    >
      <Stages />
      <OverviewCards />
      <ExtraTables />
    </motion.div>
  )
}

export default HomePage