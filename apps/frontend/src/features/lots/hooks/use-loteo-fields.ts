import { useCallback, useState } from 'react'
import type { LoteoFieldValues } from '../components/LoteoFields'

const EMPTY_FIELDS: LoteoFieldValues = {
  name: '',
  location: '',
  description: '',
}

export type UseLoteoFieldsResult = {
  values: LoteoFieldValues
  onChange: (values: LoteoFieldValues) => void
  reset: () => void
}

export function useLoteoFields(): UseLoteoFieldsResult {
  const [values, setValues] = useState<LoteoFieldValues>(EMPTY_FIELDS)

  const reset = useCallback(() => {
    setValues(EMPTY_FIELDS)
  }, [])

  return { values, onChange: setValues, reset }
}
