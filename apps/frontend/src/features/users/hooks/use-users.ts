import { useCallback, useEffect, useState } from 'react'
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

  const deactivate = useCallback(
    (id: string) =>
      run(async () => {
        await deactivateUser(token, id)
        setUsuarios((current) =>
          current.map((usuario) =>
            usuario.id === id ? { ...usuario, fechaBaja: new Date().toISOString() } : usuario,
          ),
        )
      }),
    [run, token],
  )

  const reactivate = useCallback(
    (id: string) =>
      run(async () => {
        const updated = await reactivateUser(token, id)
        setUsuarios((current) => current.map((usuario) => (usuario.id === id ? updated : usuario)))
      }),
    [run, token],
  )

  return { usuarios, isLoading, isSubmitting, error, create, update, deactivate, reactivate }
}
