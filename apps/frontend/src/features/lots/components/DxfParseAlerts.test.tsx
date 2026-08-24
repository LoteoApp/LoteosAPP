import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import DxfParseAlerts from './DxfParseAlerts'
import type { DxfValidationIssue } from '../types'

describe('DxfParseAlerts', () => {
  it('shows a parse error', () => {
    render(<DxfParseAlerts error="El archivo debe tener extensión .dxf." issues={[]} />)

    expect(screen.getByRole('alert')).toHaveTextContent('No se pudo leer el DXF')
    expect(screen.getByRole('alert')).toHaveTextContent(
      'El archivo debe tener extensión .dxf.',
    )
  })

  it('lists geometry warnings', () => {
    const issues: DxfValidationIssue[] = [
      {
        code: 'OVERLAPPING',
        layer: 'LOTES',
        message: 'Dos polígonos de la capa LOTES se superponen.',
        handle: null,
        polygonId: null,
        relatedPolygonId: 'b',
      },
    ]

    render(<DxfParseAlerts error={null} issues={issues} />)

    expect(screen.getByRole('alert')).toHaveTextContent('Avisos de geometría')
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Dos polígonos de la capa LOTES se superponen.',
    )
  })

  it('renders nothing when there is nothing to report', () => {
    render(<DxfParseAlerts error={null} issues={[]} />)

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
