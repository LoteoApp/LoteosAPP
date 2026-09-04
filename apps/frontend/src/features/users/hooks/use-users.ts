import { useCallback, useEffect, useState } from 'react'
import { ApiError } from '../../../shared/api/client'
import { createUser, deactivateUser, listUsers, reactivateUser, updateUser } from '../api/users'
import type { Usuario, UsuarioFormValues, UsuarioUpdateValues } from '../types'

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export type UseUsers = {
  usuarios: Usuario[]
  isLoading: boolean
  isSubmitting: boolean
  error: string | null
  create: (values: UsuarioFormValues) => Promise<string | null>
  update: (id: string, values: UsuarioUpdateValues) => Promise<boolean>
  deactivate: (id: string) => Promise<boolean>
  reactivate: (id: string) => Promise<boolean>
}

export function useUsers(token: string): UseUsers {
  const [usuarios, setUsuarios] = useState<Usuario[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    listUsers(token, controller.signal)
      .then((loaded) => {
        if (controller.signal.aborted) {
          return
        }
        setUsuarios(loaded)
        setError(null)
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setError(messageOf(loadError))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setIsLoading(false)
        }
      })

    return () => {
      controller.abort()
    }
  }, [token])

  const run = useCallback(async (operation: () => Promise<void>): Promise<boolean> => {
    setIsSubmitting(true)
    try {
      await operation()
      setError(null)
      return true
    } catch (operationError) {
      setError(messageOf(operationError))
      return false
    } finally {
      setIsSubmitting(false)
    }
  }, [])

  const create = useCallback(
    async (values: UsuarioFormValues): Promise<string | null> => {
      setIsSubmitting(true)
      try {
        const { usuario, temporaryPassword } = await createUser(token, values)
        setUsuarios((current) => [...current, usuario])
        setError(null)
        return temporaryPassword
      } catch (createError) {
        setError(messageOf(createError))
        return null
      } finally {
        setIsSubmitting(false)
      }
    },
    [token],
  )

  const update = useCallback(
    (id: string, values: UsuarioUpdateValues) =>
      run(async () => {
        const updated = await updateUser(token, id, values)
        setUsuarios((current) => current.map((usuario) => (usuario.id === id ? updated : usuario)))
      }),
    [run, token],
  )

  // A retried baja/reactivación can land on the backend's own already-done
  // conflict (e.g. the first attempt's response was lost, not its effect):
  // that's reconciled into the same local state a successful call would
  // have produced, instead of surfacing it as an error and leaving the UI
  // showing a stale status.
  const deactivate = useCallback(
    (id: string) =>
      run(async () => {
        try {
          await deactivateUser(token, id)
        } catch (deactivateError) {
          if (!(deactivateError instanceof ApiError) || deactivateError.code !== 'user_already_inactive') {
            throw deactivateError
          }
        }
        setUsuarios((current) =>
          current.map((usuario) =>
            usuario.id === id ? { ...usuario, fechaBaja: usuario.fechaBaja ?? new Date().toISOString() } : usuario,
          ),
        )
      }),
    [run, token],
  )

  const reactivate = useCallback(
    (id: string) =>
      run(async () => {
        try {
          const updated = await reactivateUser(token, id)
          setUsuarios((current) => current.map((usuario) => (usuario.id === id ? updated : usuario)))
        } catch (reactivateError) {
          if (!(reactivateError instanceof ApiError) || reactivateError.code !== 'user_already_active') {
            throw reactivateError
          }
          setUsuarios((current) =>
            current.map((usuario) => (usuario.id === id ? { ...usuario, fechaBaja: null } : usuario)),
          )
        }
      }),
    [run, token],
  )

  return { usuarios, isLoading, isSubmitting, error, create, update, deactivate, reactivate }
}
