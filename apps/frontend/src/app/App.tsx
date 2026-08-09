import DatabaseStatus from '../features/system-status/components/DatabaseStatus'

// Pre-dates theme tokens; migration tracked separately.
/* eslint-disable local/no-raw-tailwind-colors */
export default function App() {
  return (
    <main className="min-h-screen bg-slate-950 px-6 py-16 text-slate-100">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-10">
        <header>
          <p className="text-sm font-medium uppercase tracking-[0.24em] text-cyan-400">
            LoteosAPP
          </p>
          <h1 className="mt-3 text-4xl font-semibold tracking-tight">
            Diagnóstico del entorno
          </h1>
          <p className="mt-3 max-w-xl text-slate-400">
            Información de conexión entre el frontend, el backend Go y PostgreSQL.
          </p>
        </header>

        <DatabaseStatus />
      </div>
    </main>
  )
}
