import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import AgencyCombobox from './AgencyCombobox'
import type { AgencyOption } from '../types'

const agencies: AgencyOption[] = [
  { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
  { id: 'ag-2', razonSocial: 'Norte Propiedades' },
]

describe('AgencyCombobox', () => {
  it('selects an inmobiliaria from the list', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()

    render(<AgencyCombobox agencies={agencies} value={null} onChange={onChange} />)

    const input = screen.getByLabelText('Inmobiliaria')
    await user.click(input)
    await user.click(await screen.findByRole('option', { name: 'Norte Propiedades' }))

    expect(onChange.mock.calls[0][0]).toEqual(agencies[1])
  })

  it('filters the options by what the user types', async () => {
    const user = userEvent.setup()

    render(<AgencyCombobox agencies={agencies} value={null} onChange={vi.fn()} />)

    const input = screen.getByLabelText('Inmobiliaria')
    await user.click(input)
    await user.type(input, 'Sur')

    expect(await screen.findByRole('option', { name: 'Inmobiliaria Sur' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Norte Propiedades' })).not.toBeInTheDocument()
  })

  it('shows the selected inmobiliaria', () => {
    render(<AgencyCombobox agencies={agencies} value={agencies[0]} onChange={vi.fn()} />)

    expect(screen.getByLabelText('Inmobiliaria')).toHaveValue('Inmobiliaria Sur')
  })

  it('waits while the inmobiliarias are loading', () => {
    render(<AgencyCombobox agencies={[]} value={null} onChange={vi.fn()} isLoading />)

    const input = screen.getByLabelText('Inmobiliaria')
    expect(input).toBeDisabled()
    expect(input).toHaveAttribute('placeholder', 'Cargando inmobiliarias…')
  })
})
