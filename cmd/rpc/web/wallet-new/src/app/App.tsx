import React from 'react'
import { RouterProvider } from 'react-router-dom'
import { ConfigProvider } from './providers/ConfigProvider'
import ActionRunner from '../actions/ActionRunner'
import router from "./routes";
import {AccountsProvider} from "@/app/providers/AccountsProvider";

export default function App() {
  const params = new URLSearchParams(location.search)
  const chainId = params.get('chain') ?? undefined
  const actionId = params.get('action') ?? 'Send'

  return (

    <ConfigProvider chainId={chainId}>
        <AccountsProvider>
            <RouterProvider router={router} />
        </AccountsProvider>
    </ConfigProvider>
  )
}
