import { afterEach, describe, expect, it, vi } from 'vitest'
import { getDebtStatement, listDueInstallments, registerPayment, settleSale } from './billing'

const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

const dueInstallment = {
  id: 'c-1',
  ventaId: 'sale-1',
  numero: 1,
  cantidadCuotas: 3,
  monto: 20000,
  moneda: 'USD',
  estado: 'vencida',
  fechaVencimiento: '2026-04-15T12:00:00Z',
  loteoId: 'loteo-1',
  loteoNombre: 'Las Acacias',
  loteId: 'lot-1',
  loteNumero: '7',
  manzanaNumero: '2',
  cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
  vendedor: { id: 'seller-1', nombre: 'Marta', apellido: 'Suárez', rol: 'inmobiliaria' },
  inmobiliaria: { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
}

const payment = {
  id: 'cobro-1',
  ventaId: 'sale-1',
  tipo: 'pago',
  monto: 60000,
  moneda: 'USD',
  medioPago: 'transferencia',
  observacion: 'Transf. 123',
  incluyeEntrega: true,
  montoEntrega: 40000,
  cuotas: [{ id: 'c-1', numero: 1, monto: 20000, estado: 'pagada', fechaVencimiento: '2026-04-15T12:00:00Z', fechaPago: '2026-05-01T12:00:00Z', cobroId: 'cobro-1' }],
  usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
  fechaPago: '2026-05-01T12:00:00Z',
  fechaCreacion: '2026-05-01T12:00:00Z',
}

const statement = {
  venta: {
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
    modalidadPago: 'entrega_financiada',
    monto: 100000,
    moneda: 'USD',
    estado: 'activa',
    fechaCreacion: '2026-01-15T12:00:00Z',
    fechaModificacion: '2026-01-15T12:00:00Z',
    planPago: {
      id: 'plan-1',
      montoEntrega: 40000,
      cantidadCuotas: 3,
      tasaInteres: 0,
      periodicidad: 'trimestral',
      moneda: 'USD',
      montoFinanciado: 60000,
      montoCuota: 20000,
      montoTotal: 60000,
    },
  },
  entrega: { monto: 40000, estado: 'pendiente' },
  cuotas: [
    { id: 'c-1', numero: 1, monto: 20000, estado: 'vencida', fechaVencimiento: '2026-04-15T12:00:00Z' },
    { id: 'c-2', numero: 2, monto: 20000, estado: 'pendiente', fechaVencimiento: '2026-07-15T12:00:00Z' },
  ],
  resumen: {
    montoTotal: 100000,
    montoPagado: 0,
    montoPendiente: 100000,
    montoVencido: 20000,
    cuotasPagadas: 0,
    cuotasPendientes: 2,
    cuotasVencidas: 1,
    proximoVencimiento: '2026-07-15T12:00:00Z',
  },
  cobros: [],
  emitidoEl: '2026-05-01T12:00:00Z',
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function stubFetch(response: Response) {
  const fetchMock = vi.fn(async () => response)
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function requestOf(fetchMock: ReturnType<typeof stubFetch>): [string, RequestInit] {
  return fetchMock.mock.calls[0] as unknown as [string, RequestInit]
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('listDueInstallments', () => {
  it('sends the filters as query parameters with the bearer token', async () => {
    const fetchMock = stubFetch(
      jsonResponse(200, { cuotas: [dueInstallment], resumen: { cuotasVencidas: 1, cuotasProximas: 0 }, pagina: 2, porPagina: 10, total: 11, paginas: 2 }),
    )

    const page = await listDueInstallments('token-123', {
      estado: 'vencida', loteoId: 'loteo-1', q: 'Ana', desde: '2026-04-01', hasta: '2026-04-30', pagina: 2, porPagina: 10,
    })

    expect(page.cuotas).toHaveLength(1)
    expect(page.resumen.cuotasVencidas).toBe(1)
    const [url, init] = requestOf(fetchMock)
    expect(url).toContain('/api/v1/cobranzas/vencimientos?')
    for (const pair of ['estado=vencida', 'loteoId=loteo-1', 'q=Ana', 'desde=2026-04-01', 'hasta=2026-04-30', 'pagina=2', 'porPagina=10']) {
      expect(url).toContain(pair)
    }
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('leaves the estado out for the default "a cobrar" filter', async () => {
    const fetchMock = stubFetch(jsonResponse(200, { cuotas: [], resumen: { cuotasVencidas: 0, cuotasProximas: 0 }, pagina: 1, porPagina: 25, total: 0, paginas: 0 }))

    await listDueInstallments('token', { estado: 'pendientes' })

    expect(requestOf(fetchMock)[0]).toMatch(/\/api\/v1\/cobranzas\/vencimientos$/)
  })

  it('rejects a body that does not match the contract', async () => {
    stubFetch(jsonResponse(200, { cuotas: [{ id: 'c-1' }], pagina: 1 }))

    await expect(listDueInstallments('token')).rejects.toThrow(GENERIC_ERROR)
  })
})

describe('getDebtStatement', () => {
  it('reads the statement of a sale', async () => {
    const fetchMock = stubFetch(jsonResponse(200, statement))

    const loaded = await getDebtStatement('token', 'sale 1')

    expect(loaded.resumen.montoVencido).toBe(20000)
    expect(loaded.cuotas).toHaveLength(2)
    expect(requestOf(fetchMock)[0]).toContain('/api/v1/ventas/sale%201/estado-deuda')
  })

  it('accepts a statement without entrega and with cobros', async () => {
    stubFetch(jsonResponse(200, { ...statement, entrega: undefined, cobros: [payment] }))

    const loaded = await getDebtStatement('token', 'sale-1')

    expect(loaded.entrega).toBeUndefined()
    expect(loaded.cobros[0].incluyeEntrega).toBe(true)
  })

  it('rejects malformed statements', async () => {
    stubFetch(jsonResponse(200, { ...statement, resumen: { montoTotal: 'mucho' } }))

    await expect(getDebtStatement('token', 'sale-1')).rejects.toThrow(GENERIC_ERROR)
  })

  it('surfaces the backend error', async () => {
    stubFetch(jsonResponse(400, { code: 'sale_not_financed', message: 'Una venta al contado no tiene deuda que cobrar' }))

    await expect(getDebtStatement('token', 'sale-1')).rejects.toThrow('Una venta al contado no tiene deuda que cobrar')
  })
})

describe('registerPayment', () => {
  it('posts the selection, the terms and the day as an instant', async () => {
    const fetchMock = stubFetch(jsonResponse(201, payment))

    const created = await registerPayment('token', 'sale-1', {
      cuotaIds: ['c-1'], incluirEntrega: true, medioPago: 'transferencia', fechaPago: '2026-05-01', observacion: '  Transf. 123  ',
    })

    expect(created.id).toBe('cobro-1')
    const [url, init] = requestOf(fetchMock)
    expect(url).toContain('/api/v1/ventas/sale-1/cobros')
    expect(init.method).toBe('POST')
    const body = JSON.parse(init.body as string) as Record<string, unknown>
    expect(body.cuotaIds).toEqual(['c-1'])
    expect(body.incluirEntrega).toBe(true)
    expect(body.medioPago).toBe('transferencia')
    expect(body.observacion).toBe('Transf. 123')
    expect(new Date(body.fechaPago as string).getDate()).toBe(1)
  })

  it('omits the date and the observation when empty', async () => {
    const fetchMock = stubFetch(jsonResponse(201, payment))

    await registerPayment('token', 'sale-1', { cuotaIds: ['c-1'], incluirEntrega: false, medioPago: 'efectivo', fechaPago: '', observacion: ' ' })

    const body = JSON.parse(requestOf(fetchMock)[1].body as string) as Record<string, unknown>
    expect(body).toEqual({ cuotaIds: ['c-1'], incluirEntrega: false, medioPago: 'efectivo' })
  })

  it('rejects a malformed payment', async () => {
    stubFetch(jsonResponse(201, { ...payment, cuotas: 'ninguna' }))

    await expect(
      registerPayment('token', 'sale-1', { cuotaIds: ['c-1'], incluirEntrega: false, medioPago: 'efectivo', fechaPago: '', observacion: '' }),
    ).rejects.toThrow(GENERIC_ERROR)
  })
})

describe('settleSale', () => {
  it('posts the expected balance with the terms', async () => {
    const fetchMock = stubFetch(jsonResponse(201, { ...payment, tipo: 'cancelacion_total' }))

    const created = await settleSale('token', 'sale-1', { montoEsperado: 100000, medioPago: 'cheque', fechaPago: '2026-05-01', observacion: 'Cheque 9' })

    expect(created.tipo).toBe('cancelacion_total')
    const [url, init] = requestOf(fetchMock)
    expect(url).toContain('/api/v1/ventas/sale-1/cancelacion-total')
    const body = JSON.parse(init.body as string) as Record<string, unknown>
    expect(body.montoEsperado).toBe(100000)
    expect(body.medioPago).toBe('cheque')
    expect(body.observacion).toBe('Cheque 9')
    expect(typeof body.fechaPago).toBe('string')
  })

  it('omits the optional fields when empty', async () => {
    const fetchMock = stubFetch(jsonResponse(201, { ...payment, tipo: 'cancelacion_total' }))

    await settleSale('token', 'sale-1', { montoEsperado: 100000, medioPago: 'efectivo', fechaPago: '', observacion: '' })

    expect(JSON.parse(requestOf(fetchMock)[1].body as string)).toEqual({ montoEsperado: 100000, medioPago: 'efectivo' })
  })
})
