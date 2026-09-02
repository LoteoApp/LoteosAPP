import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import LoteoPageHeader from './LoteoPageHeader'

function renderHeader(overrides: Partial<Parameters<typeof LoteoPageHeader>[0]> = {}) {
  const props = {
    hasPlan: false,
    canSave: true,
    isSaving: false,
    onSave: vi.fn(),
    onDiscard: vi.fn(),
    ...overrides,
  }
  render(
    <MemoryRouter>
      <LoteoPageHeader {...props} />
    </MemoryRouter>,
  )
  return props
}

describe('LoteoPageHeader', () => {
  it('shows the plan status badge', () => {
    renderHeader({ hasPlan: false })

    expect(screen.getByRole('heading', { name: 'Nuevo loteo' })).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
  })

  it('links back to the loteo list', () => {
    renderHeader()

    expect(screen.getByRole('link', { name: 'Volver al listado' })).toHaveAttribute(
      'href',
      '/lotes',
    )
  })

  it('disables the save button until the loteo can be saved', () => {
    renderHeader({ canSave: false })

    expect(screen.getByRole('button', { name: 'Guardar loteo' })).toBeDisabled()
  })

  it('calls onSave when the enabled save button is clicked', async () => {
    const user = userEvent.setup()
    const { onSave } = renderHeader({ canSave: true })

    await user.click(screen.getByRole('button', { name: 'Guardar loteo' }))

    expect(onSave).toHaveBeenCalledTimes(1)
  })

  it('shows a saving label and blocks both actions while saving', () => {
    renderHeader({ canSave: true, isSaving: true })

    expect(screen.getByRole('button', { name: 'Guardando…' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Descartar' })).toBeDisabled()
  })

  it('calls onDiscard when the user clicks Descartar', async () => {
    const user = userEvent.setup()
    const { onDiscard } = renderHeader()

    await user.click(screen.getByRole('button', { name: 'Descartar' }))

    expect(onDiscard).toHaveBeenCalledTimes(1)
  })
})
