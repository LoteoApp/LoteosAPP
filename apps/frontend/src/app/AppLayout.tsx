import { useEffect, useState } from 'react'
import { ArrowLeftToLine, Menu } from 'lucide-react'
import { Outlet } from 'react-router'
import { Button } from '../shared/ui/button'
import Sidebar from './Sidebar'
import UserMenu from './UserMenu'

function getDesktopMediaQuery() {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return null
  }

  const breakpoint = getComputedStyle(document.documentElement)
    .getPropertyValue('--breakpoint-md')
    .trim()

  return window.matchMedia(`(min-width: ${breakpoint || '768px'})`)
}

function isDesktopViewport() {
  return getDesktopMediaQuery()?.matches ?? false
}

export default function AppLayout() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(isDesktopViewport)

  useEffect(() => {
    const mediaQuery = getDesktopMediaQuery()

    if (!mediaQuery) {
      return
    }

    function handleChange(event: MediaQueryListEvent) {
      setIsSidebarOpen(event.matches)
    }

    mediaQuery.addEventListener('change', handleChange)

    return () => {
      mediaQuery.removeEventListener('change', handleChange)
    }
  }, [])

  function closeSidebarOnMobile() {
    if (!isDesktopViewport()) {
      setIsSidebarOpen(false)
    }
  }

  return (
    <div className="min-h-screen bg-background text-foreground md:flex">
      {isSidebarOpen && (
        <button
          type="button"
          aria-label="Cerrar menú lateral"
          onClick={() => setIsSidebarOpen(false)}
          className="fixed inset-0 z-30 bg-foreground/20 md:hidden"
        />
      )}

      <Sidebar isOpen={isSidebarOpen} onNavigate={closeSidebarOnMobile} />

      <div className="flex min-h-screen flex-1 flex-col">
        <header className="flex h-16 shrink-0 items-center justify-between gap-4 border-b border-border px-4 md:px-8">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="text-muted-foreground"
            aria-expanded={isSidebarOpen}
            aria-controls="app-sidebar"
            aria-label={isSidebarOpen ? 'Cerrar menú' : 'Abrir menú'}
            onClick={() => setIsSidebarOpen((open) => !open)}
          >
            {isSidebarOpen ? (
              <ArrowLeftToLine aria-hidden="true" />
            ) : (
              <Menu aria-hidden="true" />
            )}
          </Button>

          <UserMenu />
        </header>

        <main className="flex min-h-0 flex-1 flex-col px-4 py-4 md:px-8 md:py-10">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
