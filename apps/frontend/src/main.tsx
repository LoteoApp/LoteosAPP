import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router'
import { router } from './app/router'
import AppAuthProvider from './features/auth/components/AppAuthProvider'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppAuthProvider>
      <RouterProvider router={router} />
    </AppAuthProvider>
  </StrictMode>,
)
