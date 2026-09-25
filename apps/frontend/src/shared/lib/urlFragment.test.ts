import { afterEach, describe, expect, it } from 'vitest'
import { clearURLFragment, readFragmentParams } from './urlFragment'

describe('readFragmentParams', () => {
  afterEach(() => {
    window.history.replaceState(null, '', '/')
  })

  it('reads params from the URL fragment', () => {
    window.history.replaceState(null, '', '/#token=abc123&type=invite')

    const params = readFragmentParams()

    expect(params.get('token')).toBe('abc123')
    expect(params.get('type')).toBe('invite')
  })

  it('returns an empty result when there is no fragment', () => {
    window.history.replaceState(null, '', '/restablecer-contrasena')

    expect(readFragmentParams().get('token')).toBeNull()
  })
})

describe('clearURLFragment', () => {
  afterEach(() => {
    window.history.replaceState(null, '', '/')
  })

  it('drops the fragment while keeping the path and query', () => {
    window.history.replaceState(null, '', '/restablecer-contrasena?foo=bar#token=abc123')

    clearURLFragment()

    expect(window.location.hash).toBe('')
    expect(window.location.pathname).toBe('/restablecer-contrasena')
    expect(window.location.search).toBe('?foo=bar')
  })
})
