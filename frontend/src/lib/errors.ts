import { ApiError } from '../api/client'

// errorMessage extracts a user-presentable message from a caught error: the
// server's message for ApiError, else the fallback (when given), else the
// stringified error.
export function errorMessage(e: unknown, fallback?: string): string {
  if (e instanceof ApiError) return e.message
  return fallback ?? String(e)
}
