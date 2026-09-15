export type User = {
  id: number
  username: string
  balance: number
  luckLevel: number
  scratchLevel: number
  trashOwned: boolean
  cardSlotsOwned: boolean
  createdAt: string
}

export type ShopItem = {
  code: 'luck' | 'scratch-range' | 'trash' | 'card-slots'
  name: string
  category: 'luck' | 'efficiency' | 'safety'
  level: number
  maxLevel: number
  effectPercent: number
  nextEffectPercent?: number
  effectText: string
  nextEffectText?: string
  nextPrice?: number
  totalSpent: number
  description: string
  notice?: string
  relockedCards: string[]
}

export type LuckTierImpact = {
  label: string
  rewardText: string
  currentBasisPoint: number
  nextBasisPoint: number
}

export type LuckCardImpact = {
  cardCode: string
  cardName: string
  currentLevel: number
  nextLevel: number
  currentRtpBasisPoint: number
  nextRtpBasisPoint: number
  tiers: LuckTierImpact[]
}

export type ShopStatus = { items: ShopItem[]; luckCards: LuckCardImpact[] }

export type Card = {
  code: string
  name: string
  price: number
  implemented: boolean
  unlocked: boolean
}

export type TicketState = 'purchased' | 'scratched' | 'redeemed' | 'discarded'

export type Ticket = {
  id: string
  cardCode: string
  cardName: string
  nominalPrice: number
  pricePaid: number
  source: 'purchase' | 'daily_wheel'
  wheelDate?: string
  luckLevel: number
  prizeTier?: string
  reward?: number
  symbols?: string[]
  state: TicketState
  location: 'tray' | 'desk' | 'slot'
  deskX: number
  deskY: number
  rotation: number
  zIndex: number
  slotIndex?: number
  createdAt: string
  scratchedAt?: string
  redeemedAt?: string
  discardedAt?: string
}

export type WheelPoolItem = {
  cardCode: string
  cardName: string
  price: number
  weight: number
  basisPoint: number
}

export type PlateAction = {
  id: string
  sequence: number
  state: 'started' | 'completed'
  startedAt: string
  availableAt: string
  completedAt?: string
}

export type DailyStatus = {
  date: string
  loginClaimed: boolean
  platesCompleted: number
  plateLimit: number
  activePlate?: PlateAction
  wheelUsed: boolean
  wheelTicketId?: string
  wheelCardCode?: string
  wheelCardName?: string
  wheelPool: WheelPoolItem[]
}

type ApiErrorBody = {
  error?: {
    code?: string
    message?: string
  }
}

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    credentials: 'same-origin',
    ...init,
    headers: {
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })

  if (!response.ok) {
    let body: ApiErrorBody = {}
    try {
      body = await response.json() as ApiErrorBody
    } catch {
      // The fallback below is intentionally user friendly.
    }
    throw new ApiError(
      response.status,
      body.error?.code ?? 'request_failed',
      body.error?.message ?? '请求失败，请稍后重试',
    )
  }

  if (response.status === 204) {
    return undefined as T
  }
  return response.json() as Promise<T>
}

export const api = {
  me: () => request<{ user: User }>('/api/v1/me'),
  register: (username: string, password: string) => request<{ user: User }>('/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, password, ageConfirmed: true }),
  }),
  login: (username: string, password: string) => request<{ user: User }>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  }),
  logout: () => request<void>('/api/v1/auth/logout', { method: 'POST' }),
  cards: () => request<{ cards: Card[] }>('/api/v1/cards'),
  shop: () => request<{ shop: ShopStatus }>('/api/v1/shop'),
  upgradeItem: (itemCode: string, idempotencyKey: string) => request<{
    user: User
    shop: ShopStatus
    itemCode: string
    idempotent: boolean
  }>(`/api/v1/shop/${itemCode}/upgrade`, {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
  }),
  tickets: () => request<{ tickets: Ticket[] }>('/api/v1/tickets'),
  daily: () => request<{ daily: DailyStatus }>('/api/v1/daily'),
  claimDailyLogin: () => request<{ user: User; daily: DailyStatus; idempotent: boolean }>('/api/v1/daily/login-claim', {
    method: 'POST',
  }),
  startPlate: () => request<{ daily: DailyStatus; idempotent: boolean }>('/api/v1/daily/plates/start', {
    method: 'POST',
  }),
  completePlate: (actionId: string) => request<{ user: User; daily: DailyStatus; idempotent: boolean }>(`/api/v1/daily/plates/${actionId}/complete`, {
    method: 'POST',
  }),
  spinWheel: () => request<{ user: User; ticket: Ticket; daily: DailyStatus; idempotent: boolean }>('/api/v1/daily/wheel/spin', {
    method: 'POST',
  }),
  purchase: (cardCode: string, idempotencyKey: string) => request<{
    user: User
    ticket: Ticket
    idempotent: boolean
  }>(`/api/v1/cards/${cardCode}/purchase`, {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
  }),
  scratch: (ticketId: string) => request<{ ticket: Ticket }>(`/api/v1/tickets/${ticketId}/scratch`, {
    method: 'POST',
  }),
  redeem: (ticketId: string) => request<{
    user: User
    ticket: Ticket
    idempotent: boolean
  }>(`/api/v1/tickets/${ticketId}/redeem`, {
    method: 'POST',
  }),
  placeTicket: (ticketId: string, placement: {
    location: 'desk' | 'slot'
    deskX: number
    deskY: number
    rotation: number
    zIndex: number
    slotIndex?: number
  }) => request<{ ticket: Ticket }>(`/api/v1/tickets/${ticketId}/placement`, {
    method: 'PATCH',
    body: JSON.stringify(placement),
  }),
  discardTicket: (ticketId: string) => request<{ ticket: Ticket }>(`/api/v1/tickets/${ticketId}/discard`, {
    method: 'POST',
  }),
}
