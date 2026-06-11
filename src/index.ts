import { Hono } from "hono"
import {
  fetchLatestHostsData,
  fetchIPFromIPAddress,
  formatHostsFile,
} from "./services/hosts"
import { Bindings } from "./types"
import { GITHUB_URLS } from "./constants"
import { rateLimit } from "./middleware/rate-limit"

const app = new Hono<{ Bindings: Bindings }>()

const ALLOWED_DOMAINS = new Set(GITHUB_URLS)

app.get("/", async (c) => {
  const html = await c.env.ASSETS.get("index.html")
  if (!html) {
    return c.text("Template not found", 404)
  }

  return c.html(html)
})

app.get("/hosts.json", async (c) => {
  const data = await fetchLatestHostsData()
  return c.json(data)
})

app.get("/hosts", async (c) => {
  const data = await fetchLatestHostsData()
  const hostsContent = formatHostsFile(data)
  return c.text(hostsContent)
})

// 管理接口限流: 5 请求/分钟
app.post("/reset", rateLimit({ limit: 5, windowMs: 60_000 }), async (c) => {
  const apiKey = c.req.query("key")

  // 验证 API key
  if (apiKey !== c.env.API_KEY) {
    return c.json({ error: "Unauthorized" }, 401)
  }

  const newEntries = await fetchLatestHostsData()

  return c.json({
    message: "Refresh completed",
    entriesCount: newEntries.length,
    entries: newEntries,
  })
})

// 域名查询限流: 30 请求/分钟
app.get("/:domain", rateLimit({ limit: 30, windowMs: 60_000 }), async (c) => {
  const domain = c.req.param("domain")

  // 只允许查询预定义域名
  if (!ALLOWED_DOMAINS.has(domain)) {
    return c.json(
      {
        error: "Domain not in allowed list",
        hint: "Use /hosts endpoint to get all supported domains",
      },
      400
    )
  }

  const ip = await fetchIPFromIPAddress(domain)

  if (!ip) {
    return c.json({ error: "Failed to resolve domain" }, 500)
  }

  const currentTime = new Date().toISOString()
  return c.json({
    ip,
    lastUpdated: currentTime,
    lastChecked: currentTime,
  })
})

export default {
  fetch: app.fetch,
}
