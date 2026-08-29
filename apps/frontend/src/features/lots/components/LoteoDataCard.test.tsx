import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import LoteoDataCard from './LoteoDataCard'
import type { LoteoFieldValues } from './LoteoFields'

const empty: LoteoFieldValues = {
  name: '',
  location: '',
  description: '',
  agencyIds: [],
}

function Harness() {
  const [values, setValues] = useState(empty)
  return <LoteoDataCard values={values} onChange={setValues} />
}

describe('LoteoDataCard', () => {
  it('renders the card title and the loteo fields', async () => {
    const user = userEvent.setup()
    render(<Harness />)

    expect(screen.getByText('Datos del loteo')).toBeInTheDocument()

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')

    expect(screen.getByLabelText('Nombre')).toHaveValue('Las Acacias')
  })
})
