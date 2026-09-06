import { describe, expect, it } from 'vitest'
import { clienteOptionLabel, isLotState, loteOptionLabel, type LoteOption } from './types'

function lote(overrides: Partial<LoteOption> = {}): LoteOption {
  return {
    id: 'lt-1',
    numero: '7',
    manzanaId: 'mz-1',
    manzanaNumero: '1',
    loteoId: 'loteo-1',
    loteoNombre: 'Norte',
    estado: 'disponible',
    precio: 150000,
    moneda: 'USD',
    superficie: 300,
    ...overrides,
  }
}

describe('isLotState', () => {
  it('accepts every state of the contract', () => {
    expect(isLotState('disponible')).toBe(true)
    expect(isLotState('reservado')).toBe(true)
    expect(isLotState('vendido')).toBe(true)
    expect(isLotState('finalizado')).toBe(true)
  })

  it('rejects anything else', () => {
    expect(isLotState('rifado')).toBe(false)
    expect(isLotState('')).toBe(false)
    expect(isLotState(7)).toBe(false)
    expect(isLotState(null)).toBe(false)
  })
})

describe('loteOptionLabel', () => {
  it('names a lote by its loteo, manzana and number', () => {
    expect(loteOptionLabel(lote())).toBe('Norte · Mz 1 · Lote 7')
  })

  it('falls back to a dash where the manzana or the lote has no number', () => {
    expect(loteOptionLabel(lote({ numero: '', manzanaNumero: '' }))).toBe('Norte · Mz — · Lote —')
  })
})

describe('clienteOptionLabel', () => {
  it('names a cliente by apellido, nombre and DNI', () => {
    expect(
      clienteOptionLabel({ id: 'cl-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }),
    ).toBe('Pérez, Ana · DNI 30111222')
  })
})
