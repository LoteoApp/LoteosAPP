import { afterEach, describe, expect, it, vi } from 'vitest'
import { newIdempotencyKey } from './idempotencyKey'

afterEach(() => {
	vi.restoreAllMocks()
	vi.unstubAllGlobals()
})

describe('newIdempotencyKey', () => {
	it('uses the platform UUID generator when available', () => {
		const randomUUID = vi.fn().mockReturnValue('reservation-key')
		vi.stubGlobal('crypto', { randomUUID })

		expect(newIdempotencyKey()).toBe('reservation-key')
		expect(randomUUID).toHaveBeenCalledOnce()
	})

	it('falls back to a timestamp and random value when UUID is unavailable', () => {
		vi.stubGlobal('crypto', { randomUUID: undefined })
		vi.spyOn(Date, 'now').mockReturnValue(123)
		vi.spyOn(Math, 'random').mockReturnValue(0.5)

		expect(newIdempotencyKey()).toBe('123-0.5')
	})
})
