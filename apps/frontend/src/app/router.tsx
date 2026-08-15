import { createBrowserRouter, Navigate } from 'react-router'
import AppLayout from './AppLayout'
import MonitorPage from '../features/system-status/pages/MonitorPage'
import RequireAuth from '../features/auth/components/RequireAuth'
import LoginPage from '../features/auth/pages/LoginPage'
import LotsPage from '../features/lots/pages/LotsPage'
import ClientsPage from '../features/clients/pages/ClientsPage'
import ReservationsPage from '../features/reservations/pages/ReservationsPage'
import SalesPage from '../features/sales/pages/SalesPage'
import BillingPage from '../features/billing/pages/BillingPage'
import UsersPage from '../features/users/pages/UsersPage'
import AgenciesPage from '../features/agencies/pages/AgenciesPage'

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
    path: '/monitor',
    element: <MonitorPage />,
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
        element: <LotsPage />,
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
        element: <UsersPage />,
      },
      {
        path: '/inmobiliaria',
        element: <AgenciesPage />,
      },
    ],
  },
])
