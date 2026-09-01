import { useCallback, useEffect, useState } from 'react'
import { createClient, deleteClient, listClients, updateClient } from '../api/clients'
import type { Cliente, ClienteFormValues } from '../types'

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export type UseClients = {
  clientes: Cliente[]
  isLoading: boolean
  isSubmitting: boolean
  error: string | null
  create: (values: ClienteFormValues) => Promise<boolean>
  update: (id: string, values: ClienteFormValues) => Promise<boolean>
  remove: (id: string) => Promise<boolean>
}

export function useClients(token: string): UseClients {
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    listClients(token, controller.signal)
      .then((loaded) => {
        if (controller.signal.aborted) {
          return
        }
        setClientes(loaded)
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
    (values: ClienteFormValues) =>
      run(async () => {
        const created = await createClient(token, values)
        setClientes((current) => [...current, created])
      }),
    [run, token],
  )

  const update = useCallback(
    (id: string, values: ClienteFormValues) =>
      run(async () => {
        const updated = await updateClient(token, id, values)
        setClientes((current) =>
          current.map((cliente) => (cliente.id === id ? updated : cliente)),
        )
      }),
    [run, token],
  )

  const remove = useCallback(
    (id: string) =>
      run(async () => {
        await deleteClient(token, id)
        setClientes((current) => current.filter((cliente) => cliente.id !== id))
      }),
    [run, token],
  )

  return { clientes, isLoading, isSubmitting, error, create, update, remove }
}
