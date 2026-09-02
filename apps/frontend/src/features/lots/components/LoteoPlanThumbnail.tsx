import { Map } from 'lucide-react'

type LoteoPlanThumbnailProps = {
  hasPlan: boolean
}

// The list endpoint carries no geometry, so this is a schematic stand-in for
// the plan, not the real drawing. The detail screen renders the actual DXF.
export default function LoteoPlanThumbnail({ hasPlan }: LoteoPlanThumbnailProps) {
  if (!hasPlan) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-1.5 bg-muted/40 p-4 text-muted-foreground">
        <div className="flex h-14 w-20 items-center justify-center rounded-md border border-dashed border-muted-foreground/40">
          <Map aria-hidden className="size-5" />
        </div>
        <span className="text-xs">Sin plano</span>
      </div>
    )
  }

  return (
    <div className="flex h-full w-full items-center justify-center bg-muted/40 p-4">
      <svg
        aria-hidden
        viewBox="0 0 176 112"
        className="h-full max-h-28 w-full"
        fill="none"
      >
        <rect
          x="8"
          y="8"
          width="160"
          height="96"
          rx="4"
          className="stroke-muted-foreground/50"
          strokeWidth="1.4"
        />
        <path
          d="M8 60h160M64 8v96M120 8v96"
          className="stroke-muted-foreground/40"
          strokeWidth="1"
        />
        <g className="stroke-muted-foreground/25" strokeWidth="0.8">
          <path d="M22 8v52M43 8v52M85 8v52M103 8v52M139 8v52M156 8v52" />
          <path d="M22 60v44M43 60v44M85 60v44M103 60v44M139 60v44M156 60v44" />
        </g>
      </svg>
    </div>
  )
}
