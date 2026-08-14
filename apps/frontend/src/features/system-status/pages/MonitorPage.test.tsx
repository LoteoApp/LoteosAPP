import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useAuth } from 'react-oidc-context'
import MonitorPage from './MonitorPage'

vi.mock('react-oidc-context', () => ({
  useAuth: vi.fn(),
}))

const useAuthMock = vi.mocked(useAuth)

describe('MonitorPage', () => {
  it('renders the environment diagnostic heading', () => {
    useAuthMock.mockReturnValue({ isLoading: true } as ReturnType<typeof useAuth>)
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockReturnValue(new Promise(() => {})),
    )

    render(<MonitorPage />)

    expect(
      screen.getByRole('heading', { name: 'Diagnóstico del entorno' }),
    ).toBeInTheDocument()
  })
})
