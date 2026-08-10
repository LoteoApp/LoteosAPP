import type { ReactNode } from 'react'
import { AuthProvider } from 'react-oidc-context'
import { oidcConfig } from '../config/oidc-config'

export default function AppAuthProvider({ children }: { children: ReactNode }) {
  return <AuthProvider {...oidcConfig}>{children}</AuthProvider>
}
