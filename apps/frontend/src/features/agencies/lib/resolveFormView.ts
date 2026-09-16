import type { Agency } from '../types'

export type FormState = { mode: 'closed' } | { mode: 'create' } | { mode: 'edit'; id: string }

export type FormView =
  | { mode: 'closed' }
  | { mode: 'create' }
  | { mode: 'edit'; agency: Agency }

// If the agency behind an "edit" FormState is no longer in the list (e.g. an
// external refresh removes it), fall back to the closed view instead of
// rendering nothing while the list underneath also stays hidden.
export function resolveFormView(formState: FormState, agencies: Agency[]): FormView {
  if (formState.mode !== 'edit') {
    return formState
  }

  const agency = agencies.find((candidate) => candidate.id === formState.id)
  return agency ? { mode: 'edit', agency } : { mode: 'closed' }
}
