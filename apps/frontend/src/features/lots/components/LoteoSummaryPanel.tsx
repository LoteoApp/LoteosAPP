import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import { formatDate } from '../../../shared/lib/formatDate'
import { countLotesByState, LOT_STATE_LABELS } from '../lib/lotStateVisuals'
import { LOT_STATES, type LoteoDetail } from '../types'
import LotStateBadge from './LotStateBadge'

type LoteoSummaryPanelProps = {
  loteo: LoteoDetail
}

export default function LoteoSummaryPanel({ loteo }: LoteoSummaryPanelProps) {
  const createdAt = formatDate(loteo.fechaCreacion)
  const counts = countLotesByState(loteo.lotes)

  return (
    <div className="flex flex-col gap-3">
      <Card size="sm">
        <CardHeader>
          <CardTitle>Estado de los lotes</CardTitle>
          <CardDescription>
            {loteo.lotes.length === 0
              ? 'Este loteo todavía no tiene lotes cargados.'
              : `${loteo.lotes.length} lotes en total.`}
          </CardDescription>
        </CardHeader>
        {loteo.lotes.length > 0 && (
          <CardContent>
            <ul aria-label="Lotes por estado" className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {LOT_STATES.map((state) => (
                <li key={state} className="flex flex-col items-start gap-1">
                  <span
                    aria-label={`${counts[state]} ${LOT_STATE_LABELS[state]}`}
                    className="text-2xl font-semibold leading-none tabular-nums"
                  >
                    {counts[state]}
                  </span>
                  <LotStateBadge state={state} />
                </li>
              ))}
            </ul>
          </CardContent>
        )}
      </Card>

      <Card size="sm">
        <CardHeader>
          <CardTitle>Datos del loteo</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-3 sm:grid-cols-2">
            <SummaryRow label="Manzanas" value={String(loteo.manzanas.length)} />
            <SummaryRow label="Calles" value={String(loteo.calles.length)} />
            <SummaryRow label="Alta" value={createdAt} />
          </dl>
        </CardContent>
      </Card>
    </div>
  )
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-border/60 pb-2 last:border-b-0 last:pb-0">
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd className="text-sm">{value || '—'}</dd>
    </div>
  )
}
