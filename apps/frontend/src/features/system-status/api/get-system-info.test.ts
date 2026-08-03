import { describe, expect, it, vi } from 'vitest'
import { getSystemInfo } from './get-system-info'
import type { SystemInfo } from '../types'

const systemInfo: SystemInfo = {
  service: 'loteosapp-backend',
  status: 'ok',
  checked_at: '2026-08-03T20:00:00Z',
  database: {
    connected: true,
    version: 'PostgreSQL 18.4',
    database_name: 'loteosapp',
    user: 'loteosapp',
    server_address: '172.22.0.2',
    server_port: 5432,
    database_time: '2026-08-03T20:00:00Z',
  },
  pool: {
    max_connections: 10,
    total_connections: 2,
    acquired_connections: 1,
    idle_connections: 1,
    new_connections: 2,
    closed_connections: 0,
  },
}

describe('getSystemInfo', () => {
  it('returns the backend diagnostic', async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValue(Response.json(systemInfo, { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    const signal = new AbortController().signal
    await expect(getSystemInfo(signal)).resolves.toEqual(systemInfo)
    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/api/v1/system', {
      signal,
      cache: 'no-store',
    })
  })

  it('rejects responses outside the successful range', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 503 })),
    )

    await expect(getSystemInfo(new AbortController().signal)).rejects.toThrow(
      'El backend respondió con HTTP 503',
    )
  })
})
