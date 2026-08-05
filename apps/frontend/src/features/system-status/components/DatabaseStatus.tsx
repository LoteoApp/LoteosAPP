import {
  useSystemInfo,
  type SystemInfoState,
} from '../hooks/use-system-info'
import type { SystemInfo } from '../types'
import { Badge } from '../../../shared/ui/badge'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../../../shared/ui/card'

export default function DatabaseStatus() {
  const state = useSystemInfo()

  return (
    <Card>
      <CardHeader>
        <CardDescription>Estado de PostgreSQL</CardDescription>
        <CardTitle className="text-2xl">Conexión de base de datos</CardTitle>
        <CardAction>
          <StatusBadge status={state.status} />
        </CardAction>
      </CardHeader>

      <CardContent>
        {state.status === 'error' ? (
          <ErrorMessage message={state.message} />
        ) : state.status === 'success' ? (
          <SystemDetails systemInfo={state.data} />
        ) : (
          <p className="text-sm text-muted-foreground" aria-live="polite">
            Consultando el backend y PostgreSQL...
          </p>
        )}
      </CardContent>
    </Card>
  )
}

function ErrorMessage({ message }: { message: string }) {
  return (
    <div
      className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"
      role="alert"
    >
      <p className="font-semibold">No se pudo consultar PostgreSQL</p>
      <p className="mt-1">{message}</p>
    </div>
  )
}

function SystemDetails({ systemInfo }: { systemInfo: SystemInfo }) {
  return (
    <div className="space-y-6">
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
        <p className="text-sm font-medium text-muted-foreground">Versión</p>
        <p className="mt-2 break-words rounded-lg bg-muted p-3 font-mono text-xs leading-5 text-foreground">
          {systemInfo.database.version}
        </p>
      </div>

      <div>
        <p className="text-sm font-medium text-muted-foreground">Pool de conexiones</p>
        <div className="mt-2 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <Detail label="Máximo" value={systemInfo.pool.max_connections} />
          <Detail label="Totales" value={systemInfo.pool.total_connections} />
          <Detail label="Adquiridas" value={systemInfo.pool.acquired_connections} />
          <Detail label="Libres" value={systemInfo.pool.idle_connections} />
          <Detail label="Creadas" value={systemInfo.pool.new_connections} />
          <Detail label="Cerradas" value={systemInfo.pool.closed_connections} />
        </div>
      </div>

      <p className="text-xs text-muted-foreground">
        Hora de PostgreSQL: {formatDate(systemInfo.database.database_time)} · Verificado:{' '}
        {formatDate(systemInfo.checked_at)}
      </p>
    </div>
  )
}

function Detail({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-xl border border-border bg-muted/40 p-4">
      <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="mt-1 break-words font-medium text-foreground">{value}</p>
    </div>
  )
}

function StatusBadge({ status }: { status: SystemInfoState['status'] }) {
  const label = status === 'loading' ? 'Consultando' : status === 'success' ? 'Conectado' : 'Desconectado'

  if (status === 'success') {
    return <Badge className="border-emerald-200 bg-emerald-50 text-emerald-700">{label}</Badge>
  }

  if (status === 'loading') {
    return <Badge className="border-amber-200 bg-amber-50 text-amber-700">{label}</Badge>
  }

  return <Badge variant="destructive">{label}</Badge>
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat('es-AR', {
    dateStyle: 'short',
    timeStyle: 'medium',
  }).format(new Date(value))
}
