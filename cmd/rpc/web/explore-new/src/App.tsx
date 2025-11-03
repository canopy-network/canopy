import { BrowserRouter as Router, Routes, Route, useLocation } from 'react-router-dom'
import { AnimatePresence } from 'framer-motion'
import { Toaster } from 'react-hot-toast'
import Navbar from './components/Navbar'
import Footer from './components/Footer'
import HomePage from './pages/Home'
import SearchPage from './pages/Search'
import BlocksPage from './components/block/BlocksPage'
import BlockDetailPage from './components/block/BlockDetailPage'
import TransactionsPage from './components/transaction/TransactionsPage'
import TransactionDetailPage from './components/transaction/TransactionDetailPage'
import ValidatorsPage from './components/validator/ValidatorsPage'
import ValidatorDetailPage from './components/validator/ValidatorDetailPage'
import AccountsPage from './components/account/AccountsPage'
import AccountDetailPage from './components/account/AccountDetailPage'
import NetworkAnalyticsPage from './components/analytics/NetworkAnalyticsPage'
import TokenSwapsPage from './components/token-swaps/TokenSwapsPage'
import StakingPage from './components/staking/StakingPage'
import GovernancePage from './components/staking/GovernancePage'
import SupplyPage from './components/staking/SupplyPage'
import { useNetworkChangeHandler } from './hooks/useApi'


function AnimatedRoutes() {
  const location = useLocation()
  return (
    <AnimatePresence mode="popLayout" initial={false}>
      <Routes location={location} key={location.pathname}>
        <Route index element={<HomePage />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/blocks" element={<BlocksPage />} />
        <Route path="/block/:blockHeight" element={<BlockDetailPage />} />
        <Route path="/transactions" element={<TransactionsPage />} />
        <Route path="/transaction/:transactionHash" element={<TransactionDetailPage />} />
        <Route path="/validators" element={<ValidatorsPage />} />
        <Route path="/delegators" element={<ValidatorsPage />} />
        <Route path="/validator/:validatorAddress" element={<ValidatorDetailPage />} />
        <Route path="/accounts" element={<AccountsPage />} />
        <Route path="/account/:address" element={<AccountDetailPage />} />
        <Route path="/analytics" element={<NetworkAnalyticsPage />} />
        <Route path="/token-swaps" element={<TokenSwapsPage />} />
        <Route path="/staking" element={<StakingPage />} />
        <Route path="/staking/governance" element={<GovernancePage />} />
        <Route path="/staking/supply" element={<SupplyPage />} />
        <Route path="/governance" element={<HomePage />} />
        <Route path="/orders" element={<HomePage />} />
      </Routes>
    </AnimatePresence>
  )
}

function App() {
  // Handle network changes and invalidate queries
  useNetworkChangeHandler();

  return (
    <Router>
      <div className="min-h-screen bg-background flex flex-col">
        <Navbar />
        <main className="flex-1">
          <AnimatedRoutes />
        </main>
        <Footer />
        <Toaster
          position="top-right"
          toastOptions={{
            duration: 3000,
            style: {
              background: '#1f2937',
              color: '#f9fafb',
              border: '1px solid #374151',
              borderRadius: '12px',
              padding: '12px 16px',
              fontSize: '14px',
              fontWeight: '500',
            },
            success: {
              iconTheme: {
                primary: '#4ade80',
                secondary: '#1f2937',
              },
              style: {
                background: '#1f2937',
                color: '#f9fafb',
                border: '1px solid #4ade80',
              },
            },
            error: {
              iconTheme: {
                primary: '#ef4444',
                secondary: '#1f2937',
              },
              style: {
                background: '#1f2937',
                color: '#f9fafb',
                border: '1px solid #ef4444',
              },
            },
          }}
        />
      </div>
    </Router>
  )
}

export default App
