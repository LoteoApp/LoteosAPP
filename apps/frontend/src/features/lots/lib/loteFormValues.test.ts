import { describe, expect, it } from 'vitest'
import {
  currencyOptions,
  formatPrecioInput,
  maskPrecioInput,
  maskPrecioTyping,
  reformatPrecioInput,
  superficieFromPolygon,
  toLoteFormValues,
  validateLoteForm,
  type LoteFormValues,
} from './loteFormValues'
import type { LoteoLote } from '../types'

function lote(overrides: Partial<LoteoLote> = {}): LoteoLote {
  return {
    id: 'lt-1',
    manzanaId: 'mz-1',
    numero: '12',
    precio: 4500000.5,
    moneda: 'ARS',
    superficie: 320.75,
    caracteristicas: 'Esquina',
    poligono: [],
    ...overrides,
  }
}

function values(overrides: Partial<LoteFormValues> = {}): LoteFormValues {
  return {
    numero: '12',
    precio: '4500000.5',
    moneda: 'ARS',
    superficie: '320.75',
    caracteristicas: 'Esquina',
    ...overrides,
  }
}

describe('toLoteFormValues', () => {
  it('stringifies numbers and turns nulls into empty strings', () => {
    expect(toLoteFormValues(lote())).toEqual({
      numero: '12',
      precio: '4.500.000,5',
      moneda: 'ARS',
      superficie: '320.75',
      caracteristicas: 'Esquina',
    })
    expect(
      toLoteFormValues(lote({ precio: null, superficie: null, moneda: '', caracteristicas: '' })),
    ).toEqual({
      numero: '12',
      precio: '',
      moneda: '',
      superficie: '',
      caracteristicas: '',
    })
  })

  it('preloads superficie from the polygon when the saved value is null', () => {
    expect(
      toLoteFormValues(
        lote({
          superficie: null,
          poligono: [
            { x: 0, y: 0 },
            { x: 10, y: 0 },
            { x: 10, y: 10 },
            { x: 0, y: 10 },
          ],
        }),
      ).superficie,
    ).toBe('100')
  })

  it('keeps a saved superficie instead of replacing it with the polygon area', () => {
    expect(
      toLoteFormValues(
        lote({
          superficie: 300,
          poligono: [
            { x: 0, y: 0 },
            { x: 10, y: 0 },
            { x: 10, y: 10 },
            { x: 0, y: 10 },
          ],
        }),
      ).superficie,
    ).toBe('300')
  })
})

describe('formatPrecioInput', () => {
  it('groups thousands with a dot and decimals with a comma', () => {
    expect(formatPrecioInput(150000)).toBe('150.000')
    expect(formatPrecioInput(4500000.5)).toBe('4.500.000,5')
    expect(formatPrecioInput(10.5)).toBe('10,5')
    expect(formatPrecioInput(0)).toBe('0')
  })
})

describe('reformatPrecioInput', () => {
  it('turns a plain number into a grouped amount', () => {
    expect(reformatPrecioInput('200000')).toBe('200.000')
    expect(reformatPrecioInput('4.500.000,50')).toBe('4.500.000,5')
    expect(reformatPrecioInput('')).toBe('')
    expect(reformatPrecioInput('abc')).toBe('abc')
  })
})

describe('maskPrecioInput', () => {
  it('groups thousands as the digits are typed', () => {
    expect(maskPrecioInput('')).toBe('')
    expect(maskPrecioInput('1')).toBe('1')
    expect(maskPrecioInput('15')).toBe('15')
    expect(maskPrecioInput('150')).toBe('150')
    expect(maskPrecioInput('1500')).toBe('1.500')
    expect(maskPrecioInput('15000')).toBe('15.000')
    expect(maskPrecioInput('150000')).toBe('150.000')
    expect(maskPrecioInput('4500000')).toBe('4.500.000')
  })

  it('keeps a trailing comma so decimals can be typed', () => {
    expect(maskPrecioInput('150000,')).toBe('150.000,')
    expect(maskPrecioInput('150000,5')).toBe('150.000,5')
    expect(maskPrecioInput('150000,50')).toBe('150.000,50')
    expect(maskPrecioInput('150000,501')).toBe('150.000,50')
    expect(maskPrecioInput('15.')).toBe('15,')
  })

  it('strips junk, leading zeros and extra integer digits', () => {
    expect(maskPrecioInput('abc')).toBe('')
    expect(maskPrecioInput('00')).toBe('0')
    expect(maskPrecioInput('0,5')).toBe('0,5')
    expect(maskPrecioInput('4.500.000,5')).toBe('4.500.000,5')
    expect(maskPrecioInput('1234567890123')).toBe('123.456.789.012')
  })
})

describe('maskPrecioTyping', () => {
  it('keeps the cursor on the same digit after grouping', () => {
    expect(maskPrecioTyping('1500', 4)).toEqual({ value: '1.500', cursor: 5 })
    expect(maskPrecioTyping('15.', 3)).toEqual({ value: '15,', cursor: 3 })
    expect(maskPrecioTyping('', 0)).toEqual({ value: '', cursor: 0 })
  })
})

