import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import type { Agency } from '../api/list-agencies'
import AgenciesField from './AgenciesField'

const catalog: Agency[] = [
  { id: 'inm-1', businessName: 'Inmobiliaria San Martín' },
  { id: 'inm-2', businessName: 'Lotes del Sur' },
  { id: 'inm-3', businessName: 'Altamira Propiedades' },
]

function Harness({ initialIds = [] as string[] }) {
  const [selectedIds, setSelectedIds] = useState<string[]>(initialIds)
  return (
    <AgenciesField
      agencies={catalog}
      selectedIds={selectedIds}
      onChange={setSelectedIds}
    />
  )
}

async function openList(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByLabelText('Inmobiliarias'))
}

describe('AgenciesField', () => {
  it('lets the user search by name and pick more than one agency', async () => {
    const user = userEvent.setup()
    render(<Harness />)

    await openList(user)
    await user.type(screen.getByLabelText('Inmobiliarias'), 'Lotes')

    expect(screen.getByRole('option', { name: 'Lotes del Sur' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Altamira Propiedades' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('option', { name: 'Lotes del Sur' }))
    await user.clear(screen.getByLabelText('Inmobiliarias'))
    await user.type(screen.getByLabelText('Inmobiliarias'), 'Altamira')
    await user.click(screen.getByRole('option', { name: 'Altamira Propiedades' }))

    expect(screen.getByLabelText('Lotes del Sur')).toBeInTheDocument()
    expect(screen.getByLabelText('Altamira Propiedades')).toBeInTheDocument()
  })

  it('selects every agency and then clears the selection', async () => {
    const user = userEvent.setup()
    render(<Harness />)

    await user.click(screen.getByRole('button', { name: 'Seleccionar todas' }))

    expect(screen.getByLabelText('Inmobiliaria San Martín')).toBeInTheDocument()
    expect(screen.getByLabelText('Lotes del Sur')).toBeInTheDocument()
    expect(screen.getByLabelText('Altamira Propiedades')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Quitar todas' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Quitar todas' }))

    expect(screen.queryByLabelText('Lotes del Sur')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Seleccionar todas' })).toBeInTheDocument()
  })

  it('shows an empty message when the search has no matches', async () => {
    const user = userEvent.setup()
    render(<Harness />)

    await openList(user)
    await user.type(screen.getByLabelText('Inmobiliarias'), 'zzzz')

    expect(screen.getByText('No hay inmobiliarias con ese nombre.')).toBeInTheDocument()
  })

  it('removes a selected agency from its chip', async () => {
    const user = userEvent.setup()
    render(<Harness initialIds={['inm-2']} />)

    await user.click(screen.getByRole('button', { name: 'Quitar Lotes del Sur' }))

    expect(screen.queryByLabelText('Lotes del Sur')).not.toBeInTheDocument()
  })

  it('disables select all when there are no agencies', () => {
    function EmptyHarness() {
      const [selectedIds, setSelectedIds] = useState<string[]>([])
      return (
        <AgenciesField
          agencies={[]}
          selectedIds={selectedIds}
          onChange={setSelectedIds}
        />
      )
    }

    render(<EmptyHarness />)

    expect(screen.getByRole('button', { name: 'Seleccionar todas' })).toBeDisabled()
  })
})
