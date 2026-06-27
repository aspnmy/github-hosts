import { GITHUB_URLS } from "./constants"
import { Bindings } from "./types"

export async function getDomains(env: Bindings): Promise<string[]> {
  if (env.DOMAINS_URL) {
    try {
      const resp = await fetch(env.DOMAINS_URL)
      if (resp.ok) {
        const text = await resp.text()
        const domains = text.split("\n").map(d => d.trim()).filter(d => d.length > 0 && !d.startsWith("#"))
        if (domains.length > 0) return domains
      }
    } catch (e) {
      console.error("Failed to fetch domains from URL, using defaults", e)
    }
  }
  return GITHUB_URLS
}
