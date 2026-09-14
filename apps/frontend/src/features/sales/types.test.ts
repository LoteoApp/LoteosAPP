import { describe, expect, it } from 'vitest'
import {
  DIRECT_SALE,
  agencyOptionsFromSellers,
  buildSaleReceipt,
  clienteOptionLabel,
  defaultSeller,
  isLotState,
  loteOptionLabel,
  sellerAgencyLabel,
  sellerOptionLabel,
  sellersOfAgency,
  type LoteOption,
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
  const cliente = { id: 'cl-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }
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
      { lote: lote(), cliente, seller, method: 'contado' },
      '2026-09-07T10:00:00.000Z',
    )

    expect(result).toEqual({
      ok: true,
      receipt: {
        emitidoEl: '2026-09-07T10:00:00.000Z',
        lote: lote(),
        cliente,
        seller,
        method: 'contado',
        monto: 150000,
        moneda: 'USD',
      },
    })
  })

  it('accepts a sale by a seller without inmobiliaria', () => {
    const result = buildSaleReceipt(
      { lote: lote(), cliente, seller: directSeller, method: 'contado' },
      '2026-09-07T10:00:00.000Z',
    )

    expect(result.ok).toBe(true)
  })

  it('asks for the lote, the cliente, the vendedor, an available modalidad and a price', () => {
    expect(
      buildSaleReceipt({ lote: null, cliente, seller, method: 'contado' }, 'now'),
    ).toEqual({ ok: false, error: 'Elegí el lote que se vende.' })

    expect(
      buildSaleReceipt({ lote: lote(), cliente: null, seller, method: 'contado' }, 'now'),
    ).toEqual({ ok: false, error: 'Elegí el cliente comprador.' })

    expect(
      buildSaleReceipt({ lote: lote(), cliente, seller: null, method: 'contado' }, 'now'),
    ).toEqual({ ok: false, error: 'Elegí el vendedor que realizó la venta.' })

    expect(
      buildSaleReceipt({ lote: lote(), cliente, seller, method: 'financiado' }, 'now'),
    ).toEqual({
      ok: false,
      error: 'Por ahora solo se puede registrar una venta al contado.',
    })

    expect(
      buildSaleReceipt(
        { lote: lote({ precio: null }), cliente, seller, method: 'contado' },
        'now',
      ),
    ).toEqual({
      ok: false,
      error: 'El lote no tiene precio cargado. Cargalo en el detalle del loteo antes de vender.',
    })
  })
})
