import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import HomePage from './pages/Home'
import SearchPage from './pages/Search'
import NotFoundPage from './pages/NotFound'
import BlocksPage from './components/block/BlocksPage'
import BlockDetailPage from './components/block/BlockDetailPage'
import TransactionsPage from './components/transaction/TransactionsPage'
import TransactionDetailPage from './components/transaction/TransactionDetailPage'
import ValidatorDetailPage from './components/validator/ValidatorDetailPage'
import AccountsPage from './components/account/AccountsPage'
import AccountDetailPage from './components/account/AccountDetailPage'
import TokenSwapsPage from './components/token-swaps/TokenSwapsPage'
import OrderDetailPage from './components/token-swaps/OrderDetailPage'
import DexBatchesPage from './components/dex/DexBatchesPage'
import StakingPage from './components/staking/StakingPage'
import GovernancePage from './components/staking/GovernancePage'
import SupplyPage from './components/staking/SupplyPage'
import { useNetworkChangeHandler } from './hooks/useApi'
import ExplorerLayout from './components/layouts/ExplorerLayout'

function App() {
  // Handle network changes and invalidate queries
  useNetworkChangeHandler();

  return (
    <Router>
      <div className="min-h-screen bg-background">
        <Routes>
          <Route element={<ExplorerLayout />}>
            <Route index element={<HomePage />} />
            <Route path="/summary" element={<Navigate to="/" replace />} />
            <Route path="/search" element={<SearchPage />} />
            <Route path="/blocks" element={<BlocksPage />} />
            <Route path="/block/:blockHeight" element={<BlockDetailPage />} />
            <Route path="/transactions" element={<TransactionsPage />} />
            <Route path="/transaction/:transactionHash" element={<TransactionDetailPage />} />
            <Route path="/validators" element={<Navigate to="/staking" replace />} />
            <Route path="/delegators" element={<Navigate to="/staking" replace />} />
            <Route path="/validator/:validatorAddress" element={<ValidatorDetailPage />} />
            <Route path="/accounts" element={<AccountsPage />} />
            <Route path="/account/:address" element={<AccountDetailPage />} />
            <Route path="/token-swaps" element={<TokenSwapsPage />} />
            <Route path="/order/:committee/:orderId" element={<OrderDetailPage />} />
            <Route path="/dex" element={<DexBatchesPage />} />
            <Route path="/staking" element={<StakingPage />} />
            <Route path="/governance" element={<GovernancePage />} />
            <Route path="/supply" element={<SupplyPage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Route>
        </Routes>
        <Toaster
          position="top-right"
          toastOptions={{
            duration: 3000,
            style: {
              background: '#1a1a1a',
              color: '#fafafa',
              border: '1px solid rgba(255,255,255,0.10)',
              borderRadius: '12px',
              padding: '12px 16px',
              fontSize: '14px',
              fontWeight: '500',
            },
            success: {
              iconTheme: {
                primary: '#45ca46',
                secondary: '#1a1a1a',
              },
              style: {
                background: '#1a1a1a',
                color: '#fafafa',
                border: '1px solid #45ca46',
              },
            },
            error: {
              iconTheme: {
                primary: '#ef4444',
                secondary: '#1a1a1a',
              },
              style: {
                background: '#1a1a1a',
                color: '#fafafa',
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
