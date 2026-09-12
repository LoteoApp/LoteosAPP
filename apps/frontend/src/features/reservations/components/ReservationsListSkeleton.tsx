import { Card, CardContent } from '../../../shared/ui/card'

const LOADING_MESSAGE = 'Cargando reservas…'

export default function ReservationsListSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <div role="status" aria-label={LOADING_MESSAGE} aria-live="polite">
      <span className="sr-only">{LOADING_MESSAGE}</span>
      <ul aria-hidden="true" className="grid animate-pulse gap-3 motion-reduce:animate-none">
        {Array.from({ length: rows }, (_, index) => (
          <li key={index}>
            <Card>
              <CardContent className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                <div className="grid gap-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <div className="h-4 w-40 rounded-md bg-muted" />
                    <div className="h-4 w-20 rounded-full bg-muted" />
                    <div className="h-5 w-20 rounded-full bg-muted" />
                  </div>
                  <div className="h-3 w-3/5 rounded-full bg-muted" />
                  <div className="h-3 w-2/5 rounded-full bg-muted" />
                </div>
                <div className="flex flex-wrap gap-2 sm:justify-end">
                  <div className="h-9 w-full rounded-md bg-muted sm:w-28" />
                  <div className="h-9 w-full rounded-md bg-muted sm:w-24" />
                </div>
              </CardContent>
            </Card>
          </li>
        ))}
      </ul>
    </div>
  )
}
