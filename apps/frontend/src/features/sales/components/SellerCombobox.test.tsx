import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import SellerCombobox from './SellerCombobox'
import type { SellerOption } from '../types'

const sellers: SellerOption[] = [
  {
    id: 'us-1',
    nombre: 'Marta',
    apellido: 'Suárez',
    rol: 'inmobiliaria',
    inmobiliariaId: 'ag-1',
    inmobiliariaRazonSocial: 'Inmobiliaria Sur',
  },
  { id: 'us-2', nombre: 'Sofía', apellido: 'Luna', rol: 'administrativo' },
]

describe('SellerCombobox', () => {
  it('selects a vendedor from the list', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()

    render(<SellerCombobox sellers={sellers} value={null} onChange={onChange} />)

    await user.click(screen.getByLabelText('Vendedor'))
    await user.click(await screen.findByRole('option', { name: 'Luna, Sofía' }))

    expect(onChange.mock.calls[0][0]).toEqual(sellers[1])
  })

  it('filters the options by what the user types', async () => {
    const user = userEvent.setup()

    render(<SellerCombobox sellers={sellers} value={null} onChange={vi.fn()} />)

    const input = screen.getByLabelText('Vendedor')
    await user.click(input)
    await user.type(input, 'Suárez')

    expect(await screen.findByRole('option', { name: 'Suárez, Marta' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Luna, Sofía' })).not.toBeInTheDocument()
  })

  it('shows the selected vendedor', () => {
    render(<SellerCombobox sellers={sellers} value={sellers[0]} onChange={vi.fn()} />)

    expect(screen.getByLabelText('Vendedor')).toHaveValue('Suárez, Marta')
  })

  it('waits while the vendedores are loading', () => {
    render(<SellerCombobox sellers={[]} value={null} onChange={vi.fn()} isLoading />)

    const input = screen.getByLabelText('Vendedor')
    expect(input).toBeDisabled()
    expect(input).toHaveAttribute('placeholder', 'Cargando vendedores…')
  })

  it('stays disabled until an inmobiliaria narrows the list', () => {
    render(<SellerCombobox sellers={[]} value={null} onChange={vi.fn()} disabled />)

    expect(screen.getByLabelText('Vendedor')).toBeDisabled()
  })
})
