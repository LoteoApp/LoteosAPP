import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import DxfLayerToggles from './DxfLayerToggles'
import { DXF_LAYERS } from '../types'

describe('DxfLayerToggles', () => {
  it('turns a layer off', async () => {
    const user = userEvent.setup()
    const onVisibleLayersChange = vi.fn()

    render(
      <DxfLayerToggles
        visibleLayers={new Set(DXF_LAYERS)}
        onVisibleLayersChange={onVisibleLayersChange}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Manzana' }))

    const next = onVisibleLayersChange.mock.calls.at(-1)?.[0] as Set<string>
    expect(next.has('MANZANA')).toBe(false)
    expect(next.has('LOTES')).toBe(true)
  })
})
