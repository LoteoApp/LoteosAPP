import type { ReactNode } from 'react'
import { Button } from '../../../shared/ui/button'
import { Badge } from '../../../shared/ui/badge'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '../../../shared/ui/dialog'
import { cn } from '../../../shared/lib/utils'
import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import {
  PAYMENT_METHOD_LABELS,
  lotOptionLabel,
  sellerAgencyLabel,
  sellerOptionLabel,
  type SaleReceipt,
} from '../types'

type SaleReceiptDialogProps = {
  open: boolean
  receipt: SaleReceipt | null
  onClose: () => void
}

function Field({
  term,
  className,
  children,
}: {
  term: string
  className?: string
  children: ReactNode
}) {
  return (
    <div className={cn('flex flex-col gap-1', className)}>
      <dt className="text-[0.6875rem] font-medium tracking-[0.12em] text-muted-foreground uppercase print:text-black">
        {term}
      </dt>
      <dd className="text-sm font-medium text-foreground print:text-black">{children}</dd>
    </div>
  )
}

function SignatureLine({ label }: { label: string }) {
  return (
    <div className="flex flex-col">
      <div className="h-12 border-b border-border print:border-black" />
      <p className="pt-2 text-xs text-muted-foreground print:text-black">{label}</p>
    </div>
  )
}

export default function SaleReceiptDialog({ open, receipt, onClose }: SaleReceiptDialogProps) {
  if (receipt === null) {
    return null
  }

  const { lot, client, seller, method, amount, currency, issuedAt } = receipt

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose()
        }
      }}
    >
      <DialogContent className="p-4 sm:p-6 print:static print:max-h-none print:w-full print:max-w-none print:translate-none print:overflow-visible print:rounded-none print:border-0 print:bg-white print:p-0 print:shadow-none">
        <DialogTitle className="print:hidden">Venta confirmada</DialogTitle>
        <DialogDescription className="mt-1 print:hidden">
          Revisá los datos y generá el recibo para imprimirlo o guardarlo en PDF.
        </DialogDescription>

        <article
          data-print-area
          className="mt-4 overflow-hidden rounded-xl border border-border border-t-4 border-t-primary bg-card print:mt-0 print:rounded-none print:border-black"
        >
          <header className="flex flex-col gap-3 border-b border-border bg-muted px-4 py-4 sm:flex-row sm:items-start sm:justify-between sm:px-6 print:border-black">
            <div className="flex flex-col gap-1">
              <p className="text-xs font-medium tracking-[0.24em] text-primary uppercase print:text-black">
                LoteosAPP
              </p>
              <h2 className="text-xl font-semibold tracking-tight text-foreground print:text-black">
                Recibo de venta
              </h2>
            </div>
            <div className="flex flex-col gap-1 sm:items-end">
              <p className="text-[0.6875rem] font-medium tracking-[0.12em] text-muted-foreground uppercase print:text-black">
                Emitido el
              </p>
              <p className="text-sm font-medium text-foreground print:text-black">
                {formatDate(issuedAt)}
              </p>
            </div>
          </header>

          <div className="flex flex-col gap-3 border-b border-border px-4 py-5 sm:flex-row sm:items-end sm:justify-between sm:px-6 print:border-black">
            <div className="flex flex-col gap-1">
              <p className="text-[0.6875rem] font-medium tracking-[0.12em] text-muted-foreground uppercase print:text-black">
                Total de la operación
              </p>
              <p className="text-3xl font-semibold tracking-tight text-foreground tabular-nums print:text-black">
                {formatCurrency(amount, currency)}
              </p>
            </div>
            <Badge className="h-6 self-start px-3 sm:self-auto print:border-black print:bg-white print:text-black">
              {PAYMENT_METHOD_LABELS[method]}
            </Badge>
          </div>

          <dl className="grid grid-cols-1 gap-x-6 gap-y-5 px-4 py-5 sm:grid-cols-2 sm:px-6">
            <Field term="Lote" className={lot.area === null ? 'sm:col-span-2' : undefined}>
              {lotOptionLabel(lot)}
            </Field>
            {lot.area !== null && (
              <Field term="Superficie">{formatArea(lot.area)}</Field>
            )}
            <Field term="Comprador">
              {client.apellido}, {client.nombre}
            </Field>
            <Field term="DNI">{client.dni}</Field>
            <Field term="Vendedor">{sellerOptionLabel(seller)}</Field>
            <Field term="Inmobiliaria">{sellerAgencyLabel(seller)}</Field>
          </dl>

          <footer className="flex flex-col gap-5 border-t border-border bg-muted/40 px-4 py-5 sm:px-6 print:border-black">
            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 sm:gap-10">
              <SignatureLine label="Firma del comprador" />
              <SignatureLine label="Firma del vendedor" />
            </div>
            <p className="text-xs text-muted-foreground print:text-black">
              Documento generado automáticamente por LoteosAPP.
            </p>
          </footer>
        </article>

        <div className="mt-6 flex flex-col gap-2 sm:flex-row sm:justify-end print:hidden">
          <DialogClose
            render={
              <Button type="button" variant="outline" className="min-h-11 sm:min-h-9">
                Cerrar
              </Button>
            }
          />
          <Button type="button" className="min-h-11 sm:min-h-9" onClick={() => window.print()}>
            Imprimir recibo
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
