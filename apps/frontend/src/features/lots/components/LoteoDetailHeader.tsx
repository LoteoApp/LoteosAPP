import { ArrowLeft, MapPin, Pencil } from 'lucide-react'
import { Link } from 'react-router'
import { Button } from '../../../shared/ui/button'
import type { LoteoDetail } from '../types'
import LoteoStatusBadge from './LoteoStatusBadge'

type LoteoDetailHeaderProps = {
  loteo: LoteoDetail
  hasPlan: boolean
}

function plural(count: number, singular: string, plural: string): string {
  return `${count} ${count === 1 ? singular : plural}`
}

export default function LoteoDetailHeader({ loteo, hasPlan }: LoteoDetailHeaderProps) {
  const stats = [
    plural(loteo.lotes.length, 'lote', 'lotes'),
    plural(loteo.manzanas.length, 'manzana', 'manzanas'),
    plural(loteo.calles.length, 'calle', 'calles'),
  ]

  return (
    <header className="flex flex-col gap-2">
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
          <p className="flex flex-wrap items-center gap-x-1 text-sm text-muted-foreground">
            {loteo.ubicacion && (
              <>
                <MapPin aria-hidden className="size-3.5 shrink-0" />
                <span>{loteo.ubicacion}</span>
                <span aria-hidden>·</span>
              </>
            )}
            <span>{stats.join(' · ')}</span>
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <LoteoStatusBadge hasPlan={hasPlan} />
          <Button type="button" variant="outline" size="sm" disabled title="Próximamente">
            <Pencil aria-hidden />
            Editar loteo
          </Button>
        </div>
      </div>

      {loteo.descripcion && (
        <p className="text-sm text-muted-foreground">{loteo.descripcion}</p>
      )}
    </header>
  )
}
