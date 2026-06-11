export interface Bindings {
  API_KEY: string
  ASSETS: { get(key: string): Promise<string | null> }
}

export interface RateLimitConfig {
  limit: number
  windowMs: number
}
