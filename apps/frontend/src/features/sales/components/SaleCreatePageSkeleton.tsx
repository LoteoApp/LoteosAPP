import { Card, CardContent, CardHeader } from '../../../shared/ui/card'

const LOADING_MESSAGE = 'Cargando los datos para registrar la venta…'

export default function SaleCreatePageSkeleton() {
  return (
    <div role="status" aria-label={LOADING_MESSAGE} aria-live="polite">
      <span className="sr-only">{LOADING_MESSAGE}</span>
      <div
        aria-hidden="true"
        className="grid min-w-0 animate-pulse gap-4 motion-reduce:animate-none lg:grid-cols-12 lg:items-start"
      >
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-5">
          <PlanPlaceholder />
          <Card>
            <CardHeader className="grid gap-2">
              <div className="h-5 w-40 rounded-md bg-muted" />
              <div className="h-4 w-56 max-w-full rounded-full bg-muted" />
            </CardHeader>
            <CardContent className="grid gap-4">
              <dl className="grid gap-3 sm:grid-cols-3">
                <DetailPlaceholder />
                <DetailPlaceholder />
                <DetailPlaceholder />
              </dl>
              <DetailPlaceholder valueClassName="w-2/3" />
            </CardContent>
          </Card>
        </div>
        <div className="flex min-w-0 flex-col gap-4 lg:col-span-7">
          <Card>
            <CardHeader className="grid gap-2">
              <div className="h-5 w-36 rounded-md bg-muted" />
            </CardHeader>
            <CardContent className="grid gap-6">
              <FieldPlaceholder withAction />
              <FieldPlaceholder />
              <FieldPlaceholder />
              <FieldPlaceholder />
            </CardContent>
          </Card>
          <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <div className="h-9 w-full rounded-md bg-muted sm:w-40" />
          </div>
        </div>
      </div>
    </div>
  )
}

function PlanPlaceholder() {
  return (
    <div className="flex h-72 min-w-0 flex-col rounded-xl bg-card p-3 ring-1 ring-foreground/10 sm:h-[28rem]">
      <div className="relative min-h-0 flex-1 overflow-hidden rounded-xl border border-border bg-muted/30">
        <div className="absolute inset-x-[10%] inset-y-[12%] rotate-2 rounded-[1.5rem] border-4 border-muted-foreground/10" />
        <div className="absolute left-[17%] top-[22%] h-[25%] w-[28%] rotate-3 rounded-md bg-muted" />
        <div className="absolute right-[18%] top-[20%] h-[22%] w-[29%] -rotate-3 rounded-md bg-muted" />
        <div className="absolute bottom-[18%] left-[26%] h-[27%] w-[24%] -rotate-2 rounded-md bg-muted" />
        <div className="absolute bottom-[20%] right-[18%] h-[25%] w-[25%] rotate-3 rounded-md bg-muted-foreground/15" />
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
