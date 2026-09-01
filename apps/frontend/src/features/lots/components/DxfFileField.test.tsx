import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import DxfFileField from './DxfFileField'
import * as readDxfFileModule from '../lib/readDxfFile'

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

describe('DxfFileField', () => {
  it('parses a DXF and reports the file name', async () => {
    const user = userEvent.setup()
    const onParsed = vi.fn()
    const onError = vi.fn()
    const onCleared = vi.fn()

    render(
      <DxfFileField
        fileName={null}
        onParsed={onParsed}
        onError={onError}
        onCleared={onCleared}
      />,
    )

    const file = new File(
      [dxfDocument(lwpolyline('LOTES', square))],
      'san-pedro.dxf',
      { type: 'application/dxf' },
    )
    await user.upload(screen.getByLabelText('Archivo DXF'), file)

    expect(onError).not.toHaveBeenCalled()
    expect(onParsed).toHaveBeenCalledTimes(1)
    const [result, parsedFile] = onParsed.mock.calls[0]
    expect(parsedFile).toBeInstanceOf(File)
    expect(parsedFile.name).toBe('san-pedro.dxf')
    expect(result.polygons).toHaveLength(1)
  })

  it('maps unexpected parse failures to a generic message', async () => {
    const user = userEvent.setup()
    vi.spyOn(readDxfFileModule, 'readDxfFile').mockRejectedValueOnce(new Error('boom'))
    const onError = vi.fn()

    render(
      <DxfFileField
        fileName={null}
        onParsed={vi.fn()}
        onError={onError}
        onCleared={vi.fn()}
      />,
    )

    const file = new File(['0\nEOF\n'], 'plano.dxf', { type: 'application/dxf' })
    await user.upload(screen.getByLabelText('Archivo DXF'), file)

    expect(onError).toHaveBeenCalledWith('No se pudo interpretar el archivo DXF.')
  })

  it('calls onCleared when the file input is emptied', async () => {
    const onCleared = vi.fn()
    render(
      <DxfFileField
        fileName="plano.dxf"
        onParsed={vi.fn()}
        onError={vi.fn()}
        onCleared={onCleared}
      />,
    )

    fireEvent.change(screen.getByLabelText('Archivo DXF'), {
      target: { files: [] },
    })

    expect(onCleared).toHaveBeenCalled()
  })

  it('rejects a file that is not a DXF', async () => {
    const user = userEvent.setup()
    const onParsed = vi.fn()
    const onError = vi.fn()

    render(
      <DxfFileField
        fileName={null}
        onParsed={onParsed}
        onError={onError}
        onCleared={vi.fn()}
      />,
    )

    const file = new File(['not a dxf document'], 'roto.dxf', {
      type: 'application/dxf',
    })
    await user.upload(screen.getByLabelText('Archivo DXF'), file)

    expect(onParsed).not.toHaveBeenCalled()
    expect(onError).toHaveBeenCalledWith(
      'No se encontraron polígonos cerrados en las capas LOTEO, MANZANA, LOTES o CALLE.',
    )
  })
})
