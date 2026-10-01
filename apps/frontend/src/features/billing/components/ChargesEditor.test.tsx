import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import ChargesEditor from './ChargesEditor'
import type { ChargeRow } from '../types'

function renderEditor(initial: ChargeRow[]) {
  const onChange = vi.fn()
  function Harness() {
    const [rows, setRows] = useState(initial)
    return (
      <ChargesEditor
        rows={rows}
        currency="USD"
        onChange={(next) => {
          onChange(next)
          setRows(next)
        }}
      />
    )
  }
  render(<Harness />)
  return { onChange }
}

describe('ChargesEditor', () => {
  it('shows the charge type by its readable name while keeping its id as the value', async () => {
    const user = userEvent.setup()
    const { onChange } = renderEditor([
      { key: 'cargo-a', tipo: 'impuesto_provincial', monto: '', moneda: 'USD', detalle: '' },
    ])

    const type = screen.getByRole('combobox', { name: 'Tipo' })
    expect(type).toHaveTextContent('Impuesto provincial')
    expect(type).not.toHaveTextContent('impuesto_provincial')

    await user.click(type)
    expect(await screen.findByRole('option', { name: 'Gasto administrativo' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Cargo de inmobiliaria' })).toBeInTheDocument()
    await user.click(screen.getByRole('option', { name: 'Gasto administrativo' }))

    expect(screen.getByRole('combobox', { name: 'Tipo' })).toHaveTextContent('Gasto administrativo')
    expect(onChange).toHaveBeenLastCalledWith([expect.objectContaining({ key: 'cargo-a', tipo: 'gasto_administrativo' })])
  })

  it('keeps the remove action named for screen readers', async () => {
    const user = userEvent.setup()
    const { onChange } = renderEditor([
      { key: 'cargo-a', tipo: 'servicios', monto: '10', moneda: 'USD', detalle: '' },
    ])

    await user.click(screen.getByRole('button', { name: 'Quitar el cargo 1' }))

    expect(onChange).toHaveBeenLastCalledWith([])
    expect(screen.queryByRole('combobox', { name: 'Tipo' })).not.toBeInTheDocument()
  })
})
