import { useCallback, useEffect, useState } from 'react'
import { CircleCheck } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { Alert, AlertTitle } from '@/shared/ui/alert'

export const SAVE_NOTICE_MS = 2_000
export const SAVE_NOTICE_FADE_MS = 300

export function useSaveNotice() {
  const [token, setToken] = useState(0)

  const show = useCallback(() => {
    setToken((current) => current + 1)
  }, [])

  const clear = useCallback(() => {
    setToken(0)
  }, [])

  return { token, show, clear }
}

type SaveNoticeProps = {
  token: number
  children: string
}

type Phase = 'enter' | 'in' | 'out' | 'gone'

export function SaveNotice({ token, children }: SaveNoticeProps) {
  if (token === 0) {
    return null
  }

  return <SaveNoticeBanner key={token}>{children}</SaveNoticeBanner>
}

function SaveNoticeBanner({ children }: { children: string }) {
  const [phase, setPhase] = useState<Phase>('enter')

  useEffect(() => {
    const enter = window.requestAnimationFrame(() => setPhase('in'))
    const hide = window.setTimeout(() => setPhase('out'), SAVE_NOTICE_MS)
    return () => {
      window.cancelAnimationFrame(enter)
      window.clearTimeout(hide)
    }
  }, [])

  useEffect(() => {
    if (phase !== 'out') {
      return
    }
    const done = window.setTimeout(() => setPhase('gone'), SAVE_NOTICE_FADE_MS)
    return () => window.clearTimeout(done)
  }, [phase])

  if (phase === 'gone') {
    return null
  }

  const open = phase === 'in'

  return (
    <div
      className={cn(
        'grid overflow-hidden transition-[grid-template-rows,opacity] duration-300 ease-out motion-reduce:transition-none',
        open ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0',
      )}
    >
      <div className="min-h-0 overflow-hidden">
        <Alert>
          <CircleCheck aria-hidden />
          <AlertTitle>{children}</AlertTitle>
        </Alert>
      </div>
    </div>
  )
}
