import type { CSSProperties } from 'react'

const coatingCellCounts: Record<string, number> = {
  'lingqian-ticket': 1,
  'street-store': 7,
  'arcade-challenge': 9,
  'gold-mine': 7,
  'rocket-launch': 4,
  'deep-sea-salvage': 8,
  'eternal-color-diamond': 5,
  'all-in': 1,
}

export function TicketScratchCoating({ cardCode, progress = 0 }: { cardCode: string; progress?: number }) {
  const count = coatingCellCounts[cardCode] ?? 1
  const safeProgress = Math.max(0, Math.min(100, progress))
  return (
    <span
      className={`ticket-scratch-coating card-${cardCode}`}
      style={{ '--scratch-progress': `${safeProgress}%` } as CSSProperties}
      aria-hidden="true"
    >
      {Array.from({ length: count }, (_, index) => <i key={index} />)}
    </span>
  )
}
