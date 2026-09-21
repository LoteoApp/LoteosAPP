import { describe, expect, it } from 'vitest'
import {
  clientLabel,
  isDueFilter,
  isInstallmentState,
  isPaymentMedium,
  isPaymentType,
  lotLabel,
  paymentDateToISO,
  paymentItemsLabel,
  pendingInstallments,
  selectionAmount,
  todayInputValue,
  toggleInstallment,
  validatePaymentTerms,
  type Installment,
} from './types'

const cuotas: Installment[] = [
  { id: 'c-1', numero: 1, monto: 100, estado: 'pagada', fechaVencimiento: '2026-02-15T12:00:00Z', fechaPago: '2026-02-10T12:00:00Z' },
  { id: 'c-2', numero: 2, monto: 100, estado: 'vencida', fechaVencimiento: '2026-03-15T12:00:00Z' },
  { id: 'c-3', numero: 3, monto: 100.5, estado: 'pendiente', fechaVencimiento: '2026-04-15T12:00:00Z' },
  { id: 'c-4', numero: 4, monto: 100, estado: 'pendiente', fechaVencimiento: '2026-05-15T12:00:00Z' },
]

describe('type guards', () => {
  it('accept the contract values and reject the rest', () => {
    expect(isInstallmentState('vencida')).toBe(true)
    expect(isInstallmentState('cobrada')).toBe(false)
    expect(isPaymentMedium('cheque')).toBe(true)
    expect(isPaymentMedium(3)).toBe(false)
    expect(isPaymentType('cancelacion_total')).toBe(true)
    expect(isPaymentType('')).toBe(false)
    expect(isDueFilter('pendientes')).toBe(true)
    expect(isDueFilter('todas')).toBe(false)
  })
})

describe('paymentDateToISO', () => {
  it('sends the typed day as noon local time and rejects other shapes', () => {
    const iso = paymentDateToISO(' 2026-09-20 ')
    const date = new Date(iso)
    expect(date.getFullYear()).toBe(2026)
    expect(date.getMonth()).toBe(8)
    expect(date.getDate()).toBe(20)
    expect(date.getHours()).toBe(12)
    expect(paymentDateToISO('')).toBe('')
    expect(paymentDateToISO('20/09/2026')).toBe('')
  })

  it('formats today for the date input', () => {
    expect(todayInputValue(new Date(2026, 0, 5))).toBe('2026-01-05')
  })
})

describe('validatePaymentTerms', () => {
  const today = '2026-09-21'

  it('accepts valid terms and an empty date', () => {
    expect(validatePaymentTerms({ medioPago: 'efectivo', fechaPago: '', observacion: '' }, today)).toBeNull()
    expect(validatePaymentTerms({ medioPago: 'cheque', fechaPago: '2026-09-21', observacion: 'ok' }, today)).toBeNull()
  })

  it('explains what is wrong', () => {
    expect(validatePaymentTerms({ medioPago: 'canje' as never, fechaPago: '', observacion: '' }, today)).toBe('Elegí el medio de pago.')
    expect(validatePaymentTerms({ medioPago: 'efectivo', fechaPago: 'ayer', observacion: '' }, today)).toBe('Ingresá una fecha de pago válida.')
    expect(validatePaymentTerms({ medioPago: 'efectivo', fechaPago: '2026-09-22', observacion: '' }, today)).toBe(
      'La fecha de pago no puede ser futura.',
    )
    expect(validatePaymentTerms({ medioPago: 'efectivo', fechaPago: '', observacion: 'x'.repeat(501) }, today)).toBe(
      'La observación no puede superar los 500 caracteres.',
    )
  })
})

describe('installment selection', () => {
  it('lists the cuotas still owed in order', () => {
    expect(pendingInstallments(cuotas).map((cuota) => cuota.id)).toEqual(['c-2', 'c-3', 'c-4'])
  })

  it('selects every pending cuota up to the toggled one', () => {
    expect(toggleInstallment(cuotas, [], 'c-3')).toEqual(['c-2', 'c-3'])
    expect(toggleInstallment(cuotas, ['c-2', 'c-3'], 'c-4')).toEqual(['c-2', 'c-3', 'c-4'])
  })

  it('unselects from the toggled cuota onwards when it was already selected', () => {
    expect(toggleInstallment(cuotas, ['c-2', 'c-3', 'c-4'], 'c-3')).toEqual(['c-2'])
    expect(toggleInstallment(cuotas, ['c-2'], 'c-2')).toEqual([])
  })

  it('ignores cuotas that cannot be selected', () => {
    expect(toggleInstallment(cuotas, ['c-2'], 'c-1')).toEqual(['c-2'])
    expect(toggleInstallment(cuotas, ['c-2'], 'nope')).toEqual(['c-2'])
  })

  it('totals the selection with the entrega only while it is owed', () => {
    const statement = { cuotas, entrega: { monto: 50.25, estado: 'pendiente' as const } }
    expect(selectionAmount(statement, ['c-2', 'c-3'], true)).toBe(250.75)
    expect(selectionAmount(statement, ['c-2'], false)).toBe(100)
    expect(selectionAmount({ cuotas, entrega: { monto: 50, estado: 'pagada' } }, [], true)).toBe(0)
    expect(selectionAmount({ cuotas }, [], true)).toBe(0)
  })
})

describe('labels', () => {
  it('describe the lote, the client and what a cobro covered', () => {
    expect(lotLabel({ loteoNombre: 'Las Acacias', manzanaNumero: '2', loteNumero: '7' })).toBe('Las Acacias · Mz 2 · Lote 7')
    expect(lotLabel({ loteoNombre: 'Las Acacias', manzanaNumero: '', loteNumero: '' })).toBe('Las Acacias · Mz — · Lote —')
    expect(clientLabel({ nombre: 'Ana', apellido: 'Pérez' })).toBe('Pérez, Ana')
    expect(paymentItemsLabel({ incluyeEntrega: true, cuotas: [] })).toBe('Entrega')
    expect(paymentItemsLabel({ incluyeEntrega: true, cuotas: [cuotas[1]] })).toBe('Entrega + Cuota 2')
    expect(paymentItemsLabel({ incluyeEntrega: false, cuotas: cuotas.slice(1) })).toBe('Cuotas 2 a 4')
    expect(paymentItemsLabel({ incluyeEntrega: false, cuotas: [] })).toBe('—')
  })
})
