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
  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="border-b border-border">
        <nav
          aria-label="Secciones"
          className="mx-auto flex max-w-6xl items-center gap-8 px-6"
        >
          <span className="py-4 text-sm font-medium">LoteosAPP</span>
          <ul className="flex gap-6">
            {navItems.map((item) => (
              <li key={item.to}>
                <NavLink
                  to={item.to}
                  className={({ isActive }) =>
                    `inline-flex items-center border-b-2 py-4 text-sm transition-colors ${
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
      </header>

      <main className="mx-auto max-w-6xl px-6 py-10">
        <Outlet />
      </main>
    </div>
  )
}
