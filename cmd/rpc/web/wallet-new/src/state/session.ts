import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

type SessionState = {
  unlockedUntil: number
  password?: string
  address?: string
  unlock: (address: string, password: string, ttlSec: number) => void
  lock: () => void
  isUnlocked: () => boolean
  getRemainingTime: () => number
}

// Use sessionStorage for persistence within browser session
export const useSession = create<SessionState>()(
  persist(
    (set, get) => ({
      unlockedUntil: 0,
      password: undefined,
      address: undefined,
      unlock: (address, password, ttlSec) =>
        set({ address, password, unlockedUntil: Date.now() + ttlSec * 1000 }),
      lock: () => set({ password: undefined, unlockedUntil: 0, address: undefined }),
      isUnlocked: () => Date.now() < get().unlockedUntil && !!get().password,
      getRemainingTime: () => Math.max(0, Math.floor((get().unlockedUntil - Date.now()) / 1000)),
    }),
    {
      name: 'wallet-session',
      storage: createJSONStorage(() => sessionStorage),
      // Only persist these fields
      partialize: (state) => ({
        unlockedUntil: state.unlockedUntil,
        password: state.password,
        address: state.address,
      }),
    }
  )
)

let idleRenewAttached = false

export function attachIdleRenew(ttlSec: number) {
  if (idleRenewAttached) return // Prevent multiple attachments
  idleRenewAttached = true

  const renew = () => {
    const s = useSession.getState()
    if (s.password && s.isUnlocked()) {
      useSession.setState({ unlockedUntil: Date.now() + ttlSec * 1000 })
    }
  }
  ;['click','keydown','mousemove','touchstart'].forEach(e =>
    window.addEventListener(e, renew, { passive: true })
  )
}
