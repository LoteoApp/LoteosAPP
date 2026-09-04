import type { ComponentType } from 'react'
import {
  Building2,
  CalendarCheck,
  Handshake,
  LandPlot,
  UserCog,
  Users,
  Wallet,
} from 'lucide-react'
import { Link, NavLink } from 'react-router'
import { useAuth } from '../features/auth/hooks/use-auth'
import { getUserRole, ROLE } from '../shared/auth/roles'

type NavItem = {
  to: string
  label: string
  icon: ComponentType<{ className?: string; 'aria-hidden'?: boolean }>
  // Omitted means every authenticated role sees the section. When present,
  // keep this in sync with the roles the matching route in router.tsx
  // allows through RequireRole — the route is the actual gate, this only
  // controls whether the link is worth showing.
  roles?: readonly string[]
}

const navItems: NavItem[] = [
  { to: '/lotes', label: 'Lotes', icon: LandPlot },
  { to: '/clientes', label: 'Clientes', icon: Users },
  { to: '/reservas', label: 'Reservas', icon: CalendarCheck },
  { to: '/ventas', label: 'Ventas', icon: Handshake },
  { to: '/cobranzas', label: 'Cobranzas', icon: Wallet },
  { to: '/usuarios', label: 'Usuarios', icon: UserCog, roles: [ROLE.administrador] },
  { to: '/inmobiliarias', label: 'Inmobiliarias', icon: Building2 },
]

type SidebarProps = {
  isOpen: boolean
  onNavigate: () => void
}

export default function Sidebar({ isOpen, onNavigate }: SidebarProps) {
  const { user } = useAuth()
  const role = getUserRole(user)
  const visibleNavItems = navItems.filter((item) => !item.roles || (role !== null && item.roles.includes(role)))

  return (
    <aside
      id="app-sidebar"
      aria-label="Secciones"
      inert={!isOpen}
      className={`fixed inset-y-0 left-0 z-40 flex w-64 shrink-0 flex-col overflow-hidden border-r border-sidebar-border bg-sidebar transition-all duration-200 md:static md:translate-x-0 ${
        isOpen ? 'translate-x-0 md:w-64' : '-translate-x-full md:w-0 md:border-r-0'
      }`}
    >
      <Link
        to="/clientes"
        onClick={onNavigate}
        className="flex h-16 w-64 shrink-0 items-center border-b border-sidebar-border px-6 text-sm font-medium text-sidebar-foreground"
      >
        LoteosAPP
      </Link>

      <nav className="w-64 flex-1 overflow-y-auto px-3 py-4">
        <ul className="flex flex-col gap-1">
          {visibleNavItems.map(({ to, label, icon: Icon }) => (
            <li key={to}>
              <NavLink
                to={to}
                onClick={onNavigate}
                className={({ isActive }) =>
                  `flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors ${
                    isActive
                      ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                      : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
                  }`
                }
              >
                <Icon className="size-4 shrink-0" aria-hidden />
                {label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  )
}
