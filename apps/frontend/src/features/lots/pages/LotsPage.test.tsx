import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import LotsPage from './LotsPage'

function lwpolyline(layer: string, points: Array<[number, number]>): string {
  const lines = ['0', 'LWPOLYLINE', '8', layer, '90', String(points.length), '70', '1']
  for (const [x, y] of points) {
    lines.push('10', String(x), '20', String(y))
  }
  return lines.join('\n')
}

function dxfDocument(...entities: string[]): string {
  return ['0', 'SECTION', '2', 'ENTITIES', ...entities, '0', 'ENDSEC', '0', 'EOF', ''].join('\n')
}

const square: Array<[number, number]> = [
  [0, 0],
  [10, 0],
  [10, 10],
  [0, 10],
]

function planDxf(): File {
  return new File(
    [
      dxfDocument(
        lwpolyline('LOTEO', square),
        lwpolyline('MANZANA', square),
        lwpolyline('LOTES', square),
      ),
    ],
    'plano.dxf',
    { type: 'application/dxf' },
  )
}

function renderPage({ token = 'test-token' }: { token?: string | null } = {}) {
  return render(
    <MemoryRouter>
      <LotsPage accessToken={token} />
    </MemoryRouter>,
  )
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

type FetchCall = { url: string; method: string; body: unknown; headers: Headers }

function stubFetch(handler: (call: FetchCall) => Response | Promise<Response>) {
  const calls: FetchCall[] = []
  const mock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const call: FetchCall = {
      url: String(input),
      method: init?.method ?? 'GET',
      body: init?.body ?? null,
      headers: new Headers(init?.headers),
    }
    calls.push(call)
    return handler(call)
  })
  vi.stubGlobal('fetch', mock)
  return calls
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('LotsPage', () => {
  it('renders the loteo form with the plan pending, DXF optional', () => {
    renderPage()

    expect(screen.getByRole('heading', { name: 'Nuevo loteo' })).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
    expect(screen.getByLabelText('Nombre')).toBeInTheDocument()
    expect(screen.getByLabelText('Ubicación/Ciudad')).toBeInTheDocument()
    expect(screen.getByLabelText('Inmobiliarias')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Seleccionar todas' })).toBeInTheDocument()
    expect(screen.getByLabelText('Descripción')).toBeInTheDocument()
    expect(screen.getByLabelText('Archivo DXF')).toBeInTheDocument()
    expect(screen.getByText('Todavía no hay plano')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Guardar loteo' })).toBeDisabled()

    const nameInput = screen.getByLabelText('Nombre')
    const fileInput = screen.getByLabelText('Archivo DXF')
    expect(
      nameInput.compareDocumentPosition(fileInput) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  it('draws a DXF, marks the plan as loaded and hides a layer when its toggle is turned off', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.upload(screen.getByLabelText('Archivo DXF'), planDxf())

    expect(await screen.findByText('Plano cargado')).toBeInTheDocument()
    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    expect(svg.querySelector('[aria-label="Manzana"]')).not.toBeNull()
    expect(screen.getByText('Cargado desde plano.dxf.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Manzana' }))
    expect(svg.querySelector('[aria-label="Manzana"]')).toBeNull()
    expect(svg.querySelector('[aria-label="Loteo"]')).not.toBeNull()
  })

  it('keeps the plan card optional when no DXF is loaded', () => {
    renderPage()

    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
    expect(screen.getByText('Opcional. Lo carga quien tenga el DXF.')).toBeInTheDocument()
    expect(screen.queryByRole('img', { name: 'Plano del loteo' })).not.toBeInTheDocument()
  })

  it('enables the save button only once the loteo has a name', async () => {
    const user = userEvent.setup()
    renderPage()

    expect(screen.getByRole('button', { name: 'Guardar loteo' })).toBeDisabled()
    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    expect(screen.getByRole('button', { name: 'Guardar loteo' })).toBeEnabled()
  })

  it('creates a loteo with data only and resets the form', async () => {
    const user = userEvent.setup()
    const calls = stubFetch(() => jsonResponse({ id: 'loteo-1', nombre: 'Las Acacias' }, 201))
    renderPage()

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    await user.type(screen.getByLabelText('Ubicación/Ciudad'), 'Córdoba')
    await user.click(screen.getByRole('button', { name: 'Guardar loteo' }))

    expect(await screen.findByText('Loteo creado')).toBeInTheDocument()

    expect(calls).toHaveLength(1)
    expect(calls[0].method).toBe('POST')
    expect(calls[0].url).toMatch(/\/api\/v1\/loteos$/)
    expect(calls[0].headers.get('Authorization')).toBe('Bearer test-token')
    expect(JSON.parse(String(calls[0].body))).toEqual({
      nombre: 'Las Acacias',
      ubicacion: 'Córdoba',
      descripcion: '',
      plano: null,
    })

    expect(screen.getByLabelText('Nombre')).toHaveValue('')
  })

  it('uploads the DXF after creating the loteo', async () => {
    const user = userEvent.setup()
    const calls = stubFetch((call) =>
      call.method === 'POST'
        ? jsonResponse({ id: 'loteo-1', nombre: 'Las Acacias' }, 201)
        : new Response(null, { status: 204 }),
    )
    renderPage()

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    await user.upload(screen.getByLabelText('Archivo DXF'), planDxf())
    expect(await screen.findByText('Plano cargado')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Guardar loteo' }))
    expect(await screen.findByText('Loteo creado')).toBeInTheDocument()

    expect(calls.map((call) => `${call.method} ${new URL(call.url).pathname}`)).toEqual([
      'POST /api/v1/loteos',
      'PUT /api/v1/loteos/loteo-1/dxf',
    ])
    const uploadBody = calls[1].body
    expect(uploadBody).toBeInstanceOf(FormData)
    expect((uploadBody as FormData).get('archivo')).toBeInstanceOf(File)

    const createBody = JSON.parse(String(calls[0].body))
    expect(createBody.plano.loteo.vertices).toHaveLength(4)
    expect(createBody.plano.lotes[0].manzanaRef).toBe(createBody.plano.manzanas[0].ref)
  })

  it('shows the backend message when creating the loteo fails', async () => {
    const user = userEvent.setup()
    stubFetch(() =>
      jsonResponse({ code: 'invalid_loteo_nombre', message: 'El nombre del loteo es obligatorio' }, 400),
    )
    renderPage()

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    await user.click(screen.getByRole('button', { name: 'Guardar loteo' }))

    expect(await screen.findByText('No se pudo guardar el loteo')).toBeInTheDocument()
    expect(screen.getByText('El nombre del loteo es obligatorio')).toBeInTheDocument()
  })

  it('reports a created loteo whose DXF upload failed as a warning', async () => {
    const user = userEvent.setup()
    let uploadAttempts = 0
    const calls = stubFetch((call) => {
      if (call.method === 'POST') {
        return jsonResponse({ id: 'loteo-1', nombre: 'Las Acacias' }, 201)
      }
      uploadAttempts++
      return uploadAttempts === 1
        ? jsonResponse(
            { code: 'storage_unavailable', message: 'El almacenamiento no está disponible' },
            503,
          )
        : new Response(null, { status: 204 })
    })
    renderPage()

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    await user.upload(screen.getByLabelText('Archivo DXF'), planDxf())
    expect(await screen.findByText('Plano cargado')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Guardar loteo' }))

    expect(await screen.findByText('Loteo creado')).toBeInTheDocument()
    expect(screen.getByText(/no se pudo guardar el archivo DXF/i)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Reintentar carga del DXF' }))

    expect(await screen.findByText('Ya podés cargar otro loteo.')).toBeInTheDocument()
    expect(calls.map((call) => call.method)).toEqual(['POST', 'PUT', 'PUT'])
  })

  it('blocks the save when the plan has no LOTEO layer', async () => {
    const user = userEvent.setup()
    const calls = stubFetch(() => jsonResponse({ id: 'loteo-1', nombre: 'x' }, 201))
    renderPage()

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    const file = new File([dxfDocument(lwpolyline('LOTES', square))], 'plano.dxf', {
      type: 'application/dxf',
    })
    await user.upload(screen.getByLabelText('Archivo DXF'), file)
    expect(await screen.findByText('Plano cargado')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Guardar loteo' }))

    expect(await screen.findByText('No se pudo guardar el loteo')).toBeInTheDocument()
    expect(screen.getByText(/capa LOTEO/)).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('blocks the save when the plan has more than one LOTEO polygon', async () => {
    const user = userEvent.setup()
    const calls = stubFetch(() => jsonResponse({ id: 'loteo-1', nombre: 'x' }, 201))
    renderPage()

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    const file = new File(
      [dxfDocument(lwpolyline('LOTEO', square), lwpolyline('LOTEO', square))],
      'plano.dxf',
      { type: 'application/dxf' },
    )
    await user.upload(screen.getByLabelText('Archivo DXF'), file)
    await user.click(screen.getByRole('button', { name: 'Guardar loteo' }))

    expect(await screen.findByText('No se pudo guardar el loteo')).toBeInTheDocument()
    expect(screen.getByText(/único polígono/)).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('asks the user to sign in again when there is no session', async () => {
    const user = userEvent.setup()
    const calls = stubFetch(() => jsonResponse({}, 201))
    renderPage({ token: null })

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    await user.click(screen.getByRole('button', { name: 'Guardar loteo' }))

    expect(await screen.findByText(/Tu sesión expiró/)).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('shows an error when the DXF has no usable polygons', async () => {
    const user = userEvent.setup()
    renderPage()

    const file = new File(['0\nSECTION\n2\nENTITIES\n0\nENDSEC\n0\nEOF\n'], 'vacio.dxf', {
      type: 'application/dxf',
    })
    await user.upload(screen.getByLabelText('Archivo DXF'), file)

    expect(await screen.findByRole('alert')).toHaveTextContent('No se pudo leer el DXF')
    expect(screen.getByText('Todavía no hay plano')).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
  })

  it('shows a warning when the DXF has overlapping lots', async () => {
    const user = userEvent.setup()
    renderPage()

    const overlapping = dxfDocument(
      lwpolyline('LOTES', square),
      lwpolyline('LOTES', [
        [5, 5],
        [15, 5],
        [15, 15],
        [5, 15],
      ]),
    )
    const file = new File([overlapping], 'solape.dxf', { type: 'application/dxf' })
    await user.upload(screen.getByLabelText('Archivo DXF'), file)

    expect(await screen.findByRole('alert')).toHaveTextContent('Avisos de geometría')
    expect(screen.getByRole('img', { name: 'Plano del loteo' })).toBeInTheDocument()
  })

  it('discards the entered data and the loaded plan', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    expect(screen.getByRole('button', { name: 'Seleccionar todas' })).toBeDisabled()
    await user.upload(screen.getByLabelText('Archivo DXF'), planDxf())
    expect(await screen.findByText('Plano cargado')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Descartar' }))

    expect(screen.getByLabelText('Nombre')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Seleccionar todas' })).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
    expect(screen.getByText('Todavía no hay plano')).toBeInTheDocument()
  })
})
