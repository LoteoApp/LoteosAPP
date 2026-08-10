import { createBrowserRouter } from 'react-router-dom'
import App from './App'
import AppLayout from './AppLayout'
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
    element: <AppLayout />,
    children: [
      { path: '/lotes', element: <LotsPage /> },
      { path: '/clientes', element: <ClientsPage /> },
      { path: '/reservas', element: <ReservationsPage /> },
      { path: '/ventas', element: <SalesPage /> },
      { path: '/cobranzas', element: <BillingPage /> },
      { path: '/usuarios', element: <UsersPage /> },
      { path: '/documentacion', element: <LegalPage /> },
    ],
  },
])
