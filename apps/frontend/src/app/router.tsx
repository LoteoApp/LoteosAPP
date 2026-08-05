import { createBrowserRouter } from 'react-router'
import App from './App'
import RequireAuth from '../features/auth/components/RequireAuth'
import LotsPage from '../features/lots/pages/LotsPage'
import ClientsPage from '../features/clients/pages/ClientsPage'
import ReservationsPage from '../features/reservations/pages/ReservationsPage'
import SalesPage from '../features/sales/pages/SalesPage'
import BillingPage from '../features/billing/pages/BillingPage'
import UsersPage from '../features/users/pages/UsersPage'
import LegalPage from '../features/legal/pages/LegalPage'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
  },
  {
    path: '/lotes',
    element: (
      <RequireAuth>
        <LotsPage />
      </RequireAuth>
    ),
  },
  {
    path: '/clientes',
    element: (
      <RequireAuth>
        <ClientsPage />
      </RequireAuth>
    ),
  },
  {
    path: '/reservas',
    element: (
      <RequireAuth>
        <ReservationsPage />
      </RequireAuth>
    ),
  },
  {
    path: '/ventas',
    element: (
      <RequireAuth>
        <SalesPage />
      </RequireAuth>
    ),
  },
  {
    path: '/cobranzas',
    element: (
      <RequireAuth>
        <BillingPage />
      </RequireAuth>
    ),
  },
  {
    path: '/usuarios',
    element: (
      <RequireAuth>
        <UsersPage />
      </RequireAuth>
    ),
  },
  {
    path: '/documentacion',
    element: (
      <RequireAuth>
        <LegalPage />
      </RequireAuth>
    ),
  },
])
