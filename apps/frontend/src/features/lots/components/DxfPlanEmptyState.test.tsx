import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import DxfPlanEmptyState from './DxfPlanEmptyState'

describe('DxfPlanEmptyState', () => {
  it('explains that the plan can be uploaded later', () => {
    render(<DxfPlanEmptyState />)

    expect(screen.getByText('Todavía no hay plano')).toBeInTheDocument()
    expect(screen.getByText(/El agrimensor puede subir el DXF más adelante/)).toBeInTheDocument()
  })
})
