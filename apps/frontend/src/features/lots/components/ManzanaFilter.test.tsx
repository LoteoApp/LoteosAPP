import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import ManzanaFilter, { ALL_MANZANAS } from './ManzanaFilter'
import type { LoteoManzana } from '../types'

function manzana(id: string, numero: string): LoteoManzana {
  return { id, numero, poligono: [] }
}

const manzanas = [manzana('mz-1', '1'), manzana('mz-2', '2'), manzana('mz-3', '3')]

describe('ManzanaFilter', () => {
  it('reports the manzana the user picks', async () => {
    const onChange = vi.fn()
    render(<ManzanaFilter manzanas={manzanas} value={ALL_MANZANAS} onChange={onChange} />)

    await userEvent.selectOptions(screen.getByLabelText('Manzana'), 'mz-2')

    expect(onChange).toHaveBeenCalledWith('mz-2')
  })

  it('offers a "Todas las manzanas" option that clears the filter', async () => {
    const onChange = vi.fn()
    render(<ManzanaFilter manzanas={manzanas} value="mz-2" onChange={onChange} />)

    await userEvent.selectOptions(screen.getByLabelText('Manzana'), 'Todas las manzanas')

    expect(onChange).toHaveBeenCalledWith(ALL_MANZANAS)
  })

  it('does not render when there is a single manzana', () => {
    const { container } = render(
      <ManzanaFilter manzanas={[manzana('mz-1', '1')]} value={ALL_MANZANAS} onChange={vi.fn()} />,
    )

    expect(container).toBeEmptyDOMElement()
  })

  it('labels a manzana with no number as "Sin número"', () => {
    render(
      <ManzanaFilter
        manzanas={[manzana('mz-1', '1'), manzana('mz-2', '')]}
        value={ALL_MANZANAS}
        onChange={vi.fn()}
      />,
    )

    expect(screen.getByRole('option', { name: 'Sin número' })).toBeInTheDocument()
  })
})
