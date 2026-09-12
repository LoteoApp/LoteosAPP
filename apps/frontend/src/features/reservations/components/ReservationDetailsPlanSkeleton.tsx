type ReservationDetailsPlanSkeletonProps = {
  message?: string
}

export default function ReservationDetailsPlanSkeleton({
  message = 'Cargando el detalle de la reserva…',
}: ReservationDetailsPlanSkeletonProps) {
  return (
    <div
      role="status"
      aria-label={message}
      aria-live="polite"
      className="flex h-[28rem] min-w-0 flex-col rounded-xl bg-card p-3 ring-1 ring-foreground/10"
    >
      <span className="sr-only">{message}</span>
      <div
        aria-hidden="true"
        className="flex min-h-0 flex-1 animate-pulse flex-col gap-3 motion-reduce:animate-none"
      >
        <div className="h-4 w-16 rounded-full bg-muted" />
        <div className="relative min-h-0 flex-1 overflow-hidden rounded-xl border border-border bg-muted/30">
          <div className="absolute inset-x-[12%] inset-y-[14%] rotate-[-3deg] rounded-[1.5rem] border-4 border-muted-foreground/10" />
          <div className="absolute left-[22%] top-[25%] h-[22%] w-[24%] -rotate-6 rounded-md bg-muted" />
          <div className="absolute right-[20%] top-[22%] h-[28%] w-[25%] rotate-3 rounded-md bg-muted" />
          <div className="absolute bottom-[20%] left-[30%] h-[24%] w-[29%] rotate-2 rounded-md bg-muted-foreground/15" />
        </div>
      </div>
    </div>
  )
}
