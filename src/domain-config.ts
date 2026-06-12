import { Bindings } from "./types"

const DEFAULT_TTL = 5 * 60 * 1000 // 5 minutes

let cache: { domains: string[]; expires: number } | null = null

export async function getDomains(env?: Bindings, ttlMs = DEFAULT_TTL): Promise<string[]> {
  if (cache && Date.now() < cache.expires) return cache.domains

  const url = env?.DOMAINS_URL ||
    "https://raw.githubusercontent.com/aspnmy/github-hosts/nokv/domains.txt"

  try {
    const res = await fetch(url)
    if (!res.ok) throw new Error(`Failed to fetch domains: ${res.status}`)
    const text = await res.text()
    const domains = text
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter((l) => l && !l.startsWith("#"))

    cache = { domains, expires: Date.now() + ttlMs }
    return domains
  } catch (e) {
    console.error("getDomains fetch failed:", e)
    // fallback to built-in lists
    try {
      const mod = await import("./constants")
      const domains = [...(mod.GITHUB_URLS || [])]
      cache = { domains, expires: Date.now() + ttlMs }
      return domains
    } catch (e2) {
      console.error("getDomains fallback failed:", e2)
      return []
    }
  }
}

export function clearDomainsCache() {
  cache = null
}
