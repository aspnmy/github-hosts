import { DNS_PROVIDERS, HOSTS_TEMPLATE } from "../constants"
import { getDomains } from "../domain-config"
import { Bindings } from "../types"

export type HostEntry = [string, string]

export interface DomainData {
  ip: string
  lastUpdated: string
  lastChecked: string
}

interface DnsQuestion {
  name: string
  type: number
}

interface DnsAnswer {
  name: string
  type: number
  TTL: number
  data: string
}

interface DnsResponse {
  Status: number
  TC: boolean
  RD: boolean
  RA: boolean
  AD: boolean
  CD: boolean
  Question: DnsQuestion[]
  Answer: DnsAnswer[]
}

async function retry<T>(
  fn: () => Promise<T>,
  retries: number = 3,
  delay: number = 1000
): Promise<T> {
  try {
    return await fn()
  } catch (error) {
    if (retries === 0) throw error
    await new Promise((resolve) => setTimeout(resolve, delay))
    return retry(fn, retries - 1, delay * 2)
  }
}

export async function fetchIPFromIPAddress(
  domain: string,
  providerName?: string
): Promise<string | null> {
  const provider =
    DNS_PROVIDERS.find((p) => p.name === providerName) || DNS_PROVIDERS[0]

  try {
    const response = await retry(() =>
      fetch(provider.url(domain), { headers: provider.headers })
    )

    if (!response.ok) return null

    const data = (await response.json()) as DnsResponse

    const aRecord = data.Answer?.find((answer) => answer.type === 1)
    const ip = aRecord?.data

    if (ip && /^\d+\.\d+\.\d+\.\d+$/.test(ip)) {
      return ip
    }
  } catch (error) {
    console.error(`Error with DNS provider:`, error)
  }

  return null
}

export async function fetchLatestHostsData(env?: Bindings): Promise<HostEntry[]> {
  const entries: HostEntry[] = []
  const batchSize = 5

  const domains = await getDomains(env)

  for (let i = 0; i < domains.length; i += batchSize) {
    console.log(
      `Processing batch ${i / batchSize + 1}/${Math.ceil(domains.length / batchSize)}`
    )

    const batch = domains.slice(i, i + batchSize)
    const batchResults = await Promise.all(
      batch.map(async (domain) => {
        const ip = await fetchIPFromIPAddress(domain)
        console.log(`Domain: ${domain}, IP: ${ip}`)
        return ip ? ([ip, domain] as HostEntry) : null
      })
    )

    entries.push(
      ...batchResults.filter((result): result is HostEntry => result !== null)
    )

    if (i + batchSize < domains.length) {
      await new Promise((resolve) => setTimeout(resolve, 2000))
    }
  }

  console.log(`Total entries found: ${entries.length}`)
  return entries
}

export function formatHostsFile(entries: HostEntry[]): string {
  const content = entries
    .map(([ip, domain]) => `${ip.padEnd(30)}${domain}`)
    .join("\n")

  const updateTime = new Date().toLocaleString("en-US", {
    timeZone: "Asia/Shanghai",
    hour12: false,
  })

  return HOSTS_TEMPLATE.replace("{content}", content).replace(
    "{updateTime}",
    updateTime
  )
}
