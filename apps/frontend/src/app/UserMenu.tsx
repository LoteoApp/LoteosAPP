import { useEffect, useRef, useState } from 'react'
import { CircleUser } from 'lucide-react'
import { useAuth } from 'react-oidc-context'

export default function UserMenu() {
  const auth = useAuth()
  const [isOpen, setIsOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!isOpen) return

    function closeOnOutsideClick(event: MouseEvent) {
      if (!containerRef.current?.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', closeOnOutsideClick)
    document.addEventListener('keydown', closeOnEscape)

    return () => {
      document.removeEventListener('mousedown', closeOnOutsideClick)
      document.removeEventListener('keydown', closeOnEscape)
    }
  }, [isOpen])

  const displayName =
    auth.user?.profile.preferred_username ??
    auth.user?.profile.name ??
    auth.user?.profile.email ??
    'Usuario'
  const email = auth.user?.profile.email

  function handleSignOut() {
    setIsOpen(false)
    void auth.signoutRedirect()
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={isOpen}
        aria-controls="user-menu"
        aria-label={`Cuenta de ${displayName}`}
        onClick={() => setIsOpen((open) => !open)}
        className="flex items-center gap-3 text-muted-foreground hover:text-foreground"
      >
        <span className="hidden text-right sm:block">
          <span className="block text-sm font-medium text-foreground">
            {displayName}
          </span>
          {email && (
            <span className="block text-xs text-muted-foreground">{email}</span>
          )}
        </span>
        <CircleUser className="size-8 shrink-0" aria-hidden />
      </button>

      {isOpen && (
        <div
          id="user-menu"
          role="menu"
          aria-label="Cuenta"
          className="absolute right-0 z-50 mt-2 w-48 rounded-md border border-border bg-popover py-1 text-sm text-popover-foreground shadow-md"
        >
          <button
            type="button"
            role="menuitem"
            onClick={() => setIsOpen(false)}
            className="block w-full px-3 py-2 text-left hover:bg-accent hover:text-accent-foreground"
          >
            Mi perfil
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={handleSignOut}
            className="block w-full px-3 py-2 text-left hover:bg-accent hover:text-accent-foreground"
          >
            Cerrar sesión
          </button>
        </div>
      )}
    </div>
  )
}
