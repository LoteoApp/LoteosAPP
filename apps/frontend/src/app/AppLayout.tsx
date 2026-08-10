import { useState } from 'react'
import { ArrowLeftToLine, Menu } from 'lucide-react'
import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar'
import UserMenu from './UserMenu'

function isDesktopViewport() {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return false
  }

  return window.matchMedia('(min-width: 768px)').matches
}

export default function AppLayout() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(isDesktopViewport)

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
          <button
            type="button"
            aria-expanded={isSidebarOpen}
            aria-controls="app-sidebar"
            aria-label={isSidebarOpen ? 'Cerrar menú' : 'Abrir menú'}
            onClick={() => setIsSidebarOpen((open) => !open)}
            className="text-muted-foreground hover:text-foreground"
          >
            {isSidebarOpen ? (
              <ArrowLeftToLine aria-hidden="true" />
            ) : (
              <Menu aria-hidden="true" />
            )}
          </button>

          <UserMenu />
        </header>

        <main className="flex-1 px-4 py-6 md:px-8 md:py-10">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
