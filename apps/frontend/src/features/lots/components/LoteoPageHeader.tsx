import { Button } from '../../../shared/ui/button'
import LoteoStatusBadge from './LoteoStatusBadge'

type LoteoPageHeaderProps = {
  hasPlan: boolean
  onDiscard: () => void
}

export default function LoteoPageHeader({ hasPlan, onDiscard }: LoteoPageHeaderProps) {
  return (
    <header className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
      <div className="flex flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <h1 className="text-xl font-semibold md:text-2xl">Nuevo loteo</h1>
          <LoteoStatusBadge hasPlan={hasPlan} />
        </div>
        <p className="text-sm text-muted-foreground">
          Cargá los datos del loteo; el plano lo podés sumar cuando llegue del agrimensor.
        </p>
      </div>

      <div className="flex flex-col items-start gap-1 md:items-end">
        <div className="flex gap-2">
          <Button
            type="button"
            variant="ghost"
            className="min-h-11 md:h-8 md:min-h-8"
            onClick={onDiscard}
          >
            Descartar
          </Button>
          <Button type="button" disabled className="min-h-11 md:h-8 md:min-h-8">
            Guardar loteo
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">El guardado en la base llega después.</p>
      </div>
    </header>
  )
}