describe('superficieFromPolygon', () => {
  it('formats the Gauss area with up to 4 decimals', () => {
    expect(
      superficieFromPolygon([
        { x: 0, y: 0 },
        { x: 10, y: 0 },
        { x: 10, y: 10 },
        { x: 0, y: 10 },
      ]),
    ).toBe('100')
    expect(
      superficieFromPolygon([
        { x: 0, y: 0 },
        { x: 1.5, y: 0 },
        { x: 1.5, y: 1.25 },
        { x: 0, y: 1.25 },
      ]),
    ).toBe('1.875')
    expect(superficieFromPolygon([])).toBeNull()
  })
})

describe('currencyOptions', () => {
  it('keeps ARS and USD, and adds a saved currency that is neither', () => {
    expect(currencyOptions('ARS')).toEqual(['ARS', 'USD'])
    expect(currencyOptions('')).toEqual(['ARS', 'USD'])
    expect(currencyOptions('EUR')).toEqual(['ARS', 'USD', 'EUR'])
  })
})

describe('validateLoteForm', () => {
  it('accepts a complete payload and trims text fields', () => {
    const result = validateLoteForm(
      values({ numero: ' 12 ', caracteristicas: '  Esquina  ', moneda: 'ars' }),
    )

    expect(result).toEqual({
      ok: true,
      payload: {
        numero: '12',
        precio: 4500000.5,
        moneda: 'ARS',
        superficie: 320.75,
        caracteristicas: 'Esquina',
      },
    })
  })

  it('sends null for empty precio and superficie', () => {
    const result = validateLoteForm(values({ precio: '', superficie: '  ', moneda: '' }))

    expect(result).toEqual({
      ok: true,
      payload: {
        numero: '12',
        precio: null,
        moneda: '',
        superficie: null,
        caracteristicas: 'Esquina',
      },
    })
  })

  it('accepts a comma as the decimal separator', () => {
    const result = validateLoteForm(values({ precio: '10,5', superficie: '1,25' }))

    expect(result.ok).toBe(true)
    if (result.ok) {
      expect(result.payload.precio).toBe(10.5)
      expect(result.payload.superficie).toBe(1.25)
    }
  })

  it('accepts a precio with dot thousand separators', () => {
    const grouped = validateLoteForm(values({ precio: '4.500.000,5' }))
    expect(grouped.ok).toBe(true)
    if (grouped.ok) {
      expect(grouped.payload.precio).toBe(4500000.5)
    }

    const thousandsOnly = validateLoteForm(values({ precio: '1.500' }))
    expect(thousandsOnly.ok).toBe(true)
    if (thousandsOnly.ok) {
      expect(thousandsOnly.payload.precio).toBe(1500)
    }
  })

  it('rejects a missing numero and one longer than 32 characters', () => {
    expect(validateLoteForm(values({ numero: '   ' })).ok).toBe(false)
    expect(validateLoteForm(values({ numero: 'a'.repeat(32) })).ok).toBe(true)
    const tooLong = validateLoteForm(values({ numero: 'a'.repeat(33) }))
    expect(tooLong.ok).toBe(false)
    if (!tooLong.ok) {
      expect(tooLong.errors.numero).toMatch(/32/)
    }
  })

  it('rejects a precio with more than 2 decimals, a negative one, or one past the cap', () => {
    expect(validateLoteForm(values({ precio: '10.12' })).ok).toBe(true)
    expect(validateLoteForm(values({ precio: '10,123' })).ok).toBe(false)
    expect(validateLoteForm(values({ precio: '-1' })).ok).toBe(false)
    expect(validateLoteForm(values({ precio: '1000000000000' })).ok).toBe(false)
    expect(validateLoteForm(values({ precio: 'abc' })).ok).toBe(false)
    expect(validateLoteForm(values({ precio: '0' })).ok).toBe(true)
  })

  it('rejects a precio without a moneda', () => {
    const result = validateLoteForm(values({ moneda: '' }))
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.moneda).toMatch(/moneda/i)
    }
  })

  it('rejects a moneda without a precio', () => {
    const result = validateLoteForm(values({ precio: '', moneda: 'ARS' }))
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors.moneda).toMatch(/junto con un precio/i)
    }
  })

  it('rejects a moneda that is not three letters', () => {
    expect(validateLoteForm(values({ moneda: 'AR' })).ok).toBe(false)
    expect(validateLoteForm(values({ moneda: 'ARS1' })).ok).toBe(false)
    expect(validateLoteForm(values({ moneda: 'AR1' })).ok).toBe(false)
  })

  it('rejects a superficie that is not strictly positive, has 5 decimals, or is past the cap', () => {
    expect(validateLoteForm(values({ superficie: '1.2345' })).ok).toBe(true)
    expect(validateLoteForm(values({ superficie: '1.23456' })).ok).toBe(false)
    expect(validateLoteForm(values({ superficie: '0' })).ok).toBe(false)
    expect(validateLoteForm(values({ superficie: '-1' })).ok).toBe(false)
    expect(validateLoteForm(values({ superficie: '100000000' })).ok).toBe(false)
    expect(validateLoteForm(values({ superficie: 'nope' })).ok).toBe(false)
  })

  it('rejects caracteristicas longer than 2000 characters', () => {
    expect(validateLoteForm(values({ caracteristicas: 'a'.repeat(2000) })).ok).toBe(true)
    const tooLong = validateLoteForm(values({ caracteristicas: 'a'.repeat(2001) }))
    expect(tooLong.ok).toBe(false)
    if (!tooLong.ok) {
      expect(tooLong.errors.caracteristicas).toMatch(/2000/)
    }
  })
})
