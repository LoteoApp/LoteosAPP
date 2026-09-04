import { describe, expect, it } from 'vitest'
import { resolveFormView } from './resolveFormView'
import type { Usuario } from '../types'

const usuario: Usuario = {
  id: '1',
  email: 'ana@example.com',
  nombre: 'Ana',
  apellido: 'Pérez',
  rol: 'administrativo',
  perfilCompleto: true,
  fechaBaja: null,
  createdAt: '2026-01-01T00:00:00Z',
}

describe('resolveFormView', () => {
  it('keeps the closed state as-is', () => {
    expect(resolveFormView({ mode: 'closed' }, [])).toEqual({ mode: 'closed' })
  })

  it('keeps the create state as-is', () => {
    expect(resolveFormView({ mode: 'create' }, [usuario])).toEqual({ mode: 'create' })
  })

  it('resolves an edit state to its matching user', () => {
    expect(resolveFormView({ mode: 'edit', id: '1' }, [usuario])).toEqual({
      mode: 'edit',
      usuario,
    })
  })

  it('falls back to closed when the user being edited no longer exists', () => {
    expect(resolveFormView({ mode: 'edit', id: 'missing' }, [usuario])).toEqual({
      mode: 'closed',
    })
  })

  it('falls back to closed when the user being edited disappears from an emptied list', () => {
    expect(resolveFormView({ mode: 'edit', id: '1' }, [])).toEqual({ mode: 'closed' })
  })
})
