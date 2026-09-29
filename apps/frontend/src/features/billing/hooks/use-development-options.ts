import { useEffect, useState } from 'react'
import { listDevelopmentOptions } from '../api/billing'
import type { DevelopmentOption } from '../types'

// A failure leaves the list empty: the filter just offers "Todos los loteos",
// and the vencimientos themselves still load.
export function useDevelopmentOptions(token: string): DevelopmentOption[] {
  const [options, setOptions] = useState<DevelopmentOption[]>([])

  useEffect(() => {
    if (token === '') {
      return
    }
    const controller = new AbortController()
    listDevelopmentOptions(token, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setOptions(loaded)
      })
      .catch(() => {
        if (!controller.signal.aborted) setOptions([])
      })
    return () => controller.abort()
  }, [token])

  return token === '' ? [] : options
}
