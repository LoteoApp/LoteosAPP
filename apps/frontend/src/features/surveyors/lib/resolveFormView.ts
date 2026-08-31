import type { Surveyor } from '../types'

export type FormState = { mode: 'closed' } | { mode: 'create' } | { mode: 'edit'; id: string }

export type FormView =
  | { mode: 'closed' }
  | { mode: 'create' }
  | { mode: 'edit'; surveyor: Surveyor }

// If the agrimensor behind an "edit" FormState is no longer in the list (e.g.
// a refresh dropped it), fall back to the closed view instead of rendering
// nothing while the list underneath also stays hidden.
export function resolveFormView(formState: FormState, surveyors: Surveyor[]): FormView {
  if (formState.mode !== 'edit') {
    return formState
  }

  const surveyor = surveyors.find((candidate) => candidate.id === formState.id)
  return surveyor ? { mode: 'edit', surveyor } : { mode: 'closed' }
}
