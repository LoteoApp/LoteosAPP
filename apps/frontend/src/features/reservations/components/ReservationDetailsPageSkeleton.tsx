import { Card, CardContent, CardHeader } from '../../../shared/ui/card'
import ReservationDetailsPlanSkeleton from './ReservationDetailsPlanSkeleton'

const LOADING_MESSAGE = 'Cargando el detalle de la reserva…'

export default function ReservationDetailsPageSkeleton() {
  return (
    <div role="status" aria-label={LOADING_MESSAGE} aria-live="polite">
      <span className="sr-only">{LOADING_MESSAGE}</span>
      <div aria-hidden="true" className="grid min-w-0 animate-pulse gap-4 motion-reduce:animate-none">
        <ReservationDetailsPlanSkeleton presentational />
        <Card>
          <CardHeader className="grid gap-2">
            <div className="flex flex-wrap items-center gap-2">
              <div className="h-5 w-40 rounded-md bg-muted" />
              <div className="h-5 w-20 rounded-full bg-muted" />
            </div>
            <div className="h-4 w-24 rounded-full bg-muted" />
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2">
            <InfoPlaceholder valueClassName="w-56" />
            <InfoPlaceholder valueClassName="w-40" />
            <InfoPlaceholder valueClassName="w-44" />
            <InfoPlaceholder valueClassName="w-36" />
            <InfoPlaceholder valueClassName="w-48" />
            <InfoPlaceholder valueClassName="w-48" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <div className="h-5 w-28 rounded-md bg-muted" />
          </CardHeader>
          <CardContent>
            <ol className="grid gap-3">
              <HistoryEntryPlaceholder />
              <HistoryEntryPlaceholder />
              <HistoryEntryPlaceholder />
            </ol>
          </CardContent>
        </Card>
        <div className="flex flex-wrap items-center gap-2">
          <div className="h-9 w-full rounded-md bg-muted sm:w-52" />
          <div className="h-9 w-full rounded-md bg-muted sm:w-28" />
        </div>
      </div>
    </div>
  )
}

function InfoPlaceholder({ valueClassName }: { valueClassName: string }) {
  return (
    <div className="grid gap-1.5">
      <div className="h-3 w-20 rounded-full bg-muted" />
      <div className={`h-4 max-w-full rounded-full bg-muted ${valueClassName}`} />
    </div>
  )
}

function HistoryEntryPlaceholder() {
  return (
    <li className="grid gap-2 border-l-2 border-border pl-3">
      <div className="flex flex-wrap items-center gap-2">
        <div className="h-5 w-20 rounded-full bg-muted" />
        <div className="h-3 w-32 rounded-full bg-muted" />
      </div>
      <div className="h-3 w-3/5 rounded-full bg-muted" />
    </li>
  )
}
