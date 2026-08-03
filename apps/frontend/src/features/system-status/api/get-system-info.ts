import { apiUrl } from '../../../shared/config/env'
import type { SystemInfo } from '../types'

export async function getSystemInfo(signal: AbortSignal): Promise<SystemInfo> {
  const response = await fetch(`${apiUrl}/api/v1/system`, {
    signal,
    cache: 'no-store',
  })

  if (!response.ok) {
    throw new Error(`El backend respondió con HTTP ${response.status}`)
  }

  return (await response.json()) as SystemInfo
}
