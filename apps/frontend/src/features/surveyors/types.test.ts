import { describe, expect, it } from 'vitest'
import { fullName, isActive, toSurveyorFormValues, type Surveyor } from './types'

const surveyor: Surveyor = {
  id: 'agri-1',
  nombre: 'Ana',
  apellido: 'Gómez',
  email: 'ana@example.com',
  fechaBaja: null,
}

describe('isActive', () => {
  it('treats a surveyor without fecha de baja as active', () => {
    expect(isActive(surveyor)).toBe(true)
  })

  it('treats a surveyor with fecha de baja as inactive', () => {
    expect(isActive({ ...surveyor, fechaBaja: '2026-03-01T12:00:00Z' })).toBe(false)
  })
})

describe('fullName', () => {
  it('joins nombre and apellido', () => {
    expect(fullName(surveyor)).toBe('Ana Gómez')
  })

  it('does not leave a trailing space when the apellido is missing', () => {
    expect(fullName({ ...surveyor, apellido: '' })).toBe('Ana')
  })
})

describe('toSurveyorFormValues', () => {
  it('keeps only the editable fields', () => {
    expect(toSurveyorFormValues(surveyor)).toEqual({
      nombre: 'Ana',
      apellido: 'Gómez',
      email: 'ana@example.com',
    })
  })
})
