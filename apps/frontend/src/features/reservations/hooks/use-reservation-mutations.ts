import { useCallback, useState } from 'react'
import { cancelReservation, createReservation } from '../api/reservations'
import type { Reservation } from '../types'

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

type CreateValues = {
  loteoId: string
  loteId: string
  clienteId: string
  vendedorId?: string
}

export function useReservationMutations(token: string) {
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const create = useCallback(async (values: CreateValues, idempotencyKey: string): Promise<Reservation | null> => {
    setIsSubmitting(true)
    try {
      const reservation = await createReservation(token, values, idempotencyKey)
      setError(null)
      return reservation
    } catch (createError) {
      setError(messageOf(createError))
      return null
    } finally {
      setIsSubmitting(false)
    }
  }, [token])

  const cancel = useCallback(async (id: string, reason: string): Promise<Reservation | null> => {
    setIsSubmitting(true)
    try {
      const reservation = await cancelReservation(token, id, reason)
      setError(null)
      return reservation
    } catch (cancelError) {
      setError(messageOf(cancelError))
      return null
    } finally {
      setIsSubmitting(false)
    }
  }, [token])

  const reset = useCallback(() => setError(null), [])
  return { create, cancel, isSubmitting, error, reset }
}
