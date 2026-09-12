export default function ReservationCreatePlanSkeleton() {
  return (
    <div
      role="status"
      aria-label="Cargando los datos para crear la reserva…"
      aria-live="polite"
      className="flex h-[28rem] min-w-0 flex-col rounded-xl bg-card p-3 ring-1 ring-foreground/10"
    >
      <span className="sr-only">Cargando los datos para crear la reserva…</span>
      <div
        aria-hidden="true"
        className="flex min-h-0 flex-1 animate-pulse flex-col gap-3 motion-reduce:animate-none"
      >
        <div className="flex items-center justify-between gap-4">
          <div className="h-4 w-16 rounded-full bg-muted" />
          <div className="h-4 w-24 rounded-full bg-muted" />
        </div>
        <div className="flex gap-2">
          <div className="h-7 w-20 rounded-md bg-muted" />
          <div className="h-7 w-24 rounded-md bg-muted" />
        </div>
        <div className="relative min-h-0 flex-1 overflow-hidden rounded-xl border border-border bg-muted/30">
          <div className="absolute inset-x-[10%] inset-y-[12%] rotate-2 rounded-[1.5rem] border-4 border-muted-foreground/10" />
          <div className="absolute left-[17%] top-[22%] h-[25%] w-[28%] rotate-3 rounded-md bg-muted" />
          <div className="absolute right-[18%] top-[20%] h-[22%] w-[29%] -rotate-3 rounded-md bg-muted" />
          <div className="absolute bottom-[18%] left-[26%] h-[27%] w-[24%] -rotate-2 rounded-md bg-muted" />
          <div className="absolute bottom-[20%] right-[18%] h-[25%] w-[25%] rotate-3 rounded-md bg-muted-foreground/15" />
        </div>
      </div>
    </div>
  )
}
