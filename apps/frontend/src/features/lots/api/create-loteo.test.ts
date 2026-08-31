import { describe, expect, it, vi } from 'vitest'
import { createLoteo } from './create-loteo'
import * as client from '../../../shared/api/client'
import type { CreateLoteoPayload } from '../lib/buildCreateLoteoPayload'

const payload: CreateLoteoPayload = {
  nombre: 'Las Acacias',
  ubicacion: 'Córdoba',
  descripcion: '',
  plano: null,
}

describe('createLoteo', () => {
  it('POSTs the payload to the loteos endpoint with the token', async () => {
    const apiFetch = vi
      .spyOn(client, 'apiFetch')
      .mockResolvedValue({ id: 'loteo-1', nombre: 'Las Acacias' })

    const result = await createLoteo(payload, 'tok')

    expect(result).toEqual({ id: 'loteo-1', nombre: 'Las Acacias' })
    expect(apiFetch).toHaveBeenCalledWith('/api/v1/loteos', {
      method: 'POST',
      body: payload,
      token: 'tok',
    })
  })
})
