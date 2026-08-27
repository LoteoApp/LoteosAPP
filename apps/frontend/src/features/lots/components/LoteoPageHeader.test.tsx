import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import LoteoPageHeader from './LoteoPageHeader'

describe('LoteoPageHeader', () => {
  it('shows the plan status badge', () => {
    render(<LoteoPageHeader hasPlan={false} onDiscard={vi.fn()} />)

    expect(screen.getByRole('heading', { name: 'Nuevo loteo' })).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
  })

  it('keeps the save button disabled', () => {
    render(<LoteoPageHeader hasPlan onDiscard={vi.fn()} />)

    expect(screen.getByRole('button', { name: 'Guardar loteo' })).toBeDisabled()
  })

  it('calls onDiscard when the user clicks Descartar', async () => {
    const user = userEvent.setup()
    const onDiscard = vi.fn()
    render(<LoteoPageHeader hasPlan={false} onDiscard={onDiscard} />)

    await user.click(screen.getByRole('button', { name: 'Descartar' }))

    expect(onDiscard).toHaveBeenCalledTimes(1)
  })
})
