import React from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'react-hot-toast'
import App from './app/App'
import './index.css'

const qc = new QueryClient({
  defaultOptions: {
    queries: {
      refetchInterval: 20000, // 20 seconds
      refetchIntervalInBackground: true, // Continue to refetch in background
      staleTime: 10000, // Data is considered stale after 10 seconds
      refetchOnWindowFocus: true, // Update when the window regains focus
    },
  },
})
createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={qc}>
      <App />
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          style: {
            background: '#2a2b35',
            color: '#ffffff',
            border: '1px solid #3a3b45',
            borderRadius: '0.5rem',
            padding: '12px 16px',
            fontSize: '14px',
            fontWeight: '500',
          },
          success: {
            iconTheme: {
              primary: '#6fe3b4',
              secondary: '#1a1b23',
            },
            style: {
              background: '#1a2b23',
              border: '1px solid #6fe3b4',
            },
          },
          error: {
            iconTheme: {
              primary: '#ef4444',
              secondary: '#1a1b23',
            },
            style: {
              background: '#2b1a1a',
              border: '1px solid #ef4444',
            },
          },
          loading: {
            iconTheme: {
              primary: '#3b82f6',
              secondary: '#1a1b23',
            },
            style: {
              background: '#1a1f2b',
              border: '1px solid #3b82f6',
            },
          },
        }}
      />
    </QueryClientProvider>
  </React.StrictMode>
)
