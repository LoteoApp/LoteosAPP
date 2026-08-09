import { useEffect, useRef, useState } from 'react'
import { CircleUser } from 'lucide-react'

const placeholderAccount = {
  name: 'Admin User',
  email: 'admin@loteosapp.com',
  role: 'Administrador',
}

export default function UserMenu() {
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

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={isOpen}
        aria-controls="user-menu"
        aria-label={`Cuenta de ${placeholderAccount.name}`}
        onClick={() => setIsOpen((open) => !open)}
        className="flex items-center gap-3 text-muted-foreground hover:text-foreground"
      >
        <span className="hidden text-right sm:block">
          <span className="block text-sm font-medium text-foreground">
            {placeholderAccount.name}
          </span>
          <span className="block text-xs text-muted-foreground">
            {placeholderAccount.role}
          </span>
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
            onClick={() => setIsOpen(false)}
            className="block w-full px-3 py-2 text-left hover:bg-accent hover:text-accent-foreground"
          >
            Cerrar sesión
          </button>
        </div>
      )}
    </div>
  )
}
