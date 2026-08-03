import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import DatabaseStatus from './DatabaseStatus'
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

describe('DatabaseStatus', () => {
  it('shows loading and then the database diagnostic', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockResolvedValue(Response.json(systemInfo, { status: 200 })),
    )

    render(<DatabaseStatus />)

    expect(screen.getByText('Consultando el backend y PostgreSQL...')).toBeInTheDocument()
    expect(await screen.findByText('Conectado')).toBeInTheDocument()
    expect(screen.getAllByText('loteosapp')).toHaveLength(2)
    expect(screen.getByText('PostgreSQL 18.4')).toBeInTheDocument()
    expect(screen.getByText('172.22.0.2:5432')).toBeInTheDocument()
  })

  it('shows a useful error when the backend is unavailable', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 503 })),
    )

    render(<DatabaseStatus />)

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'El backend respondió con HTTP 503',
    )
    expect(screen.getByText('Desconectado')).toBeInTheDocument()
  })

  it('cancels the request when it is unmounted', () => {
    const fetchMock = vi.fn<typeof fetch>().mockImplementation((_input, init) => {
      return new Promise((_resolve, reject) => {
        init?.signal?.addEventListener('abort', () => {
          reject(new DOMException('Request aborted', 'AbortError'))
        })
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    const { unmount } = render(<DatabaseStatus />)
    unmount()

    expect(fetchMock).toHaveBeenCalledOnce()
  })
})
