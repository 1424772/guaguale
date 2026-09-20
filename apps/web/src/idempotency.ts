export function newIdempotencyKey(prefix: string) {
  if (typeof globalThis.crypto?.randomUUID === 'function') {
    return `${prefix}-${globalThis.crypto.randomUUID()}`
  }

  const random = new Uint32Array(4)
  if (typeof globalThis.crypto?.getRandomValues === 'function') {
    globalThis.crypto.getRandomValues(random)
    const suffix = Array.from(random, (value) => value.toString(16).padStart(8, '0')).join('')
    return `${prefix}-${Date.now().toString(36)}-${suffix}`
  }

  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 14)}`
}
