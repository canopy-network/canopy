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
                        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        onClick={onClose}
                    />

                    {/* Modal */}
                    <motion.div
                        className="relative w-full max-w-md max-h-[calc(100dvh-1.5rem)] sm:max-h-[calc(100dvh-2rem)] bg-card border border-white/10 rounded-2xl shadow-2xl overflow-hidden flex flex-col"
                        initial={{ opacity: 0, scale: 0.95, y: 20 }}
                        animate={{ opacity: 1, scale: 1, y: 0 }}
                        exit={{ opacity: 0, scale: 0.95, y: 20 }}
                        transition={{ duration: 0.2, ease: 'easeOut' }}
                    >
                        {/* Close button */}
                        <button
                            onClick={onClose}
                            className="absolute top-4 right-4 p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent/60 transition-colors"
                        >
                            <X className="w-5 h-5" />
                        </button>

                        <div className="p-4 pt-7 sm:p-6 sm:pt-8 overflow-y-auto min-h-0">
                            {/* Icon */}
                            <div className="flex justify-center mb-5">
                                <div className="relative w-16 h-16 rounded-full bg-primary/15 border border-primary/25 flex items-center justify-center">
                                    <Shield className="w-8 h-8 text-primary" />
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
                                            w-full bg-background/50 text-foreground rounded-xl px-4 py-3 pr-12
                                            border transition-all duration-200 outline-none
                                            placeholder:text-muted-foreground
                                            ${err
                                                ? 'border-red-500/50 focus:border-red-500 focus:ring-2 focus:ring-red-500/20'
                                                : 'border-border/50 focus:border-primary/50 focus:ring-2 focus:ring-primary/20'
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
                                            className="flex items-center gap-2 text-red-400 text-sm"
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
                                    className="flex-1 px-4 py-3 rounded-xl bg-muted/50 text-foreground/80 font-medium
                                        hover:bg-muted/70 hover:text-foreground transition-all duration-200
                                        disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={submit}
                                    disabled={isSubmitting || !pwd}
                                    className="flex-1 flex items-center justify-center gap-2 px-4 py-3 rounded-xl
                                        bg-primary text-primary-foreground font-semibold
                                        hover:bg-primary/90 transition-all duration-200
                                        disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                    {isSubmitting ? (
                                        <motion.div
                                            className="w-5 h-5 border-2 border-bg-primary/30 border-t-bg-primary rounded-full"
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

