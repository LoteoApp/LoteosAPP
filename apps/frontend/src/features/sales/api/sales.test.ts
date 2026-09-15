import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSale, getSale, listSales } from './sales'

const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

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
  modalidadPago: 'contado',
  monto: 120000,
  moneda: 'USD',
  estado: 'activa',
  fechaCreacion: '2026-09-14T15:00:00Z',
  fechaModificacion: '2026-09-14T15:00:00Z',
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(response: Response) {
  const fetchMock = vi.fn(async () => response)
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('listSales', () => {
  it('sends the filters as query parameters with the bearer token', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { ventas: [sale], pagina: 2, porPagina: 10, total: 11, paginas: 2 }))

    const page = await listSales('token-123', { q: 'Ana', estado: 'activa', loteoId: 'loteo-1', loteId: 'lot-1', pagina: 2, porPagina: 10 })

    expect(page.ventas).toHaveLength(1)
    expect(page.total).toBe(11)
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/ventas?')
    for (const pair of ['q=Ana', 'estado=activa', 'loteoId=loteo-1', 'loteId=lot-1', 'pagina=2', 'porPagina=10']) {
      expect(url).toContain(pair)
    }
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('omits the query string without filters', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { ventas: [], pagina: 1, porPagina: 25, total: 0, paginas: 0 }))

    await listSales('token-123')

    const [url] = fetchMock.mock.calls[0] as unknown as [string]
    expect(url.endsWith('/api/v1/ventas')).toBe(true)
  })

  it('accepts a venta directa without an agency', async () => {
    const { inmobiliaria: _agency, ...direct } = sale
    stubFetch(jsonResponse(200, { ventas: [direct], pagina: 1, porPagina: 25, total: 1, paginas: 1 }))

    const page = await listSales('token-123')

    expect(page.ventas[0].inmobiliaria).toBeUndefined()
  })

  it('rejects a page whose items are not sales', async () => {
    stubFetch(jsonResponse(200, { ventas: [{ id: 'sale-1' }], pagina: 1, porPagina: 25, total: 1, paginas: 1 }))

    await expect(listSales('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('rejects a sale with an unknown payment method or state', async () => {
    stubFetch(jsonResponse(200, { ventas: [{ ...sale, modalidadPago: 'cripto' }], pagina: 1, porPagina: 25, total: 1, paginas: 1 }))
    await expect(listSales('token-123')).rejects.toThrow(GENERIC_ERROR)

    stubFetch(jsonResponse(200, { ventas: [{ ...sale, estado: 'vencida' }], pagina: 1, porPagina: 25, total: 1, paginas: 1 }))
    await expect(listSales('token-123')).rejects.toThrow(GENERIC_ERROR)
  })

  it('throws the message returned by the backend', async () => {
    stubFetch(jsonResponse(403, { code: 'forbidden', message: 'No tenés permisos para esta acción' }))

    await expect(listSales('token-123')).rejects.toThrow('No tenés permisos para esta acción')
  })
})

describe('getSale', () => {
  it('reads the sale by id', async () => {
    const fetchMock = stubFetch(jsonResponse(200, sale))

    const loaded = await getSale('token-123', 'sale 1')

    expect(loaded.id).toBe('sale-1')
    const [url] = fetchMock.mock.calls[0] as unknown as [string]
    expect(url).toContain('/api/v1/ventas/sale%201')
  })

  it('rejects a body that is not a sale', async () => {
    stubFetch(jsonResponse(200, { id: 'sale-1' }))

    await expect(getSale('token-123', 'sale-1')).rejects.toThrow(GENERIC_ERROR)
  })
})

describe('createSale', () => {
  it('posts the sale under the lote of its loteo and returns the persisted venta', async () => {
    const fetchMock = stubFetch(jsonResponse(201, sale))

    const created = await createSale('token-123', {
      loteoId: 'loteo 1',
      loteId: 'lot-1',
      clienteId: 'client-1',
      vendedorId: 'seller-1',
      modalidadPago: 'contado',
    })

    expect(created.id).toBe('sale-1')
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toContain('/api/v1/loteos/loteo%201/lotes/lot-1/ventas')
    expect(init.method).toBe('POST')
    expect(JSON.parse(String(init.body))).toEqual({ clienteId: 'client-1', vendedorId: 'seller-1', modalidadPago: 'contado' })
  })

  it('throws the backend message when the sale is rejected', async () => {
    stubFetch(jsonResponse(409, { code: 'sale_lot_unavailable', message: 'El lote no está disponible para vender' }))

    await expect(
      createSale('token-123', { loteoId: 'loteo-1', loteId: 'lot-1', clienteId: 'client-1', vendedorId: 'seller-1', modalidadPago: 'contado' }),
    ).rejects.toThrow('El lote no está disponible para vender')
  })
})
