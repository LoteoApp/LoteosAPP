import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import LoteoSummaryPanel from './LoteoSummaryPanel'
import type { LoteoDetail, LoteoLote } from '../types'

function lote(overrides: Partial<LoteoLote> = {}): LoteoLote {
  return {
    id: 'lt-1',
    manzanaId: 'mz-1',
    numero: '1',
    estado: 'disponible',
    precio: null,
    moneda: 'USD',
    superficie: null,
    caracteristicas: '',
    poligono: [],
    ...overrides,
  }
}

function detail(overrides: Partial<LoteoDetail> = {}): LoteoDetail {
  return {
    id: 'loteo-1',
    nombre: 'Las Acacias',
    ubicacion: 'Río Ceballos, Córdoba',
    descripcion: 'Sobre ruta E-53.',
    contorno: [],
    manzanas: [
      { id: 'mz-1', numero: '1', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: [] },
      { id: 'mz-2', numero: '2', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: [] },
    ],
    lotes: [
      lote({ id: 'lt-1' }),
      lote({ id: 'lt-2', estado: 'reservado' }),
      lote({ id: 'lt-3', estado: 'reservado' }),
      lote({ id: 'lt-4', estado: 'vendido' }),
    ],
    calles: [{ id: 'ca-1', nombre: 'Los Álamos', tipo: 'asfalto', poligono: [] }],
    fechaCreacion: '2026-08-20T12:00:00Z',
    ...overrides,
  }
}

describe('LoteoSummaryPanel', () => {
  it('counts the lotes by state', () => {
    render(<LoteoSummaryPanel loteo={detail()} />)

    expect(screen.getByText('4 lotes en total.')).toBeInTheDocument()
    expect(screen.getByLabelText('1 Disponible')).toBeInTheDocument()
    expect(screen.getByLabelText('2 Reservado')).toBeInTheDocument()
    expect(screen.getByLabelText('1 Vendido')).toBeInTheDocument()
    expect(screen.getByLabelText('0 Finalizado')).toBeInTheDocument()
  })

  it('shows how many manzanas and calles the loteo has', () => {
    render(<LoteoSummaryPanel loteo={detail()} />)

    expect(screen.getByText('Manzanas').nextSibling).toHaveTextContent('2')
    expect(screen.getByText('Calles').nextSibling).toHaveTextContent('1')
    expect(screen.getByText('Alta').nextSibling).not.toHaveTextContent('—')
  })

  it('tells the user when there are no lotes yet', () => {
    render(<LoteoSummaryPanel loteo={detail({ lotes: [] })} />)

    expect(screen.getByText('Este loteo todavía no tiene lotes cargados.')).toBeInTheDocument()
    expect(screen.queryByRole('list', { name: 'Lotes por estado' })).not.toBeInTheDocument()
  })
})
