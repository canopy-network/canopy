import Stages from '../components/Home/Stages'
import OverviewCards from '../components/Home/OverviewCards'
import ExtraTables from '../components/Home/ExtraTables'

const HomePage = () => {
  return (
    <div className='mx-auto px-4 sm:px-6 lg:px-8 py-10 flex flex-col gap-8'>
      <Stages />
      <OverviewCards />
      <ExtraTables />
    </div>
  )
}

export default HomePage