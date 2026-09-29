import { describe, expect, it } from 'vitest'
import {
  MAX_PAYMENT_CHARGES,
  chargeLabel,
  chargeTotals,
  clientLabel,
  currencyOptions,
  isDueFilter,
  isInstallmentState,
  isChargeType,
  isPaymentMedium,
  isPaymentType,
  lotLabel,
  newChargeRow,
  parseCharges,
  paymentDateToISO,
  paymentInstant,
  paymentItemsLabel,
  paymentTotals,
  pendingInstallments,
  selectionAmount,
  todayInputValue,
  toggleInstallment,
  validatePaymentTerms,
  type ChargeRow,
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
    expect(isChargeType('gasto_administrativo')).toBe(true)
    expect(isChargeType('propina')).toBe(false)
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

  // Anchoring today at noon is a future instant all morning, and the backend
  // rejects a payment dated in the future (payment_date_in_future).
  it('sends nothing for today so the backend stamps its own now', () => {
    expect(paymentInstant('2026-09-22', '2026-09-22')).toBe('')
    expect(paymentInstant(' 2026-09-22 ', '2026-09-22')).toBe('')
    expect(paymentInstant('', '2026-09-22')).toBe('')
    expect(paymentInstant(todayInputValue())).toBe('')
  })

  it('sends a past day as its local noon', () => {
    const iso = paymentInstant('2026-09-20', '2026-09-22')
    expect(new Date(iso).getDate()).toBe(20)
    expect(new Date(iso).getHours()).toBe(12)
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

describe('parseCharges', () => {
  function row(overrides: Partial<ChargeRow> = {}): ChargeRow {
    return { ...newChargeRow('USD'), monto: '100', ...overrides }
  }

  it('parses the typed rows and defaults the currency to the sale\'s', () => {
    const parsed = parseCharges(
      [
        row({ tipo: 'servicios', monto: '1.234,50', moneda: 'ars', detalle: '  Agua  ' }),
        row({ tipo: 'gasto_administrativo', monto: '25', moneda: '  ' }),
      ],
      'usd',
    )

    expect(parsed).toEqual({
      ok: true,
      charges: [
        { tipo: 'servicios', monto: 1234.5, moneda: 'ARS', detalle: 'Agua' },
        { tipo: 'gasto_administrativo', monto: 25, moneda: 'USD', detalle: '' },
      ],
    })
    expect(parseCharges([], 'USD')).toEqual({ ok: true, charges: [] })
  })

  it('explains what is wrong with a row', () => {
    expect(parseCharges([row({ monto: '' })], 'USD')).toEqual({
      ok: false,
      error: 'Ingresá el monto de "Servicios", mayor a cero y con hasta 2 decimales.',
    })
    expect(parseCharges([row({ monto: '0' })], 'USD').ok).toBe(false)
    expect(parseCharges([row({ monto: '10,555' })], 'USD').ok).toBe(false)
    expect(parseCharges([row({ tipo: 'propina' as never })], 'USD')).toEqual({
      ok: false,
      error: 'Elegí el tipo de cada cargo adicional.',
    })
    expect(parseCharges([row({ moneda: '' })], '')).toEqual({
      ok: false,
      error: 'Ingresá la moneda de "Servicios".',
    })
    expect(parseCharges([row({ moneda: 'PESOS ARGENTINOS' })], 'USD').ok).toBe(false)
    expect(parseCharges([row({ detalle: 'x'.repeat(201) })], 'USD')).toEqual({
      ok: false,
      error: 'El detalle del cargo no puede superar los 200 caracteres.',
    })
    const tooMany = Array.from({ length: MAX_PAYMENT_CHARGES + 1 }, () => row())
    expect(parseCharges(tooMany, 'USD')).toEqual({
      ok: false,
      error: `No se pueden cargar más de ${MAX_PAYMENT_CHARGES} cargos adicionales.`,
    })
  })

  it('gives every row its own key', () => {
    expect(newChargeRow('USD').key).not.toBe(newChargeRow('USD').key)
    expect(newChargeRow('ARS').moneda).toBe('ARS')
  })

  it('offers the sale currency first and never repeats it', () => {
    expect(currencyOptions('usd')).toEqual(['USD', 'ARS'])
    expect(currencyOptions('ARS')).toEqual(['ARS', 'USD'])
    // A lote priced in something else keeps its currency available.
    expect(currencyOptions('EUR')).toEqual(['EUR', 'ARS', 'USD'])
    expect(currencyOptions(' ')).toEqual(['ARS', 'USD'])
  })
})

describe('paymentTotals', () => {
  it('keeps each currency apart, the sale\'s first', () => {
    const totals = paymentTotals(20000, 'usd', [
      { monto: 150000, moneda: 'ARS' },
      { monto: 12000.5, moneda: 'ARS' },
      { monto: 300, moneda: 'USD' },
      { monto: 40, moneda: 'EUR' },
    ])

    expect(totals).toEqual([
      { moneda: 'USD', monto: 20300 },
      { moneda: 'ARS', monto: 162000.5 },
      { moneda: 'EUR', monto: 40 },
    ])
  })

  it('handles a plan-only and a charges-only cobro', () => {
    expect(paymentTotals(500, 'USD', [])).toEqual([{ moneda: 'USD', monto: 500 }])
    expect(paymentTotals(0, 'USD', [{ monto: 10, moneda: 'ARS' }])).toEqual([{ moneda: 'ARS', monto: 10 }])
    expect(paymentTotals(0, 'USD', [])).toEqual([])
    expect(chargeTotals([{ monto: 1000, moneda: 'ARS' }, { monto: 20, moneda: 'USD' }])).toEqual([
      { moneda: 'ARS', monto: 1000 },
      { moneda: 'USD', monto: 20 },
    ])
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
    expect(chargeLabel({ tipo: 'servicios', detalle: 'Agua y luz' })).toBe('Servicios · Agua y luz')
    expect(chargeLabel({ tipo: 'honorarios' })).toBe('Honorarios')
    expect(chargeLabel({ tipo: 'otros', detalle: '   ' })).toBe('Otros')
  })
})
