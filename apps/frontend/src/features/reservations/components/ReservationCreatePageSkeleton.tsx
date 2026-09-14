import { Card, CardContent, CardHeader } from '../../../shared/ui/card'
import ReservationCreatePlanSkeleton from './ReservationCreatePlanSkeleton'

const LOADING_MESSAGE = 'Cargando los datos para crear la reserva…'

export default function ReservationCreatePageSkeleton() {
  return (
    <div role="status" aria-label={LOADING_MESSAGE} aria-live="polite">
      <span className="sr-only">{LOADING_MESSAGE}</span>
      <div
        aria-hidden="true"
        className="grid min-w-0 animate-pulse gap-4 motion-reduce:animate-none lg:grid-cols-12 lg:items-start"
      >
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-5">
          <ReservationCreatePlanSkeleton />
          <Card>
            <CardHeader className="grid gap-2">
              <div className="h-5 w-40 rounded-md bg-muted" />
              <div className="h-4 w-56 max-w-full rounded-full bg-muted" />
            </CardHeader>
            <CardContent className="grid gap-4">
              <div className="grid gap-2">
                <div className="h-3 w-full rounded-full bg-muted" />
                <div className="h-3 w-4/5 rounded-full bg-muted" />
              </div>
              <dl className="grid gap-3 sm:grid-cols-3">
                <DetailPlaceholder />
                <DetailPlaceholder />
                <DetailPlaceholder />
              </dl>
              <DetailPlaceholder valueClassName="w-3/4" />
              <DetailPlaceholder valueClassName="w-2/3" />
            </CardContent>
          </Card>
        </div>
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-7">
          <Card>
            <CardHeader className="grid gap-2">
              <div className="h-5 w-36 rounded-md bg-muted" />
              <div className="h-4 w-64 max-w-full rounded-full bg-muted" />
            </CardHeader>
            <CardContent className="grid gap-4">
              <div className="grid gap-2 rounded-lg border border-border bg-muted/30 p-3">
                <div className="h-4 w-24 rounded-full bg-muted" />
                <div className="h-3 w-3/4 rounded-full bg-muted" />
              </div>
              <FieldPlaceholder withAction />
              <FieldPlaceholder />
              <div className="h-3 w-2/3 rounded-full bg-muted" />
              <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
                <div className="h-9 w-full rounded-md bg-muted sm:w-40" />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}

function DetailPlaceholder({ valueClassName = 'w-24' }: { valueClassName?: string }) {
  return (
    <div className="grid gap-1.5">
      <div className="h-3 w-16 rounded-full bg-muted" />
      <div className={`h-4 max-w-full rounded-full bg-muted ${valueClassName}`} />
    </div>
  )
}

function FieldPlaceholder({ withAction = false }: { withAction?: boolean }) {
  return (
    <div className="grid gap-2">
      <div className="flex items-center justify-between gap-3">
        <div className="h-4 w-20 rounded-full bg-muted" />
        {withAction && <div className="h-8 w-32 rounded-md bg-muted" />}
      </div>
      <div className="h-9 w-full rounded-md bg-muted" />
    </div>
  )
}
