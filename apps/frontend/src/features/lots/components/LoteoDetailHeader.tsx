import { ArrowLeft, MapPin } from 'lucide-react'
import { Link } from 'react-router'
import { cn } from '../../../shared/lib/utils'
import { formatDate } from '../../../shared/lib/formatDate'
import type { LoteoDetail } from '../types'
import LoteoStatusBadge from './LoteoStatusBadge'

type LoteoDetailHeaderProps = {
  loteo: LoteoDetail
  hasPlan: boolean
}

export default function LoteoDetailHeader({ loteo, hasPlan }: LoteoDetailHeaderProps) {
  const stats = [
    { key: 'Lotes', value: loteo.lotes.length },
    { key: 'Manzanas', value: loteo.manzanas.length },
    { key: 'Calles', value: loteo.calles.length },
  ]
  const createdAt = formatDate(loteo.fechaCreacion)

  return (
    <header className="flex flex-col gap-3">
      <Link
        to="/lotes"
        className="inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft aria-hidden className="size-4" />
        Volver al listado
      </Link>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex flex-col gap-1">
          <h1 className="text-xl font-semibold md:text-2xl">{loteo.nombre}</h1>
          {loteo.ubicacion && (
            <p className="flex items-center gap-1 text-sm text-muted-foreground">
              <MapPin aria-hidden className="size-3.5 shrink-0" />
              <span>{loteo.ubicacion}</span>
            </p>
          )}
        </div>
        <LoteoStatusBadge hasPlan={hasPlan} />
      </div>

      {loteo.descripcion && (
        <p className="text-sm text-muted-foreground">{loteo.descripcion}</p>
      )}

      <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
        {stats.map((stat, index) => (
          <div
            key={stat.key}
            className={cn('flex flex-col', index > 0 && 'border-l border-border pl-5')}
          >
            <span className="text-lg font-semibold leading-none tabular-nums">
              {stat.value}
            </span>
            <span className="mt-1 text-[0.65rem] uppercase tracking-wide text-muted-foreground">
              {stat.key}
            </span>
          </div>
        ))}
        {createdAt && (
          <div className="flex flex-col border-l border-border pl-5">
            <span className="text-lg font-semibold leading-none tabular-nums">
              {createdAt}
            </span>
            <span className="mt-1 text-[0.65rem] uppercase tracking-wide text-muted-foreground">
              Alta
            </span>
          </div>
        )}
      </div>
    </header>
  )
}
