import React from 'react'
import {RouterProvider} from 'react-router-dom'
import {ConfigProvider} from './providers/ConfigProvider'
import router from "./routes";
import {AccountsProvider} from "@/app/providers/AccountsProvider";
import {ToastProvider} from "@/toast/ToastContext";
import {ActionModalProvider} from "@/app/providers/ActionModalProvider";
import {Theme} from "@radix-ui/themes";

export default function App() {
    return (
        <ConfigProvider>
            <AccountsProvider>
                <ToastProvider position="top-right" defaultDurationMs={3000}>
                    <ActionModalProvider>
                        <Theme>
                            <RouterProvider router={router}/>
                        </Theme>
                    </ActionModalProvider>
                </ToastProvider>
            </AccountsProvider>
        </ConfigProvider>
    )
}
