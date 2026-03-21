import React from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './app/App'
import './index.css'
import "@radix-ui/themes/styles.css";


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
    </QueryClientProvider>
  </React.StrictMode>
)
