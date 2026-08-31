import { describe, expect, it } from 'vitest'
import { resolveFormView } from './resolveFormView'
import type { Surveyor } from '../types'

const surveyor: Surveyor = {
  id: 'agri-1',
  nombre: 'Ana',
  apellido: 'Gómez',
  email: 'ana@example.com',
  fechaBaja: null,
}

describe('resolveFormView', () => {
  it('passes the closed and create states through', () => {
    expect(resolveFormView({ mode: 'closed' }, [])).toEqual({ mode: 'closed' })
    expect(resolveFormView({ mode: 'create' }, [])).toEqual({ mode: 'create' })
  })

  it('resolves an edit state to the surveyor being edited', () => {
    expect(resolveFormView({ mode: 'edit', id: 'agri-1' }, [surveyor])).toEqual({
      mode: 'edit',
      surveyor,
    })
  })

  it('falls back to the closed view when the surveyor is gone', () => {
    expect(resolveFormView({ mode: 'edit', id: 'agri-1' }, [])).toEqual({ mode: 'closed' })
  })
})
