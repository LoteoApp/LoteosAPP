import { Link } from 'react-router'
import { Button } from '../../../shared/ui/button'
import LoteoStatusBadge from './LoteoStatusBadge'

type LoteoPageHeaderProps = {
  hasPlan: boolean
  canSave: boolean
  isSaving: boolean
  onSave: () => void
  onDiscard: () => void
}

export default function LoteoPageHeader({
  hasPlan,
  canSave,
  isSaving,
  onSave,
  onDiscard,
}: LoteoPageHeaderProps) {
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

      <div className="flex gap-2 md:items-end">
        <Button
          variant="ghost"
          className="min-h-11 md:h-8 md:min-h-8"
          render={<Link to="/lotes" />}
        >
          Volver al listado
        </Button>
        <Button
          type="button"
          variant="ghost"
          className="min-h-11 md:h-8 md:min-h-8"
          onClick={onDiscard}
          disabled={isSaving}
        >
          Descartar
        </Button>
        <Button
          type="button"
          className="min-h-11 md:h-8 md:min-h-8"
          onClick={onSave}
          disabled={!canSave || isSaving}
        >
          {isSaving ? 'Guardando…' : 'Guardar loteo'}
        </Button>
      </div>
    </header>
  )
}
