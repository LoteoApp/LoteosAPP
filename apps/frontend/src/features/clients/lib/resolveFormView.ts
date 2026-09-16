import type { Cliente } from '../types'

export type FormState = { mode: 'closed' } | { mode: 'create' } | { mode: 'edit'; id: string }

export type FormView =
  | { mode: 'closed' }
  | { mode: 'create' }
  | { mode: 'edit'; client: Cliente }

// If the client behind an "edit" FormState is no longer in the list (e.g. an
// external refresh removes it), fall back to the closed view instead of
// rendering nothing while the list underneath also stays hidden.
export function resolveFormView(formState: FormState, clientes: Cliente[]): FormView {
  if (formState.mode !== 'edit') {
    return formState
  }

  const client = clientes.find((cliente) => cliente.id === formState.id)
  return client ? { mode: 'edit', client } : { mode: 'closed' }
}
