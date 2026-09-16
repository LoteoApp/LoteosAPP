import { act, render, renderHook, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  SaveNotice,
  SAVE_NOTICE_FADE_MS,
  SAVE_NOTICE_MS,
  useSaveNotice,
} from './save-notice'

describe('SaveNotice', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('stays hidden until a token is shown', () => {
    const { rerender } = render(<SaveNotice token={0}>Lote guardado</SaveNotice>)
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()

    rerender(<SaveNotice token={1}>Lote guardado</SaveNotice>)
    expect(screen.getByRole('alert')).toHaveTextContent('Lote guardado')
  })

  it('fades out after two seconds and then leaves the document', () => {
    vi.useFakeTimers()
    render(<SaveNotice token={1}>Lote guardado</SaveNotice>)

    act(() => {
      vi.advanceTimersByTime(SAVE_NOTICE_MS - 1)
    })
    expect(screen.getByRole('alert')).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(screen.getByRole('alert')).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(SAVE_NOTICE_FADE_MS)
    })
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})

describe('useSaveNotice', () => {
  it('issues a new token on each show and clears it', () => {
    const { result } = renderHook(() => useSaveNotice())

    expect(result.current.token).toBe(0)

    act(() => {
      result.current.show()
    })
    expect(result.current.token).toBe(1)

    act(() => {
      result.current.show()
    })
    expect(result.current.token).toBe(2)

    act(() => {
      result.current.clear()
    })
    expect(result.current.token).toBe(0)
  })
})
