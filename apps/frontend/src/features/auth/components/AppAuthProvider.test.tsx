import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Session } from '@supabase/supabase-js'
import AppAuthProvider from './AppAuthProvider'
import { supabaseClient } from '../../../shared/config/supabase-client'
import { useAuth } from '../hooks/use-auth'

vi.mock('../../../shared/config/supabase-client', () => ({
  supabaseClient: {
    auth: {
      onAuthStateChange: vi.fn(),
      signInWithPassword: vi.fn(),
      signOut: vi.fn(),
    },
  },
}))

const onAuthStateChangeMock = vi.mocked(supabaseClient.auth.onAuthStateChange)
const signInWithPasswordMock = vi.mocked(supabaseClient.auth.signInWithPassword)
const signOutMock = vi.mocked(supabaseClient.auth.signOut)
const unsubscribeMock = vi.fn()

function Consumer() {
  const auth = useAuth()

  return (
    <div>
      <p>{auth.isLoading ? 'loading' : 'ready'}</p>
      <p>{auth.session ? `signed-in:${auth.user?.email}` : 'signed-out'}</p>
      {auth.error && <p role="alert">{auth.error.message}</p>}
      <button type="button" onClick={() => auth.login('user@example.com', 'secret').catch(() => {})}>
        login
      </button>
      <button type="button" onClick={() => auth.logout().catch(() => {})}>
        logout
      </button>
    </div>
  )
}

function renderProvider() {
  return render(
    <AppAuthProvider>
      <Consumer />
    </AppAuthProvider>,
  )
}

describe('AppAuthProvider', () => {
  beforeEach(() => {
    onAuthStateChangeMock.mockReset().mockImplementation((callback) => {
      callback('INITIAL_SESSION', null)
      return { data: { subscription: { unsubscribe: unsubscribeMock } } } as unknown as ReturnType<
        typeof supabaseClient.auth.onAuthStateChange
      >
    })
    signInWithPasswordMock.mockReset()
    signOutMock.mockReset()
    unsubscribeMock.mockReset()
  })

  it('resolves loading once the initial Supabase session arrives', () => {
    renderProvider()

    expect(screen.getByText('ready')).toBeInTheDocument()
    expect(screen.getByText('signed-out')).toBeInTheDocument()
  })

  it('exposes the signed-in session and user from Supabase auth state changes', () => {
    const session = { user: { email: 'user@example.com' } } as unknown as Session
    onAuthStateChangeMock.mockImplementation((callback) => {
      callback('SIGNED_IN', session)
      return { data: { subscription: { unsubscribe: unsubscribeMock } } } as unknown as ReturnType<
        typeof supabaseClient.auth.onAuthStateChange
      >
    })

    renderProvider()

    expect(screen.getByText('signed-in:user@example.com')).toBeInTheDocument()
  })

  it('logs in through Supabase and clears a previous error', async () => {
    signInWithPasswordMock.mockResolvedValue({ error: null } as Awaited<
      ReturnType<typeof supabaseClient.auth.signInWithPassword>
    >)

    renderProvider()

    await userEvent.click(screen.getByRole('button', { name: 'login' }))

    expect(signInWithPasswordMock).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'secret',
    })
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('surfaces a login error', async () => {
    signInWithPasswordMock.mockResolvedValue({ error: new Error('credenciales inválidas') } as Awaited<
      ReturnType<typeof supabaseClient.auth.signInWithPassword>
    >)

    renderProvider()

    await userEvent.click(screen.getByRole('button', { name: 'login' }))

    expect(screen.getByRole('alert')).toHaveTextContent('credenciales inválidas')
  })

  it('logs out through Supabase', async () => {
    signOutMock.mockResolvedValue({ error: null })

    renderProvider()

    await userEvent.click(screen.getByRole('button', { name: 'logout' }))

    expect(signOutMock).toHaveBeenCalledOnce()
  })

  it('surfaces a logout error', async () => {
    signOutMock.mockResolvedValue({ error: new Error('sesión ya cerrada') } as Awaited<
      ReturnType<typeof supabaseClient.auth.signOut>
    >)

    renderProvider()

    await userEvent.click(screen.getByRole('button', { name: 'logout' }))

    expect(screen.getByRole('alert')).toHaveTextContent('sesión ya cerrada')
  })

  it('unsubscribes from Supabase auth state changes on unmount', () => {
    const { unmount } = renderProvider()

    unmount()

    expect(unsubscribeMock).toHaveBeenCalledOnce()
  })
})
