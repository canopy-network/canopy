import React, { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Shield, Eye, EyeOff, X, Unlock, AlertCircle } from 'lucide-react'

interface UnlockModalProps {
    open: boolean
    onClose: () => void
    onUnlock: (password: string) => void
}

export default function UnlockModal({ open, onClose, onUnlock }: UnlockModalProps) {
    const [pwd, setPwd] = useState('')
    const [err, setErr] = useState<string>('')
    const [showPassword, setShowPassword] = useState(false)
    const [isSubmitting, setIsSubmitting] = useState(false)
    const inputRef = useRef<HTMLInputElement>(null)

    // Focus input when modal opens
    useEffect(() => {
        if (open && inputRef.current) {
            setTimeout(() => inputRef.current?.focus(), 100)
        }
        // Reset state when modal opens
        if (open) {
            setPwd('')
            setErr('')
            setShowPassword(false)
            setIsSubmitting(false)
        }
    }, [open])

    // Handle Enter key
    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && pwd) {
            submit()
        } else if (e.key === 'Escape') {
            onClose()
        }
    }

    const submit = async () => {
        if (!pwd) {
            setErr('Password is required')
            inputRef.current?.focus()
            return
        }

        setIsSubmitting(true)
        setErr('')

        // Simulate brief delay for UX
        await new Promise(resolve => setTimeout(resolve, 200))

        // Success path is handled by onUnlock callback.
        // onClose should represent cancel/dismiss only.
        onUnlock(pwd)
    }

    return (
        <AnimatePresence>
            {open && (
                <motion.div
                    className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.2 }}
                >
                    {/* Backdrop */}
                    <motion.div
                        className="absolute inset-0 bg-[#0f0f0f]/80 backdrop-blur-md"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        onClick={onClose}
                    />

                    {/* Modal */}
                    <motion.div
                        className="relative w-full max-w-md max-h-[calc(100dvh-1.5rem)] sm:max-h-[calc(100dvh-2rem)] bg-[#171717] border border-[#272729] rounded-2xl shadow-[0_24px_72px_rgba(0,0,0,0.55)] overflow-hidden flex flex-col"
                        initial={{ opacity: 0, scale: 0.95, y: 20 }}
                        animate={{ opacity: 1, scale: 1, y: 0 }}
                        exit={{ opacity: 0, scale: 0.95, y: 20 }}
                        transition={{ duration: 0.2, ease: 'easeOut' }}
                    >
                        {/* Close button */}
                        <button
                            onClick={onClose}
                            className="absolute top-4 right-4 rounded-lg border border-[#272729] bg-[#0f0f0f] p-1.5 text-white/60 hover:bg-[#272729] hover:text-white transition-colors"
                        >
                            <X className="w-5 h-5" />
                        </button>

                        <div className="p-4 pt-7 sm:p-6 sm:pt-8 overflow-y-auto min-h-0">
                            {/* Icon */}
                            <div className="flex justify-center mb-5">
                                <div className="relative flex h-16 w-16 items-center justify-center rounded-full border border-[#272729] bg-[#0f0f0f]">
                                    <Shield className="h-8 w-8 text-[#35cd48]" />
                                </div>
                            </div>

                            {/* Title */}
                            <h2 className="text-xl font-semibold text-foreground text-center mb-2">
                                Unlock Wallet
                            </h2>

                            {/* Description */}
                            <p className="text-sm text-muted-foreground text-center mb-6">
                                Enter your password to authorize transactions
                            </p>

                            {/* Password input */}
                            <div className="space-y-2">
                                <label className="block text-sm font-medium text-foreground/80">
                                    Password
                                </label>
                                <div className="relative">
                                    <input
                                        ref={inputRef}
                                        type={showPassword ? 'text' : 'password'}
                                        value={pwd}
                                        onChange={e => {
                                            setPwd(e.target.value)
                                            if (err) setErr('')
                                        }}
                                        onKeyDown={handleKeyDown}
                                        placeholder="Enter your wallet password"
                                        className={`
                                            w-full bg-[#0f0f0f] text-foreground rounded-xl px-4 py-3 pr-12
                                            border transition-all duration-200 outline-none
                                            placeholder:text-muted-foreground
                                            ${err
                                                ? 'border-[#ff1845]/50 focus:border-[#ff1845] focus:ring-2 focus:ring-[#ff1845]/20'
                                                : 'border-[#272729] focus:border-[#35cd48]/50 focus:ring-2 focus:ring-[#35cd48]/20'
                                            }
                                        `}
                                        disabled={isSubmitting}
                                    />
                                    <button
                                        type="button"
                                        onClick={() => setShowPassword(!showPassword)}
                                        className="absolute right-3 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground transition-colors"
                                        tabIndex={-1}
                                    >
                                        {showPassword ? (
                                            <EyeOff className="w-5 h-5" />
                                        ) : (
                                            <Eye className="w-5 h-5" />
                                        )}
                                    </button>
                                </div>

                                {/* Error message */}
                                <AnimatePresence>
                                    {err && (
                                        <motion.div
                                            initial={{ opacity: 0, y: -10 }}
                                            animate={{ opacity: 1, y: 0 }}
                                            exit={{ opacity: 0, y: -10 }}
                                            className="flex items-center gap-2 text-[#ff1845] text-sm"
                                        >
                                            <AlertCircle className="w-4 h-4" />
                                            {err}
                                        </motion.div>
                                    )}
                                </AnimatePresence>
                            </div>

                            {/* Actions */}
                            <div className="flex gap-3 mt-6">
                                <button
                                    onClick={onClose}
                                    disabled={isSubmitting}
                                    className="flex-1 px-4 py-3 rounded-xl border border-[#272729] bg-[#0f0f0f] text-white font-medium
                                        hover:bg-[#272729] transition-all duration-200
                                        disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={submit}
                                    disabled={isSubmitting || !pwd}
                                    className="flex-1 flex items-center justify-center gap-2 px-4 py-3 rounded-xl
                                        bg-[#35cd48] text-[#0f0f0f] font-semibold
                                        hover:bg-[#35cd48]/90 transition-all duration-200
                                        disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    {isSubmitting ? (
                                        <motion.div
                                            className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full"
                                            animate={{ rotate: 360 }}
                                            transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                                        />
                                    ) : (
                                        <>
                                            <Unlock className="w-4 h-4" />
                                            Unlock
                                        </>
                                    )}
                                </button>
                            </div>
                        </div>
                    </motion.div>
                </motion.div>
            )}
        </AnimatePresence>
    )
}
