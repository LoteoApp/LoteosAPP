import { Map } from 'lucide-react'

export default function DxfPlanEmptyState() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-border bg-muted/30 p-6 text-center">
      <Map className="size-6 text-muted-foreground" aria-hidden />
      <p className="text-sm font-medium">Todavía no hay plano</p>
      <p className="max-w-xs text-sm text-muted-foreground">
        El agrimensor puede subir el DXF más adelante. El loteo se guarda igual.
      </p>
    </div>
  )
}
