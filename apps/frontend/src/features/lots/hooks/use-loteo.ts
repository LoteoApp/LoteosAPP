import { useEffect, useState } from 'react'
import { ApiError } from '../../../shared/api/client'
import { getLoteo } from '../api/get-loteo'
import type { LoteoDetail } from '../types'

const SESSION_EXPIRED_MESSAGE =
  'Tu sesión expiró. Volvé a iniciar sesión y probá de nuevo.'

function isNotFound(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    (error.status === 404 || error.code === 'loteo_not_found')
  )
}

// Both ApiError (backend message) and the plain Error the api client throws for
// an unexpected shape carry a user-facing message; keep it.
function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export type UseLoteo =
  | { status: 'loading' }
  | { status: 'loaded'; loteo: LoteoDetail }
  | { status: 'not-found' }
  | { status: 'error'; message: string }

function pending(token: string): UseLoteo {
  return token === ''
    ? { status: 'error', message: SESSION_EXPIRED_MESSAGE }
    : { status: 'loading' }
}

export function useLoteo(loteoId: string, token: string): UseLoteo {
  const [state, setState] = useState<UseLoteo>(() => pending(token))

  // A new id or token starts a fresh load: show the spinner (or the expired
  // session error) now, during render, so the UI never presents the previous
  // loteo as the current one. Same "reset state on prop change" pattern as
  // useLoteos.
  const requestKey = JSON.stringify([token, loteoId])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setState(pending(token))
  }

  useEffect(() => {
    if (token === '') {
      return
    }

    const controller = new AbortController()

    getLoteo(loteoId, token, { signal: controller.signal })
      .then((loteo) => {
        if (!controller.signal.aborted) {
          setState({ status: 'loaded', loteo })
        }
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) {
          return
        }
        if (isNotFound(error)) {
          setState({ status: 'not-found' })
          return
        }
        setState({ status: 'error', message: messageOf(error) })
      })

    return () => {
      controller.abort()
    }
  }, [loteoId, token])

  return state
}
