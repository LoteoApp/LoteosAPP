import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import type { Inmobiliaria } from '../api/list-inmobiliarias'
import InmobiliariasField from './InmobiliariasField'

const catalog: Inmobiliaria[] = [
  { id: 'inm-1', razonSocial: 'Inmobiliaria San Martín' },
  { id: 'inm-2', razonSocial: 'Lotes del Sur' },
  { id: 'inm-3', razonSocial: 'Altamira Propiedades' },
]

function Harness({ initialIds = [] as string[] }) {
  const [selectedIds, setSelectedIds] = useState<string[]>(initialIds)
  return (
    <InmobiliariasField
      inmobiliarias={catalog}
      selectedIds={selectedIds}
      onChange={setSelectedIds}
    />
  )
}

async function openList(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByLabelText('Inmobiliarias'))
}

describe('InmobiliariasField', () => {
  it('lets the user search by name and pick more than one inmobiliaria', async () => {
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

  it('selects every inmobiliaria and then clears the selection', async () => {
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

  it('removes a selected inmobiliaria from its chip', async () => {
    const user = userEvent.setup()
    render(<Harness initialIds={['inm-2']} />)

    await user.click(screen.getByRole('button', { name: 'Quitar Lotes del Sur' }))

    expect(screen.queryByLabelText('Lotes del Sur')).not.toBeInTheDocument()
  })

  it('disables select all when there are no inmobiliarias', () => {
    function EmptyHarness() {
      const [selectedIds, setSelectedIds] = useState<string[]>([])
      return (
        <InmobiliariasField
          inmobiliarias={[]}
          selectedIds={selectedIds}
          onChange={setSelectedIds}
        />
      )
    }

    render(<EmptyHarness />)

    expect(screen.getByRole('button', { name: 'Seleccionar todas' })).toBeDisabled()
  })
})
