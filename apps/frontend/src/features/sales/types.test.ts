import { describe, expect, it } from 'vitest'
import {
  DIRECT_SALE,
  EMPTY_PAYMENT_PLAN,
  agencyOptionsFromSellers,
  buildPaymentSchedule,
  buildSaleReceipt,
  roundMoney,
  clientOptionLabel,
  defaultSeller,
  isLotState,
  lotOptionFromDevelopment,
  lotOptionLabel,
  saleDisabledReason,
  saleLotLabel,
  saleReceiptFromSale,
  sellerAgencyLabel,
  sellerOptionLabel,
  sellersOfAgency,
  type LotOption,
  type SellerOption,
} from './types'

const agencySeller: SellerOption = {
  id: 'us-1',
  nombre: 'Marta',
  apellido: 'Suárez',
  rol: 'inmobiliaria',
  inmobiliariaId: 'ag-1',
  inmobiliariaRazonSocial: 'Inmobiliaria Sur',
}

const otherAgencySeller: SellerOption = {
  id: 'us-2',
  nombre: 'Diego',
  apellido: 'Ramos',
  rol: 'inmobiliaria',
  inmobiliariaId: 'ag-2',
  inmobiliariaRazonSocial: 'Norte Propiedades',
}

const sameAgencySeller: SellerOption = {
  id: 'us-4',
  nombre: 'Carla',
  apellido: 'Vega',
  rol: 'inmobiliaria',
  inmobiliariaId: 'ag-1',
  inmobiliariaRazonSocial: 'Inmobiliaria Sur',
}

const directSeller: SellerOption = {
  id: 'us-3',
  nombre: 'Sofía',
  apellido: 'Luna',
  rol: 'administrativo',
}

