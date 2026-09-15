export type User = {
  id: number
  username: string
  balance: number
  luckLevel: number
  createdAt: string
}

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
  price: number
  luckLevel: number
  prizeTier?: string
  reward?: number
  symbols?: string[]
  state: TicketState
  createdAt: string
  scratchedAt?: string
  redeemedAt?: string
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
  tickets: () => request<{ tickets: Ticket[] }>('/api/v1/tickets'),
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
}
