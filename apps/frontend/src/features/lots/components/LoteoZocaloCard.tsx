import { ChevronRight, MapPin } from 'lucide-react'
import { Link } from 'react-router'
import { Card } from '../../../shared/ui/card'
import { cn } from '../../../shared/lib/utils'
import type { LoteoSummary } from '../types'
import LoteoPlanThumbnail from './LoteoPlanThumbnail'
import LoteoStatusBadge from './LoteoStatusBadge'

type LoteoZocaloCardProps = {
  loteo: LoteoSummary
}

export default function LoteoZocaloCard({ loteo }: LoteoZocaloCardProps) {
  const stats = [
    { key: 'Lotes', value: loteo.cantidadLotes },
    { key: 'Manzanas', value: loteo.cantidadManzanas },
    { key: 'Calles', value: loteo.cantidadCalles },
  ]

  return (
    <Link
      to={`/lotes/${loteo.id}`}
      aria-label={`Ver detalle de ${loteo.nombre}`}
      className="block rounded-xl transition-shadow hover:shadow-md focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
    >
      <Card className="grid grid-cols-1 gap-0 p-0 sm:grid-cols-[13rem_minmax(0,1fr)]">
        <div className="h-32 border-b border-border sm:h-full sm:border-r sm:border-b-0">
          <LoteoPlanThumbnail hasPlan={loteo.tienePlano} />
        </div>

        <div className="flex flex-col gap-2.5 p-4">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <p className="font-medium leading-tight">{loteo.nombre}</p>
              {loteo.ubicacion && (
                <p className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                  <MapPin aria-hidden className="size-3.5 shrink-0" />
                  <span className="truncate">{loteo.ubicacion}</span>
                </p>
              )}
            </div>
            <LoteoStatusBadge hasPlan={loteo.tienePlano} />
          </div>

          {loteo.descripcion && (
            <p className="line-clamp-1 text-xs text-muted-foreground">{loteo.descripcion}</p>
          )}

          <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
            {stats.map((stat, index) => (
              <div
                key={stat.key}
                className={cn(
                  'flex flex-col',
                  index > 0 && 'border-l border-border pl-5',
                )}
              >
                <span className="text-lg font-semibold leading-none tabular-nums">
                  {stat.value}
                </span>
                <span className="mt-1 text-[0.65rem] uppercase tracking-wide text-muted-foreground">
                  {stat.key}
                </span>
              </div>
            ))}

            <span className="grow" />

            <span className="inline-flex h-7 w-full items-center justify-center gap-1 rounded-md border border-border px-2.5 text-xs font-medium sm:w-auto">
              Ver detalle
              <ChevronRight aria-hidden className="size-3.5" />
            </span>
          </div>
        </div>
      </Card>
    </Link>
  )
}
