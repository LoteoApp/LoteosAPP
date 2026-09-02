import type { ReactNode } from 'react'
import { Navigate } from 'react-router'
import { useAuth } from '../hooks/use-auth'
import { getUserRole } from '../lib/getUserRole'

type RequireRoleProps = {
  roles: readonly string[]
  children: ReactNode
}

// Redirects a caller whose role isn't in `roles` away from the route, so it
// never mounts and never fires a request the backend would reject anyway.
// UX only: the backend enforces its own role checks independently on every
// request. Nests inside RequireAuth, which already resolves isLoading and
// guarantees a session before this renders.
//
// This is a coarse, all-or-nothing gate for a role that either has a
// section or doesn't (e.g. Usuarios: administrador only). It can't express
// per-record access — a role that only sees the subset of loteos assigned
// to it — that's a different mechanism (see "Gestión de roles y permisos"
// in docs/domain.md, still undesigned).
export default function RequireRole({ roles, children }: RequireRoleProps) {
  const { user } = useAuth()
  const role = getUserRole(user)

  if (!role || !roles.includes(role)) {
    return <Navigate to="/lotes" replace />
  }

  return children
}
