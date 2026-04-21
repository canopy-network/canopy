import type { CSSProperties } from 'react'

type LandingConfettiPiece = {
    id: number
    left: string
    width: number
    height: number
    color: string
    duration: number
    delay: number
    drift: number
    opacity: number
    rotate: number
}

const LANDING_CONFETTI_PIECES: LandingConfettiPiece[] = [
    { id: 1, left: '2%', width: 5, height: 14, color: 'rgba(53,205,72,0.82)', duration: 24, delay: -3, drift: -20, opacity: 0.75, rotate: 300 },
    { id: 2, left: '6%', width: 6, height: 12, color: 'rgba(96,165,250,0.82)', duration: 22, delay: -12, drift: 18, opacity: 0.72, rotate: 280 },
    { id: 3, left: '11%', width: 4, height: 11, color: 'rgba(239,68,68,0.76)', duration: 26, delay: -8, drift: -14, opacity: 0.68, rotate: 320 },
    { id: 4, left: '16%', width: 5, height: 13, color: 'rgba(255,255,255,0.72)', duration: 21, delay: -15, drift: 12, opacity: 0.65, rotate: 260 },
    { id: 5, left: '21%', width: 6, height: 14, color: 'rgba(53,205,72,0.78)', duration: 25, delay: -5, drift: -16, opacity: 0.7, rotate: 310 },
    { id: 6, left: '26%', width: 5, height: 12, color: 'rgba(96,165,250,0.8)', duration: 23, delay: -10, drift: 15, opacity: 0.7, rotate: 295 },
    { id: 7, left: '31%', width: 4, height: 10, color: 'rgba(239,68,68,0.72)', duration: 27, delay: -18, drift: -11, opacity: 0.66, rotate: 300 },
    { id: 8, left: '36%', width: 6, height: 15, color: 'rgba(53,205,72,0.84)', duration: 24, delay: -2, drift: 20, opacity: 0.74, rotate: 330 },
    { id: 9, left: '41%', width: 5, height: 12, color: 'rgba(96,165,250,0.82)', duration: 22, delay: -16, drift: -15, opacity: 0.72, rotate: 280 },
    { id: 10, left: '46%', width: 4, height: 11, color: 'rgba(255,255,255,0.7)', duration: 26, delay: -9, drift: 13, opacity: 0.65, rotate: 255 },
    { id: 11, left: '51%', width: 6, height: 13, color: 'rgba(53,205,72,0.8)', duration: 23, delay: -13, drift: -18, opacity: 0.72, rotate: 300 },
    { id: 12, left: '56%', width: 5, height: 12, color: 'rgba(239,68,68,0.75)', duration: 25, delay: -6, drift: 16, opacity: 0.68, rotate: 290 },
    { id: 13, left: '61%', width: 4, height: 10, color: 'rgba(96,165,250,0.8)', duration: 27, delay: -17, drift: -12, opacity: 0.7, rotate: 315 },
    { id: 14, left: '66%', width: 6, height: 15, color: 'rgba(53,205,72,0.83)', duration: 24, delay: -11, drift: 19, opacity: 0.74, rotate: 325 },
    { id: 15, left: '71%', width: 5, height: 13, color: 'rgba(255,255,255,0.72)', duration: 22, delay: -4, drift: -17, opacity: 0.66, rotate: 265 },
    { id: 16, left: '76%', width: 4, height: 11, color: 'rgba(239,68,68,0.74)', duration: 26, delay: -14, drift: 14, opacity: 0.67, rotate: 300 },
    { id: 17, left: '81%', width: 6, height: 14, color: 'rgba(96,165,250,0.82)', duration: 23, delay: -7, drift: -20, opacity: 0.72, rotate: 310 },
    { id: 18, left: '86%', width: 5, height: 12, color: 'rgba(53,205,72,0.8)', duration: 25, delay: -19, drift: 15, opacity: 0.71, rotate: 285 },
    { id: 19, left: '91%', width: 4, height: 10, color: 'rgba(255,255,255,0.7)', duration: 27, delay: -1, drift: -13, opacity: 0.64, rotate: 270 },
    { id: 20, left: '96%', width: 6, height: 14, color: 'rgba(96,165,250,0.8)', duration: 24, delay: -20, drift: 17, opacity: 0.7, rotate: 305 },
]

export default function ExplorerConfetti() {
    return (
        <div className="pointer-events-none absolute inset-0 z-0 overflow-hidden" aria-hidden>
            {LANDING_CONFETTI_PIECES.map((piece) => {
                const confettiStyle: CSSProperties & Record<string, string | number> = {
                    left: piece.left,
                    width: `${piece.width}px`,
                    height: `${piece.height}px`,
                    background: piece.color,
                    animationDuration: `${piece.duration}s`,
                    animationDelay: `${piece.delay}s`,
                    ['--landing-confetti-opacity']: piece.opacity,
                    ['--landing-confetti-x1']: `${Math.round(piece.drift * -0.45)}px`,
                    ['--landing-confetti-x2']: `${Math.round(piece.drift * 0.2)}px`,
                    ['--landing-confetti-x3']: `${Math.round(piece.drift * -0.2)}px`,
                    ['--landing-confetti-x4']: `${piece.drift}px`,
                    ['--landing-confetti-rotate']: `${piece.rotate}deg`,
                }

                return <span key={piece.id} className="landing-confetti-piece" style={confettiStyle} />
            })}
        </div>
    )
}
