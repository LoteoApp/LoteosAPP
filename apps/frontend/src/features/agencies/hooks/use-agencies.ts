import { useCallback, useEffect, useState } from 'react'
import { messageFromError } from '../../../shared/api/client'
import { createAgency, deleteAgency, listAgencies, updateAgency } from '../api/agencies'
import type { Inmobiliaria, InmobiliariaFormValues } from '../types'

export type UseAgencies = {
  inmobiliarias: Inmobiliaria[]
  isLoading: boolean
  isSubmitting: boolean
  error: string | null
  create: (values: InmobiliariaFormValues) => Promise<boolean>
  update: (id: string, values: InmobiliariaFormValues) => Promise<boolean>
  remove: (id: string) => Promise<boolean>
}

export function useAgencies(token: string): UseAgencies {
  const [inmobiliarias, setInmobiliarias] = useState<Inmobiliaria[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    listAgencies(token, controller.signal)
      .then((loaded) => {
        if (controller.signal.aborted) {
          return
        }
        setInmobiliarias(loaded)
        setError(null)
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setError(messageFromError(loadError))
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
      setError(messageFromError(operationError))
      return false
    } finally {
      setIsSubmitting(false)
    }
  }, [])

  const create = useCallback(
    (values: InmobiliariaFormValues) =>
      run(async () => {
        const created = await createAgency(token, values)
        setInmobiliarias((current) => [...current, created])
      }),
    [run, token],
  )

  const update = useCallback(
    (id: string, values: InmobiliariaFormValues) =>
      run(async () => {
        const updated = await updateAgency(token, id, values)
        setInmobiliarias((current) =>
          current.map((inmobiliaria) => (inmobiliaria.id === id ? updated : inmobiliaria)),
        )
      }),
    [run, token],
  )

  const remove = useCallback(
    (id: string) =>
      run(async () => {
        await deleteAgency(token, id)
        setInmobiliarias((current) =>
          current.filter((inmobiliaria) => inmobiliaria.id !== id),
        )
      }),
    [run, token],
  )

  return { inmobiliarias, isLoading, isSubmitting, error, create, update, remove }
}
