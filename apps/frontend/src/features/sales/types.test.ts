import { describe, expect, it } from 'vitest'
import {
  DIRECT_SALE,
  agencyOptionsFromSellers,
  buildSaleReceipt,
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

function lote(overrides: Partial<LotOption> = {}): LotOption {
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

describe('lotOptionLabel', () => {
  it('names a lote by its loteo, manzana and number', () => {
    expect(lotOptionLabel(lote())).toBe('Norte · Mz 1 · Lote 7')
  })

  it('falls back to a dash where the manzana or the lote has no number', () => {
    expect(lotOptionLabel(lote({ numero: '', manzanaNumero: '' }))).toBe('Norte · Mz — · Lote —')
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

  it('builds the receipt of a contado sale with the price of the lote', () => {
    const result = buildSaleReceipt(
      { lot: lote(), client, seller, method: 'contado' },
      '2026-09-07T10:00:00.000Z',
    )

    expect(result).toEqual({
      ok: true,
      receipt: {
        issuedAt: '2026-09-07T10:00:00.000Z',
        lot: lote(),
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
      { lot: lote(), client, seller: directSeller, method: 'contado' },
      '2026-09-07T10:00:00.000Z',
    )

    expect(result.ok).toBe(true)
  })

  it('asks for the lote, the cliente, the vendedor, an available modalidad and a price', () => {
    expect(
      buildSaleReceipt({ lot: null, client, seller, method: 'contado' }, 'now'),
    ).toEqual({ ok: false, error: 'Elegí el lote que se vende.' })

    expect(
      buildSaleReceipt({ lot: lote(), client: null, seller, method: 'contado' }, 'now'),
    ).toEqual({ ok: false, error: 'Elegí el cliente comprador.' })

    expect(
      buildSaleReceipt({ lot: lote(), client, seller: null, method: 'contado' }, 'now'),
    ).toEqual({ ok: false, error: 'Elegí el vendedor que realizó la venta.' })

    expect(
      buildSaleReceipt({ lot: lote(), client, seller, method: 'financiado' }, 'now'),
    ).toEqual({
      ok: false,
      error: 'Por ahora solo se puede registrar una venta al contado.',
    })

    expect(
      buildSaleReceipt(
        { lot: lote({ precio: null }), client, seller, method: 'contado' },
        'now',
      ),
    ).toEqual({
      ok: false,
      error: 'Completá el precio del lote para habilitar la venta.',
    })
  })

  it('refuses a lote the backend would reject: price zero or no currency', () => {
    expect(
      buildSaleReceipt({ lot: lote({ precio: 0 }), client, seller, method: 'contado' }, 'now'),
    ).toEqual({ ok: false, error: 'Completá el precio del lote para habilitar la venta.' })

    expect(
      buildSaleReceipt({ lot: lote({ moneda: '' }), client, seller, method: 'contado' }, 'now'),
    ).toEqual({ ok: false, error: 'Completá la moneda del lote para habilitar la venta.' })
  })
})

describe('lotOptionFromDevelopment', () => {
  const loteo = {
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
    expect(lotOptionFromDevelopment(loteo, 'lot-1')).toEqual({
      id: 'lot-1',
      numero: '7',
      manzanaId: 'block-1',
      manzanaNumero: '2',
      loteoId: 'loteo-1',
      loteoNombre: 'Las Acacias',
      estado: 'disponible',
      precio: 120000,
      moneda: 'USD',
      superficie: 300,
    })
  })

  it('leaves the manzana number empty when the manzana is unknown', () => {
    expect(lotOptionFromDevelopment(loteo, 'lot-2')).toMatchObject({ manzanaNumero: '', precio: null })
  })

  it('returns null for a lote that is not in the loteo', () => {
    expect(lotOptionFromDevelopment(loteo, 'lot-999')).toBeNull()
  })
})

describe('saleDisabledReason', () => {
  it('allows a lote with number, a price above zero and a currency', () => {
    expect(saleDisabledReason({ numero: '7', precio: 1, moneda: 'USD' })).toBeNull()
  })

  it('names each missing field', () => {
    expect(saleDisabledReason({ numero: '', precio: 1, moneda: 'USD' })).toBe('Completá el número del lote para habilitar la venta.')
    expect(saleDisabledReason({ numero: '7', precio: null, moneda: 'USD' })).toBe('Completá el precio del lote para habilitar la venta.')
    expect(saleDisabledReason({ numero: '  ', precio: null, moneda: '' })).toBe(
      'Completá el número y el precio del lote para habilitar la venta.',
    )
  })

  it('treats a price of zero like a missing price and asks for the currency of a priced lote', () => {
    expect(saleDisabledReason({ numero: '7', precio: 0, moneda: 'USD' })).toBe('Completá el precio del lote para habilitar la venta.')
    expect(saleDisabledReason({ numero: '7', precio: 100, moneda: ' ' })).toBe('Completá la moneda del lote para habilitar la venta.')
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
    expect(receipt.lot.superficie).toBe(300)
    expect(receipt.client).toEqual(sale.cliente)
    expect(sellerOptionLabel(receipt.seller)).toBe('Suárez, Marta')
    expect(sellerAgencyLabel(receipt.seller)).toBe('Inmobiliaria Sur')
  })

  it('labels a sale without an agency as a venta directa', () => {
    const { inmobiliaria: _agency, ...direct } = sale

    expect(sellerAgencyLabel(saleReceiptFromSale(direct).seller)).toBe('Venta directa')
  })

  it('labels the lote of a sale', () => {
    expect(saleLotLabel(sale)).toBe('Las Acacias · Mz 2 · Lote 7')
    expect(saleLotLabel({ loteoNombre: 'Norte', manzanaNumero: '', loteNumero: '' })).toBe('Norte · Mz — · Lote —')
  })
})
