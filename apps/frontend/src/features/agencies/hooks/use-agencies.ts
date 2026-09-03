import { useCallback, useEffect, useState } from 'react'
import { messageFromError } from '../../../shared/api/client'
import { createAgency, deleteAgency, listAgencies, updateAgency } from '../api/agencies'
import type { Agency, AgencyFormValues } from '../types'

export type UseAgencies = {
  agencies: Agency[]
  isLoading: boolean
  isSubmitting: boolean
  error: string | null
  create: (values: AgencyFormValues) => Promise<boolean>
  update: (id: string, values: AgencyFormValues) => Promise<boolean>
  remove: (id: string) => Promise<boolean>
}

export function useAgencies(token: string): UseAgencies {
  const [agencies, setAgencies] = useState<Agency[]>([])
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
        setAgencies(loaded)
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
    (values: AgencyFormValues) =>
      run(async () => {
        const created = await createAgency(token, values)
        setAgencies((current) => [...current, created])
      }),
    [run, token],
  )

  const update = useCallback(
    (id: string, values: AgencyFormValues) =>
      run(async () => {
        const updated = await updateAgency(token, id, values)
        setAgencies((current) =>
          current.map((agency) => (agency.id === id ? updated : agency)),
        )
      }),
    [run, token],
  )

  const remove = useCallback(
    (id: string) =>
      run(async () => {
        await deleteAgency(token, id)
        setAgencies((current) =>
          current.filter((agency) => agency.id !== id),
        )
      }),
    [run, token],
  )

  return { agencies, isLoading, isSubmitting, error, create, update, remove }
}
