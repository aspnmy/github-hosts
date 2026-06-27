import { HOSTS_TEMPLATE } from "../constants"
import { Bindings } from "../types"
import { getDomains } from "../domain-config"

const DOMAIN_CACHE_TTL_MS = 60 * 60 * 1000
export type HostEntry = [string, string]

interface DomainData { ip: string; lastUpdated: string; lastChecked: string }
interface DomainDataList { [key: string]: DomainData }
interface KVData { domain_data: DomainDataList; lastUpdated: string }
interface DnsAnswer { name: string; type: number; TTL: number; data: string }
interface DnsResponse { Status: number; Answer?: DnsAnswer[] }

async function retry<T>(fn: () => Promise<T>, retries = 3, delay = 1000): Promise<T> {
  try { return await fn() }
  catch (error) {
    if (retries === 0) throw error
    await new Promise(r => setTimeout(r, delay))
    return retry(fn, retries - 1, delay * 2)
  }
}

export async function fetchIPFromIPAddress(domain: string, providerName?: string): Promise<string | null> {
  const providers = [
    { url: (d: string) => `https://1.1.1.1/dns-query?name=${d}&type=A`, headers: { Accept: "application/dns-json" }, name: "Cloudflare" },
    { url: (d: string) => `https://dns.google/resolve?name=${d}&type=A`, headers: { Accept: "application/dns-json" }, name: "Google" },
  ]
  const provider = providers.find(p => p.name === providerName) || providers[0]
  try {
    const resp = await retry(() => fetch(provider.url(domain), { headers: provider.headers }))
    if (!resp.ok) return null
    const data = await resp.json() as DnsResponse
    const a = data.Answer?.find(a => a.type === 1)
    return a?.data && /^\d+\.\d+\.\d+\.\d+$/.test(a.data) ? a.data : null
  } catch { return null }
}

export async function fetchLatestHostsData(env: Bindings): Promise<HostEntry[]> {
  const domains = await getDomains(env)
  const entries: HostEntry[] = []
  const batchSize = 5
  for (let i = 0; i < domains.length; i += batchSize) {
    const batch = domains.slice(i, i + batchSize)
    const results = await Promise.all(
      batch.map(async d => { const ip = await fetchIPFromIPAddress(d); return ip ? [ip, d] as HostEntry : null })
    )
    entries.push(...results.filter(r => r !== null) as HostEntry[])
    if (i + batchSize < domains.length) await new Promise(r => setTimeout(r, 500))
  }
  return entries
}

export function formatHostsFile(entries: HostEntry[]): string {
  const content = entries.map(([ip, d]) => `${ip.padEnd(30)}${d}`).join("\n")
  const updateTime = new Date().toLocaleString("en-US", { timeZone: "Asia/Shanghai", hour12: false })
  return HOSTS_TEMPLATE.replace("{content}", content).replace("{updateTime}", updateTime)
}

export async function getDomainData(env: Bindings, domain: string): Promise<DomainData | null> {
  const ip = await fetchIPFromIPAddress(domain)
  if (!ip) return null
  const now = new Date().toISOString()
  return { ip, lastUpdated: now, lastChecked: now }
}
