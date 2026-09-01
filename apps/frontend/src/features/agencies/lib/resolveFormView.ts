import type { Inmobiliaria } from '../types'

export type FormState = { mode: 'closed' } | { mode: 'create' } | { mode: 'edit'; id: string }

export type FormView =
  | { mode: 'closed' }
  | { mode: 'create' }
  | { mode: 'edit'; agency: Inmobiliaria }

// If the agency behind an "edit" FormState is no longer in the list (e.g. an
// external refresh removes it), fall back to the closed view instead of
// rendering nothing while the list underneath also stays hidden.
export function resolveFormView(
  formState: FormState,
  inmobiliarias: Inmobiliaria[],
): FormView {
  if (formState.mode !== 'edit') {
    return formState
  }

  const agency = inmobiliarias.find((inmobiliaria) => inmobiliaria.id === formState.id)
  return agency ? { mode: 'edit', agency } : { mode: 'closed' }
}
