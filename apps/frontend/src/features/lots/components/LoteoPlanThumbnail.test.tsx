import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import LoteoPlanThumbnail from './LoteoPlanThumbnail'

describe('LoteoPlanThumbnail', () => {
  it('shows a "Sin plano" placeholder when there is no plan', () => {
    render(<LoteoPlanThumbnail hasPlan={false} />)

    expect(screen.getByText('Sin plano')).toBeInTheDocument()
  })

  it('does not show the placeholder when there is a plan', () => {
    render(<LoteoPlanThumbnail hasPlan />)

    expect(screen.queryByText('Sin plano')).not.toBeInTheDocument()
  })
})
