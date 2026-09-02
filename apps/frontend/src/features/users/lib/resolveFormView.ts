import type { Usuario } from '../types'

export type FormState = { mode: 'closed' } | { mode: 'create' } | { mode: 'edit'; id: string }

export type FormView =
  | { mode: 'closed' }
  | { mode: 'create' }
  | { mode: 'edit'; usuario: Usuario }

// If the user behind an "edit" FormState is no longer in the list (e.g. an
// external refresh removes it), fall back to the closed view instead of
// rendering nothing while the list underneath also stays hidden.
export function resolveFormView(formState: FormState, usuarios: Usuario[]): FormView {
  if (formState.mode !== 'edit') {
    return formState
  }

  const usuario = usuarios.find((candidate) => candidate.id === formState.id)
  return usuario ? { mode: 'edit', usuario } : { mode: 'closed' }
}
