import {
  useSystemInfo,
  type SystemInfoState,
} from '../hooks/use-system-info'
import type { SystemInfo } from '../types'

export default function DatabaseStatus() {
  const state = useSystemInfo()

  return (
    <section className="rounded-2xl border border-slate-800 bg-slate-900/70 p-6 shadow-2xl shadow-cyan-950/20">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-sm font-medium text-slate-400">Estado de PostgreSQL</p>
          <h2 className="mt-1 text-2xl font-semibold">Conexión de base de datos</h2>
        </div>
        <StatusBadge status={state.status} />
      </div>

      {state.status === 'error' ? (
        <ErrorMessage message={state.message} />
      ) : state.status === 'success' ? (
        <SystemDetails systemInfo={state.data} />
      ) : (
        <p className="mt-6 text-sm text-slate-400" aria-live="polite">
          Consultando el backend y PostgreSQL...
        </p>
      )}
    </section>
  )
}

function ErrorMessage({ message }: { message: string }) {
  return (
    <div
      className="mt-6 rounded-xl border border-rose-900/70 bg-rose-950/40 p-4 text-sm text-rose-200"
      role="alert"
    >
      <p className="font-semibold">No se pudo consultar PostgreSQL</p>
      <p className="mt-1 text-rose-300">{message}</p>
    </div>
  )
}

function SystemDetails({ systemInfo }: { systemInfo: SystemInfo }) {
  return (
    <div className="mt-6 space-y-6">
      <div className="grid gap-3 sm:grid-cols-2">
        <Detail label="Base de datos" value={systemInfo.database.database_name} />
        <Detail label="Usuario" value={systemInfo.database.user} />
        <Detail
          label="Servidor"
          value={`${systemInfo.database.server_address}:${systemInfo.database.server_port}`}
        />
        <Detail label="Estado del backend" value={systemInfo.status} />
      </div>

      <div>
        <p className="text-sm font-medium text-slate-400">Versión</p>
        <p className="mt-2 break-words rounded-lg bg-slate-950 p-3 font-mono text-xs leading-5 text-cyan-200">
          {systemInfo.database.version}
        </p>
      </div>

      <div>
        <p className="text-sm font-medium text-slate-400">Pool de conexiones</p>
        <div className="mt-2 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <Detail label="Máximo" value={systemInfo.pool.max_connections} />
          <Detail label="Totales" value={systemInfo.pool.total_connections} />
          <Detail label="Adquiridas" value={systemInfo.pool.acquired_connections} />
          <Detail label="Libres" value={systemInfo.pool.idle_connections} />
          <Detail label="Creadas" value={systemInfo.pool.new_connections} />
          <Detail label="Cerradas" value={systemInfo.pool.closed_connections} />
        </div>
      </div>

      <p className="text-xs text-slate-500">
        Hora de PostgreSQL: {formatDate(systemInfo.database.database_time)} · Verificado:{' '}
        {formatDate(systemInfo.checked_at)}
      </p>
    </div>
  )
}

function Detail({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
      <p className="text-xs uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 break-words font-medium text-slate-200">{value}</p>
    </div>
  )
}

function StatusBadge({ status }: { status: SystemInfoState['status'] }) {
  const label = status === 'loading' ? 'Consultando' : status === 'success' ? 'Conectado' : 'Desconectado'
  const color =
    status === 'loading'
      ? 'border-amber-800 bg-amber-950/50 text-amber-300'
      : status === 'success'
        ? 'border-emerald-800 bg-emerald-950/50 text-emerald-300'
        : 'border-rose-800 bg-rose-950/50 text-rose-300'

  return <span className={`rounded-full border px-3 py-1 text-xs font-medium ${color}`}>{label}</span>
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat('es-AR', {
    dateStyle: 'short',
    timeStyle: 'medium',
  }).format(new Date(value))
}
