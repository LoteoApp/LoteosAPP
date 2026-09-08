import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../../../shared/api/client'

export type UpdateResourceState<Field extends string> =
  | { status: 'idle' }
  | { status: 'saving' }
  | { status: 'error'; message: string; field?: Field }

type StoredUpdateResourceState<Field extends string> = {
  value: UpdateResourceState<Field>
  token: string | null
}

type UpdateResourceRequest<Entity, Payload> = (
  loteoId: string,
  resourceId: string,
  payload: Payload,
  token: string,
) => Promise<Entity>

type UseUpdateResourceResult<Entity, Payload, Field extends string> =
  UpdateResourceState<Field> & {
    update: (
      loteoId: string,
      resourceId: string,
      payload: Payload,
    ) => Promise<Entity | null>
    reset: () => void
  }

const SESSION_EXPIRED_MESSAGE =
  'Tu sesión expiró. Volvé a iniciar sesión y probá de nuevo.'
const UNEXPECTED_ERROR_MESSAGE = 'Ocurrió un error inesperado.'

export function useUpdateResource<Entity, Payload, Field extends string>(
  accessToken: string | null,
  request: UpdateResourceRequest<Entity, Payload>,
  fieldForCode: (code: string) => Field | undefined,
): UseUpdateResourceResult<Entity, Payload, Field> {
  const [storedState, setStoredState] = useState<StoredUpdateResourceState<Field>>({
    value: { status: 'idle' },
    token: accessToken,
  })
  const operationRef = useRef(0)
  const accessTokenRef = useRef(accessToken)

  useEffect(() => {
    accessTokenRef.current = accessToken
    operationRef.current += 1
  }, [accessToken])

  useEffect(() => {
    return () => {
      operationRef.current += 1
    }
  }, [])

  const reset = useCallback(() => {
    ++operationRef.current
    setStoredState({ value: { status: 'idle' }, token: accessToken })
  }, [accessToken])

  const update = useCallback(
    async (loteoId: string, resourceId: string, payload: Payload) => {
      const operation = ++operationRef.current
      const isCurrent = () =>
        operation === operationRef.current && accessTokenRef.current === accessToken

      if (!accessToken) {
        setStoredState({
          value: { status: 'error', message: SESSION_EXPIRED_MESSAGE },
          token: accessToken,
        })
        return null
      }

      setStoredState({ value: { status: 'saving' }, token: accessToken })

      try {
        const entity = await request(loteoId, resourceId, payload, accessToken)
        if (!isCurrent()) {
          return null
        }
        setStoredState({ value: { status: 'idle' }, token: accessToken })
        return entity
      } catch (error) {
        if (!isCurrent()) {
          return null
        }
        if (error instanceof ApiError) {
          setStoredState({
            value: {
              status: 'error',
              message: error.message,
              field: fieldForCode(error.code),
            },
            token: accessToken,
          })
          return null
        }
        setStoredState({
          value: {
            status: 'error',
            message: error instanceof Error ? error.message : UNEXPECTED_ERROR_MESSAGE,
          },
          token: accessToken,
        })
        return null
      }
    },
    [accessToken, fieldForCode, request],
  )

  const state =
    storedState.token === accessToken
      ? storedState.value
      : ({ status: 'idle' } satisfies UpdateResourceState<Field>)

  return { ...state, update, reset }
}
