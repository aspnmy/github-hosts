import { Context, MiddlewareHandler } from "hono"
import { RateLimitConfig } from "../types"

const DEFAULT_CONFIG: RateLimitConfig = {
  limit: 60,
  windowMs: 60_000,
}

/**
 * Normalize IPv6 to /64 prefix to prevent bypass via suffix rotation.
 * IPv4 addresses are returned as-is.
 */
function normalizeIP(ip: string): string {
  if (!ip.includes(":")) {
    return ip
  }

  // Expand :: shorthand to full form
  const parts = ip.split(":")
  if (parts.length < 8) {
    const emptyIndex = parts.indexOf("")
    if (emptyIndex !== -1) {
      const missing = 8 - parts.filter((p) => p !== "").length
      const fill = Array(missing).fill("0000")
      parts.splice(emptyIndex, 1, ...fill)
    }
  }

  // Take first 4 groups (/64 prefix)
  return parts.slice(0, 4).join(":")
}

function getWindowStart(windowMs: number): number {
  return Math.floor(Date.now() / windowMs) * windowMs
}

// 内存中的限流计数器，Worker 实例生命周期内有效
const counters = new Map<string, { count: number; windowEnd: number }>()

export function rateLimit(
  config: Partial<RateLimitConfig> = {}
): MiddlewareHandler {
  const { limit, windowMs } = { ...DEFAULT_CONFIG, ...config }

  return async (c, next) => {
    const ip = c.req.header("cf-connecting-ip") || "unknown"
    const normalizedIP = normalizeIP(ip)
    const windowStart = getWindowStart(windowMs)
    const key = `ratelimit:${normalizedIP}:${windowStart}`

    const now = Date.now()
    const entry = counters.get(key)

    const current = entry && now < entry.windowEnd ? entry.count : 0

    const windowEnd = windowStart + windowMs
    const resetTimestamp = Math.ceil(windowEnd / 1000)
    const remaining = Math.max(0, limit - current - 1)
    const retryAfter = Math.ceil((windowEnd - now) / 1000)

    c.header("RateLimit-Limit", String(limit))
    c.header("RateLimit-Remaining", String(Math.max(0, limit - current - 1)))
    c.header("RateLimit-Reset", String(resetTimestamp))

    if (current >= limit) {
      c.header("Retry-After", String(retryAfter))
      return c.json(
        { error: "Too Many Requests", retryAfter },
        429
      )
    }

    counters.set(key, { count: current + 1, windowEnd })

    // 定期清理过期计数器，防止内存泄漏
    if (counters.size > 1000) {
      for (const [k, v] of counters) {
        if (now >= v.windowEnd) {
          counters.delete(k)
        }
      }
    }

    await next()
  }
}