function lot(overrides: Partial<LotOption> = {}): LotOption {
  return {
    id: 'lt-1',
    number: '7',
    blockId: 'mz-1',
    blockNumber: '1',
    developmentId: 'loteo-1',
    developmentName: 'Norte',
    state: 'disponible',
    price: 150000,
    currency: 'USD',
    area: 300,
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

describe('lotOptionLabel', () => {
  it('names a lote by its loteo, manzana and number', () => {
    expect(lotOptionLabel(lot())).toBe('Norte · Mz 1 · Lote 7')
  })

  it('falls back to a dash where the manzana or the lote has no number', () => {
    expect(lotOptionLabel(lot({ number: '', blockNumber: '' }))).toBe('Norte · Mz — · Lote —')
  })
})

describe('clientOptionLabel', () => {
  it('names a cliente by apellido, nombre and DNI', () => {
    expect(
      clientOptionLabel({ id: 'cl-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }),
    ).toBe('Pérez, Ana · DNI 30111222')
  })
})

describe('sellerOptionLabel', () => {
  it('names a vendedor by apellido and nombre', () => {
    expect(sellerOptionLabel(agencySeller)).toBe('Suárez, Marta')
  })
})

describe('sellerAgencyLabel', () => {
  it('reads the agency of the vendedor', () => {
    expect(sellerAgencyLabel(agencySeller)).toBe('Inmobiliaria Sur')
  })

  it('reads a vendedor without agency as a direct sale', () => {
    expect(sellerAgencyLabel(directSeller)).toBe('Venta directa')
  })
})

describe('agencyOptionsFromSellers', () => {
  it('lists each agency once, sorted, with venta directa first', () => {
    expect(
      agencyOptionsFromSellers([otherAgencySeller, directSeller, agencySeller, sameAgencySeller]),
    ).toEqual([
      DIRECT_SALE,
      { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
      { id: 'ag-2', razonSocial: 'Norte Propiedades' },
    ])
  })

  it('omits venta directa when every vendedor belongs to an agency', () => {
    expect(agencyOptionsFromSellers([agencySeller, otherAgencySeller])).toEqual([
      { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
      { id: 'ag-2', razonSocial: 'Norte Propiedades' },
    ])
  })

  it('has nothing to offer without sellers', () => {
    expect(agencyOptionsFromSellers([])).toEqual([])
  })
})

describe('defaultSeller', () => {
  it('picks the logged-in user, whoever else is eligible', () => {
    expect(
      defaultSeller([agencySeller, { ...directSeller, esActor: true }, otherAgencySeller]),
    ).toEqual({ ...directSeller, esActor: true })
  })

  it('picks the only eligible seller when none is the logged-in user', () => {
    expect(defaultSeller([agencySeller])).toEqual(agencySeller)
  })

  it('picks nobody when there are several and none is the logged-in user', () => {
    expect(defaultSeller([agencySeller, otherAgencySeller])).toBeNull()
  })

  it('picks nobody without sellers', () => {
    expect(defaultSeller([])).toBeNull()
  })
})

describe('sellersOfAgency', () => {
  const sellers = [agencySeller, otherAgencySeller, directSeller, sameAgencySeller]

  it('keeps the vendedores of the chosen agency', () => {
    expect(sellersOfAgency(sellers, { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' })).toEqual([
      agencySeller,
      sameAgencySeller,
    ])
  })

  it('reads venta directa as the vendedores without agency', () => {
    expect(sellersOfAgency(sellers, DIRECT_SALE)).toEqual([directSeller])
  })

  it('offers nobody until an agency is chosen', () => {
    expect(sellersOfAgency(sellers, null)).toEqual([])
  })
})

describe('buildSaleReceipt', () => {
  const client = { id: 'cl-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }
  const seller: SellerOption = {
    id: 'us-1',
    nombre: 'Marta',
    apellido: 'Suárez',
    rol: 'inmobiliaria',
    inmobiliariaId: 'ag-1',
    inmobiliariaRazonSocial: 'Inmobiliaria Sur',
  }

  const plan = EMPTY_PAYMENT_PLAN

  it('builds the receipt of a contado sale with the price of the lote', () => {
    const result = buildSaleReceipt(
      { lot: lot(), client, seller, method: 'contado', plan },
      '2026-09-07T10:00:00.000Z',
    )

    expect(result).toEqual({
      ok: true,
      plan: undefined,
      receipt: {
        issuedAt: '2026-09-07T10:00:00.000Z',
        lot: lot(),
        client,
        seller,
        method: 'contado',
        amount: 150000,
        currency: 'USD',
      },
    })
  })

  it('accepts a sale by a seller without inmobiliaria', () => {
    const result = buildSaleReceipt(
      { lot: lot(), client, seller: directSeller, method: 'contado', plan },
      '2026-09-07T10:00:00.000Z',
    )

    expect(result.ok).toBe(true)
  })

  it('asks for the lote, the cliente, the vendedor and a price', () => {
    expect(
      buildSaleReceipt({ lot: null, client, seller, method: 'contado', plan }, 'now'),
    ).toEqual({ ok: false, error: 'Elegí el lote que se vende.' })

    expect(
      buildSaleReceipt({ lot: lot(), client: null, seller, method: 'contado', plan }, 'now'),
    ).toEqual({ ok: false, error: 'Elegí el cliente comprador.' })

    expect(
      buildSaleReceipt({ lot: lot(), client, seller: null, method: 'contado', plan }, 'now'),
    ).toEqual({ ok: false, error: 'Elegí el vendedor que realizó la venta.' })

    expect(
      buildSaleReceipt(
        { lot: lot({ price: null }), client, seller, method: 'contado', plan },
        'now',
      ),
    ).toEqual({
      ok: false,
      error: 'Completá el precio del lote para habilitar la venta.',
    })
  })

  it('refuses a lote the backend would reject: price zero or no currency', () => {
    expect(
      buildSaleReceipt({ lot: lot({ price: 0 }), client, seller, method: 'contado', plan }, 'now'),
    ).toEqual({ ok: false, error: 'Completá el precio del lote para habilitar la venta.' })

    expect(
      buildSaleReceipt({ lot: lot({ currency: '' }), client, seller, method: 'contado', plan }, 'now'),
    ).toEqual({ ok: false, error: 'Completá la moneda del lote para habilitar la venta.' })
  })

  it('builds a financed receipt with the plan and its derived amounts', () => {
    const result = buildSaleReceipt(
      {
        lot: lot(),
        client,
        seller,
        method: 'financiado',
        plan: { cantidadCuotas: '12', tasaInteres: '10', periodicidad: 'mensual', montoEntrega: '' },
      },
      'now',
    )

    expect(result).toEqual({
      ok: true,
      plan: { cantidadCuotas: 12, tasaInteres: 10, periodicidad: 'mensual', montoEntrega: 0 },
      receipt: expect.objectContaining({
        method: 'financiado',
        amount: 150000,
        plan: {
          montoEntrega: 0,
          cantidadCuotas: 12,
          tasaInteres: 10,
          periodicidad: 'mensual',
          montoFinanciado: 150000,
          montoCuota: 13750,
          montoTotal: 165000,
        },
      }),
    })
  })

  it('builds an entrega + financiación receipt with the down payment taken off', () => {
    const result = buildSaleReceipt(
      {
        lot: lot(),
        client,
        seller,
        method: 'entrega_financiada',
        plan: { cantidadCuotas: '3', tasaInteres: '', periodicidad: 'trimestral', montoEntrega: '50.000,50' },
      },
      'now',
    )

    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.plan).toEqual({ cantidadCuotas: 3, tasaInteres: 0, periodicidad: 'trimestral', montoEntrega: 50000.5 })
    expect(result.receipt.plan).toEqual({
      montoEntrega: 50000.5,
      cantidadCuotas: 3,
      tasaInteres: 0,
      periodicidad: 'trimestral',
      montoFinanciado: 99999.5,
      montoCuota: 33333.17,
      montoTotal: 99999.5,
    })
  })

  it('explains what is wrong with the plan before confirming', () => {
    const draft = (method: 'financiado' | 'entrega_financiada', values: Partial<typeof plan>) =>
      buildSaleReceipt({ lot: lot(), client, seller, method, plan: { ...plan, ...values } }, 'now')

    expect(draft('financiado', {})).toEqual({
      ok: false,
      error: 'Ingresá una cantidad de cuotas entre 1 y 360.',
    })
    expect(draft('financiado', { cantidadCuotas: '2.5' })).toEqual({
      ok: false,
      error: 'Ingresá una cantidad de cuotas entre 1 y 360.',
    })
    expect(draft('financiado', { cantidadCuotas: '361' })).toEqual({
      ok: false,
      error: 'Ingresá una cantidad de cuotas entre 1 y 360.',
    })
    expect(draft('financiado', { cantidadCuotas: '12', tasaInteres: '-1' })).toEqual({
      ok: false,
      error: 'Ingresá una tasa de interés entre 0 y 1000 %.',
    })
    expect(draft('financiado', { cantidadCuotas: '12', tasaInteres: 'diez' })).toEqual({
      ok: false,
      error: 'Ingresá una tasa de interés entre 0 y 1000 %.',
    })
    expect(draft('entrega_financiada', { cantidadCuotas: '12' })).toEqual({
      ok: false,
      error: 'Ingresá el monto de la entrega.',
    })
    expect(draft('entrega_financiada', { cantidadCuotas: '12', montoEntrega: '0' })).toEqual({
      ok: false,
      error: 'Ingresá el monto de la entrega.',
    })
    expect(draft('entrega_financiada', { cantidadCuotas: '12', montoEntrega: '150000' })).toEqual({
      ok: false,
      error: 'La entrega tiene que ser menor al precio del lote.',
    })
    expect(draft('entrega_financiada', { cantidadCuotas: '360', montoEntrega: '149.999,99' })).toEqual({
      ok: false,
      error: 'El monto financiado no alcanza para esa cantidad de cuotas.',
    })
  })
})

describe('buildPaymentSchedule', () => {
  const sum = (cuotas: number[]) => roundMoney(cuotas.reduce((total, cuota) => total + cuota, 0))

  it('splits the price into equal cuotas and lets the last one absorb the remainder', () => {
    const schedule = buildPaymentSchedule(100000, {
      cantidadCuotas: 12,
      tasaInteres: 0,
      periodicidad: 'mensual',
      montoEntrega: 0,
    })

    expect(schedule.montoFinanciado).toBe(100000)
    expect(schedule.montoTotal).toBe(100000)
    expect(schedule.montoCuota).toBe(8333.33)
    expect(schedule.cuotas).toHaveLength(12)
    expect(schedule.cuotas.slice(0, 11).every((cuota) => cuota === 8333.33)).toBe(true)
    expect(schedule.cuotas[11]).toBe(8333.37)
    expect(sum(schedule.cuotas)).toBe(100000)
  })

  it('applies simple interest on the amount left after the down payment', () => {
    const schedule = buildPaymentSchedule(1100, {
      cantidadCuotas: 3,
      tasaInteres: 10,
      periodicidad: 'mensual',
      montoEntrega: 1000,
    })

    expect(schedule).toEqual({
      montoFinanciado: 100,
      montoTotal: 110,
      montoCuota: 36.67,
      cuotas: [36.67, 36.67, 36.66],
    })
  })

  it('matches the backend on a single cuota and on awkward decimals', () => {
    expect(
      buildPaymentSchedule(999.99, { cantidadCuotas: 1, tasaInteres: 12.5, periodicidad: 'mensual', montoEntrega: 0 }),
    ).toEqual({ montoFinanciado: 999.99, montoTotal: 1124.99, montoCuota: 1124.99, cuotas: [1124.99] })

    for (const cantidadCuotas of [2, 7, 13, 97, 360]) {
      const schedule = buildPaymentSchedule(123456.78, { cantidadCuotas, tasaInteres: 33.3, periodicidad: 'mensual', montoEntrega: 0 })
      expect(sum(schedule.cuotas)).toBe(schedule.montoTotal)
    }
  })

  it('rounds half away from zero on the same doubles as the backend', () => {
    expect(roundMoney(1.005)).toBe(1)
    expect(roundMoney(2.675)).toBe(2.68)
    expect(roundMoney(0.125)).toBe(0.13)
    expect(roundMoney(33.33333)).toBe(33.33)
  })
})

describe('lotOptionFromDevelopment', () => {
  const development = {
    id: 'loteo-1',
    nombre: 'Las Acacias',
    ubicacion: 'Córdoba',
    descripcion: '',
    manzanas: [{ id: 'block-1', numero: '2', tieneAgua: true, tieneCloaca: false, tieneLuz: true, tieneGas: false }],
    lotes: [
      { id: 'lot-1', manzanaId: 'block-1', numero: '7', estado: 'disponible' as const, precio: 120000, moneda: 'USD', superficie: 300, caracteristicas: '' },
      { id: 'lot-2', manzanaId: 'block-x', numero: '8', estado: 'reservado' as const, precio: null, moneda: 'ARS', superficie: null, caracteristicas: '' },
    ],
  }

  it('builds the lote option with the names of its loteo and manzana', () => {
    expect(lotOptionFromDevelopment(development, 'lot-1')).toEqual({
      id: 'lot-1',
      number: '7',
      blockId: 'block-1',
      blockNumber: '2',
      developmentId: 'loteo-1',
      developmentName: 'Las Acacias',
      state: 'disponible',
      price: 120000,
      currency: 'USD',
      area: 300,
    })
  })

  it('leaves the manzana number empty when the manzana is unknown', () => {
    expect(lotOptionFromDevelopment(development, 'lot-2')).toMatchObject({ blockNumber: '', price: null })
  })

  it('returns null for a lote that is not in the loteo', () => {
    expect(lotOptionFromDevelopment(development, 'lot-999')).toBeNull()
  })
})

describe('saleDisabledReason', () => {
  it('allows a lote with number, a price above zero and a currency', () => {
    expect(saleDisabledReason({ number: '7', price: 1, currency: 'USD' })).toBeNull()
  })

  it('names each missing field', () => {
    expect(saleDisabledReason({ number: '', price: 1, currency: 'USD' })).toBe('Completá el número del lote para habilitar la venta.')
    expect(saleDisabledReason({ number: '7', price: null, currency: 'USD' })).toBe('Completá el precio del lote para habilitar la venta.')
    expect(saleDisabledReason({ number: '  ', price: null, currency: '' })).toBe(
      'Completá el número y el precio del lote para habilitar la venta.',
    )
  })

  it('treats a price of zero like a missing price and asks for the currency of a priced lote', () => {
    expect(saleDisabledReason({ number: '7', price: 0, currency: 'USD' })).toBe('Completá el precio del lote para habilitar la venta.')
    expect(saleDisabledReason({ number: '7', price: 100, currency: ' ' })).toBe('Completá la moneda del lote para habilitar la venta.')
  })
})

describe('saleReceiptFromSale', () => {
  const sale = {
    id: 'sale-1',
    loteoId: 'loteo-1',
    loteoNombre: 'Las Acacias',
    loteId: 'lot-1',
    loteNumero: '7',
    manzanaNumero: '2',
    loteSuperficie: 300,
    cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
    vendedor: { id: 'seller-1', nombre: 'Marta', apellido: 'Suárez', rol: 'inmobiliaria' },
    usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
    inmobiliaria: { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
    modalidadPago: 'contado' as const,
    monto: 120000,
    moneda: 'USD',
    estado: 'activa' as const,
    fechaCreacion: '2026-09-14T15:00:00Z',
    fechaModificacion: '2026-09-14T15:00:00Z',
  }

  it('turns the persisted sale into the receipt the dialog prints', () => {
    const receipt = saleReceiptFromSale(sale)

    expect(receipt.issuedAt).toBe('2026-09-14T15:00:00Z')
    expect(receipt.amount).toBe(120000)
    expect(receipt.currency).toBe('USD')
    expect(receipt.method).toBe('contado')
    expect(lotOptionLabel(receipt.lot)).toBe('Las Acacias · Mz 2 · Lote 7')
    expect(receipt.lot.area).toBe(300)
    expect(receipt.client).toEqual(sale.cliente)
    expect(sellerOptionLabel(receipt.seller)).toBe('Suárez, Marta')
    expect(sellerAgencyLabel(receipt.seller)).toBe('Inmobiliaria Sur')
  })

  it('labels a sale without an agency as a venta directa', () => {
    const { inmobiliaria: _agency, ...direct } = sale

    expect(sellerAgencyLabel(saleReceiptFromSale(direct).seller)).toBe('Venta directa')
    expect(saleReceiptFromSale(direct).plan).toBeUndefined()
  })

  it('carries the plan of a financed sale into the receipt', () => {
    const receipt = saleReceiptFromSale({
      ...sale,
      modalidadPago: 'entrega_financiada',
      planPago: {
        id: 'plan-1',
        montoEntrega: 20000,
        cantidadCuotas: 10,
        tasaInteres: 5,
        periodicidad: 'mensual',
        moneda: 'USD',
        montoFinanciado: 100000,
        montoCuota: 10500,
        montoTotal: 105000,
      },
    })

    expect(receipt.method).toBe('entrega_financiada')
    expect(receipt.plan).toEqual({
      montoEntrega: 20000,
      cantidadCuotas: 10,
      tasaInteres: 5,
      periodicidad: 'mensual',
      montoFinanciado: 100000,
      montoCuota: 10500,
      montoTotal: 105000,
    })
  })

  it('labels the lote of a sale', () => {
    expect(saleLotLabel(sale)).toBe('Las Acacias · Mz 2 · Lote 7')
    expect(saleLotLabel({ loteoNombre: 'Norte', manzanaNumero: '', loteNumero: '' })).toBe('Norte · Mz — · Lote —')
  })
})
