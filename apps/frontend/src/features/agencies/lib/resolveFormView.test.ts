import { describe, expect, it } from 'vitest'
import { resolveFormView } from './resolveFormView'
import type { Inmobiliaria } from '../types'

const agency: Inmobiliaria = {
  id: 'inmobiliaria-1',
  razonSocial: 'Lotes del Sur',
  cuit: '30712345678',
  telefono: '3415551234',
  email: 'contacto@lotesdelsur.com',
}

describe('resolveFormView', () => {
  it('keeps the closed and create views as they are', () => {
    expect(resolveFormView({ mode: 'closed' }, [agency])).toEqual({ mode: 'closed' })
    expect(resolveFormView({ mode: 'create' }, [agency])).toEqual({ mode: 'create' })
  })

  it('resolves an edit state to the agency it points at', () => {
    expect(resolveFormView({ mode: 'edit', id: 'inmobiliaria-1' }, [agency])).toEqual({
      mode: 'edit',
      agency,
    })
  })

  it('falls back to the closed view when the agency is gone', () => {
    expect(resolveFormView({ mode: 'edit', id: 'inmobiliaria-1' }, [])).toEqual({
      mode: 'closed',
    })
  })
})
