import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import LoteoFields, { type LoteoFieldValues } from './LoteoFields'

const empty: LoteoFieldValues = {
  name: '',
  location: '',
  description: '',
}

function Harness() {
  const [values, setValues] = useState(empty)
  return <LoteoFields values={values} onChange={setValues} />
}

describe('LoteoFields', () => {
  it('lets the user fill the available fields and keeps mock agencies disabled', async () => {
    const user = userEvent.setup()
    render(<Harness />)

    await user.type(screen.getByLabelText('Nombre'), 'San Pedro')
    await user.type(screen.getByLabelText('Ubicación/Ciudad'), 'Paraná')
    await user.type(screen.getByLabelText('Descripción'), 'Frente al río')

    expect(screen.getByLabelText('Nombre')).toHaveValue('San Pedro')
    expect(screen.getByLabelText('Ubicación/Ciudad')).toHaveValue('Paraná')
    expect(screen.getByRole('button', { name: 'Seleccionar todas' })).toBeDisabled()
    expect(screen.getByLabelText('Inmobiliarias')).toBeDisabled()
    expect(screen.getByText(/catálogo de inmobiliarias esté conectado/i)).toBeInTheDocument()
    expect(screen.getByLabelText('Descripción')).toHaveValue('Frente al río')
  })
})
