import { useState } from 'react'
import { Menu, X } from 'lucide-react'
import { NavLink, Outlet } from 'react-router'

const navItems = [
  { to: '/lotes', label: 'Lotes' },
  { to: '/clientes', label: 'Clientes' },
  { to: '/reservas', label: 'Reservas' },
  { to: '/ventas', label: 'Ventas' },
  { to: '/cobranzas', label: 'Cobranzas' },
  { to: '/usuarios', label: 'Usuarios' },
  { to: '/documentacion', label: 'Documentación' },
]

export default function AppLayout() {
  const [isMenuOpen, setIsMenuOpen] = useState(false)

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="border-b border-border">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 px-6 py-4 md:flex-nowrap md:gap-8">
          <span className="text-sm font-medium">LoteosAPP</span>

          <button
            type="button"
            aria-expanded={isMenuOpen}
            aria-controls="secciones-nav"
            aria-label={isMenuOpen ? 'Cerrar menú' : 'Abrir menú'}
            onClick={() => setIsMenuOpen((open) => !open)}
            className="text-muted-foreground hover:text-foreground md:hidden"
          >
            {isMenuOpen ? <X aria-hidden="true" /> : <Menu aria-hidden="true" />}
          </button>

          <nav
            id="secciones-nav"
            aria-label="Secciones"
            className={`w-full md:flex md:w-auto ${isMenuOpen ? 'flex' : 'hidden'}`}
          >
            <ul className="flex flex-col gap-1 py-2 md:flex-row md:gap-6 md:py-0">
              {navItems.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    onClick={() => setIsMenuOpen(false)}
                    className={({ isActive }) =>
                      `block border-b-2 py-2 text-sm transition-colors md:inline-flex md:items-center md:py-4 ${
                        isActive
                          ? 'border-foreground text-foreground'
                          : 'border-transparent text-muted-foreground hover:text-foreground'
                      }`
                    }
                  >
                    {item.label}
                  </NavLink>
                </li>
              ))}
            </ul>
          </nav>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6 py-10">
        <Outlet />
      </main>
    </div>
  )
}
