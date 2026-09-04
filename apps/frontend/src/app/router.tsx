import { createBrowserRouter, Navigate } from 'react-router'
import AppLayout from './AppLayout'
import RequireAuth from '../features/auth/components/RequireAuth'
import RequireRole from '../features/auth/components/RequireRole'
import { ROLE } from '../shared/auth/roles'
import LoginPage from '../features/auth/pages/LoginPage'
import LotsRoute from './LotsRoute'
import LoteosRoute from './LoteosRoute'
import LoteoDetailRoute from './LoteoDetailRoute'
import ClientsPage from '../features/clients/pages/ClientsPage'
import ReservationsPage from '../features/reservations/pages/ReservationsPage'
import SalesPage from '../features/sales/pages/SalesPage'
import BillingPage from '../features/billing/pages/BillingPage'
import UsersRoute from './UsersRoute'
import AgenciesRoute from './AgenciesRoute'

export const router = createBrowserRouter([
  {
    path: '/',
    element: (
      <RequireAuth>
        <Navigate to="/lotes" replace />
      </RequireAuth>
    ),
  },
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      {
        path: '/lotes',
        element: <LoteosRoute />,
      },
      {
        path: '/lotes/nuevo',
        element: <LotsRoute />,
      },
      {
        path: '/lotes/:loteoId',
        element: <LoteoDetailRoute />,
      },
      {
        path: '/clientes',
        element: <ClientsPage />,
      },
      {
        path: '/reservas',
        element: <ReservationsPage />,
      },
      {
        path: '/ventas',
        element: <SalesPage />,
      },
      {
        path: '/cobranzas',
        element: <BillingPage />,
      },
      {
        path: '/usuarios',
        element: (
          <RequireRole roles={[ROLE.administrador]}>
            <UsersRoute />
          </RequireRole>
        ),
      },
      {
        path: '/inmobiliarias',
        element: <AgenciesRoute />,
      },
    ],
  },
])
