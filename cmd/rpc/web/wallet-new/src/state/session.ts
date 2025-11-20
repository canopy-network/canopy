import { create } from 'zustand'

type SessionState = {
  unlockedUntil: number
  password?: string
  address?: string
  unlock: (address: string, password: string, ttlSec: number) => void
  lock: () => void
  isUnlocked: () => boolean
}

export const useSession = create<SessionState>((set, get) => ({
  unlockedUntil: 0,
  password: undefined,
  address: undefined,
  unlock: (address, password, ttlSec) =>
    set({ address, password, unlockedUntil: Date.now() + ttlSec * 1000 }),
  lock: () => set({ password: undefined, unlockedUntil: 0 }),
  isUnlocked: () => Date.now() < get().unlockedUntil && !!get().password,
}))

export function attachIdleRenew(ttlSec: number) {
  const renew = () => {
    const s = useSession.getState()
    if (s.password) useSession.setState({ unlockedUntil: Date.now() + ttlSec * 1000 })
  }
  ;['click','keydown','mousemove','touchstart'].forEach(e => window.addEventListener(e, renew))
}
