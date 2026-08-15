import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router'
import MonitorPage from './MonitorPage'
import {
  AuthContext,
  type AuthContextValue,
} from '../../auth/hooks/use-auth'

const authValue: AuthContextValue = {
  isLoading: true,
  session: null,
  user: null,
  error: null,
  login: vi.fn(),
  logout: vi.fn(),
}

describe('MonitorPage', () => {
  it('renders the environment diagnostic heading', () => {
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockReturnValue(new Promise(() => {})),
    )

    render(
      <AuthContext.Provider value={authValue}>
        <MemoryRouter>
          <MonitorPage />
        </MemoryRouter>
      </AuthContext.Provider>,
    )

    expect(
      screen.getByRole('heading', { name: 'Diagnóstico del entorno' }),
    ).toBeInTheDocument()
  })
})
