import { useEffect, useState } from 'react'
import { getSystemInfo } from '../api/get-system-info'
import type { SystemInfo } from '../types'

export type SystemInfoState =
  | { status: 'loading' }
  | { status: 'success'; data: SystemInfo }
  | { status: 'error'; message: string }

export function useSystemInfo(): SystemInfoState {
  const [state, setState] = useState<SystemInfoState>({ status: 'loading' })

  useEffect(() => {
    const controller = new AbortController()

    async function loadSystemInfo() {
      try {
        const data = await getSystemInfo(controller.signal)
        if (!controller.signal.aborted) {
          setState({ status: 'success', data })
        }
      } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
          return
        }

        setState({
          status: 'error',
          message: error instanceof Error ? error.message : 'Error desconocido',
        })
      }
    }

    void loadSystemInfo()

    return () => controller.abort()
  }, [])

  return state
}
