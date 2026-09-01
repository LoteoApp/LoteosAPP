import { describe, expect, it } from 'vitest'
import { resolveFormView } from './resolveFormView'
import type { Cliente } from '../types'

const cliente: Cliente = {
  id: '1',
  nombre: 'Ana',
  apellido: 'Pérez',
  dni: '30111222',
  celular: '',
  email: '',
}

describe('resolveFormView', () => {
  it('keeps the closed state as-is', () => {
    expect(resolveFormView({ mode: 'closed' }, [])).toEqual({ mode: 'closed' })
  })

  it('keeps the create state as-is', () => {
    expect(resolveFormView({ mode: 'create' }, [cliente])).toEqual({ mode: 'create' })
  })

  it('resolves an edit state to its matching client', () => {
    expect(resolveFormView({ mode: 'edit', id: '1' }, [cliente])).toEqual({
      mode: 'edit',
      client: cliente,
    })
  })

  it('falls back to closed when the client being edited no longer exists', () => {
    expect(resolveFormView({ mode: 'edit', id: 'missing' }, [cliente])).toEqual({
      mode: 'closed',
    })
  })

  it('falls back to closed when the client being edited disappears from an emptied list', () => {
    expect(resolveFormView({ mode: 'edit', id: '1' }, [])).toEqual({ mode: 'closed' })
  })
})
