import { describe, expect, it } from 'vitest'
import { entityEquals, type PlanEntityRef } from './types'

describe('entityEquals', () => {
  it('compares loteo refs without an id', () => {
    expect(entityEquals({ kind: 'loteo' }, { kind: 'loteo' })).toBe(true)
    expect(entityEquals({ kind: 'loteo' }, { kind: 'lote', id: 'lt-1' })).toBe(false)
  })

  it('compares manzana, lote and calle refs by id', () => {
    const lote: PlanEntityRef = { kind: 'lote', id: 'lt-1' }
    expect(entityEquals(lote, { kind: 'lote', id: 'lt-1' })).toBe(true)
    expect(entityEquals(lote, { kind: 'lote', id: 'lt-2' })).toBe(false)
    expect(entityEquals(lote, { kind: 'manzana', id: 'lt-1' })).toBe(false)
  })
})
