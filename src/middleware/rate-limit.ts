import { Context, Next } from "hono"

interface RateLimitConfig {
  limit: number
  windowMs: number
}

export function rateLimit(config: RateLimitConfig) {
  const hits = new Map<string, { count: number; resetAt: number }>()

  return async (c: Context, next: Next) => {
    const key = c.req.header("cf-connecting-ip") || "unknown"
    const now = Date.now()
    let record = hits.get(key)

    if (!record || now > record.resetAt) {
      record = { count: 0, resetAt: now + config.windowMs }
      hits.set(key, record)
    }

    record.count++
    if (record.count > config.limit) {
      return c.json({ error: "Rate limit exceeded" }, 429)
    }

    await next()
  }
}
