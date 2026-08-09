import AuthStatus from '../features/auth/components/AuthStatus'
import DatabaseStatus from '../features/system-status/components/DatabaseStatus'

export default function App() {
  return (
    <main className="min-h-screen bg-background px-6 py-16 text-foreground">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-10">
        <header className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-sm font-medium uppercase tracking-[0.24em] text-primary">
              LoteosAPP
            </p>
            <h1 className="mt-3 text-4xl font-semibold tracking-tight">
              Diagnóstico del entorno
            </h1>
            <p className="mt-3 max-w-xl text-muted-foreground">
              Información de conexión entre el frontend, el backend Go y PostgreSQL.
            </p>
          </div>
          <AuthStatus />
        </header>

        <DatabaseStatus />
      </div>
    </main>
  )
}
