export interface Bindings {
  API_KEY: string
  ASSETS: { get(key: string): Promise<string | null> }
  DOMAINS_URL?: string
}

export interface RateLimitConfig {
  limit: number
  windowMs: number
}
