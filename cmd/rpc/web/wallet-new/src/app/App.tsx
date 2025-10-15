import React from 'react'
import { RouterProvider } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import { ConfigProvider } from './providers/ConfigProvider'
import ActionRunner from '../actions/ActionRunner'
import router from "./routes";

export default function App() {
  const params = new URLSearchParams(location.search)
  const chainId = params.get('chain') ?? undefined
  const actionId = params.get('action') ?? 'Send'

  return (
    <ConfigProvider chainId={chainId}>
        <RouterProvider router={router} />
        <Toaster 
          position="top-right"
          toastOptions={{
            duration: 4000,
            style: {
              fontSize: '14px',
            },
            success: {
              duration: 3000,
            },
            error: {
              duration: 5000,
            },
          }}
        />
    </ConfigProvider>
  )
}
