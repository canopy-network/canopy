import React from 'react'
import { RouterProvider } from 'react-router-dom'
import { ConfigProvider } from './providers/ConfigProvider'
import router from "./routes";
import {AccountsProvider} from "@/app/providers/AccountsProvider";
import {ToastProvider} from "@/toast/ToastContext";
import {Theme} from "@radix-ui/themes";

export default function App() {
  return (
      <ConfigProvider>
          <ToastProvider>
              <Theme>
                  <AccountsProvider>
                      <RouterProvider router={router}/>
                  </AccountsProvider>
              </Theme>
          </ToastProvider>
      </ConfigProvider>
  )
}
