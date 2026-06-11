import { describe, it, expect, vi, beforeEach } from "vitest"
import { fetchIPFromIPAddress } from "../hosts"

const mockDnsResponse = (ip: string) => ({
  ok: true,
  json: () =>
    Promise.resolve({
      Status: 0,
      TC: false,
      RD: true,
      RA: true,
      AD: false,
      CD: false,
      Question: [{ name: "github.com", type: 1 }],
      Answer: [{ name: "github.com", type: 1, TTL: 60, data: ip }],
    }),
})

describe("fetchIPFromIPAddress", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("should successfully extract IP from DNS response", async () => {
    global.fetch = vi.fn().mockResolvedValue(mockDnsResponse("140.82.114.25"))

    const result = await fetchIPFromIPAddress("github.com")
    expect(result).toBe("140.82.114.25")
  })

  it("should return null when no A record in response", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          Status: 0,
          Question: [{ name: "invalid.com", type: 1 }],
          Answer: [],
        }),
    })

    const result = await fetchIPFromIPAddress("invalid-domain.com")
    expect(result).toBeNull()
  })

  it("should handle fetch errors gracefully", async () => {
    global.fetch = vi.fn().mockRejectedValue(new Error("Network error"))

    const result = await fetchIPFromIPAddress("github.com")
    expect(result).toBeNull()
  })

  it("should handle non-ok response", async () => {
    global.fetch = vi.fn().mockResolvedValue({ ok: false })

    const result = await fetchIPFromIPAddress("github.com")
    expect(result).toBeNull()
  })

  it("should return first A record when multiple exist", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          Status: 0,
          Question: [{ name: "github.com", type: 1 }],
          Answer: [
            { name: "github.com", type: 1, TTL: 60, data: "140.82.114.4" },
            { name: "github.com", type: 1, TTL: 60, data: "140.82.114.5" },
          ],
        }),
    })

    const result = await fetchIPFromIPAddress("github.com")
    expect(result).toBe("140.82.114.4")
  })
})
