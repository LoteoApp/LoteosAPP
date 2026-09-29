import type { ReactNode } from 'react'
import { Button } from '../../../shared/ui/button'
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from '../../../shared/ui/dialog'
import { cn } from '../../../shared/lib/utils'
import { formatDate } from '../../../shared/lib/formatDate'

type PrintDialogProps = {
  open: boolean
  title: string
  description: string
  documentTitle: string
  issuedAt: string
  printLabel: string
  onClose: () => void
  children: ReactNode
}

// A dialog whose body is what gets printed: everything else on the page is
// hidden by the [data-print-area] rule in index.css.
export default function PrintDialog({
  open,
  title,
  description,
  documentTitle,
  issuedAt,
  printLabel,
  onClose,
  children,
}: PrintDialogProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={() => onClose()}
    >
      <DialogContent className="max-w-3xl p-4 sm:p-6 print:static print:max-h-none print:w-full print:max-w-none print:translate-none print:overflow-visible print:rounded-none print:border-0 print:bg-white print:p-0 print:shadow-none">
        <DialogTitle className="print:hidden">{title}</DialogTitle>
        <DialogDescription className="mt-1 print:hidden">{description}</DialogDescription>
        <article
          data-print-area
          className="mt-4 overflow-hidden rounded-xl border border-border border-t-4 border-t-primary bg-card print:mt-0 print:rounded-none print:border-black"
        >
          <header className="flex flex-col gap-3 border-b border-border bg-muted px-4 py-4 sm:flex-row sm:items-start sm:justify-between sm:px-6 print:border-black">
            <div className="flex flex-col gap-1">
              <p className="text-xs font-medium tracking-[0.24em] text-primary uppercase print:text-black">LoteosAPP</p>
              <h2 className="text-xl font-semibold tracking-tight text-foreground print:text-black">{documentTitle}</h2>
            </div>
            <div className="flex flex-col gap-1 sm:items-end">
              <p className="text-[0.6875rem] font-medium tracking-[0.12em] text-muted-foreground uppercase print:text-black">
                Emitido el
              </p>
              <p className="text-sm font-medium text-foreground print:text-black">{formatDate(issuedAt)}</p>
            </div>
          </header>
          {children}
          <footer className="border-t border-border bg-muted/40 px-4 py-4 sm:px-6 print:border-black">
            <p className="text-xs text-muted-foreground print:text-black">Documento generado automáticamente por LoteosAPP.</p>
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
            {printLabel}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

export function PrintField({ term, className, children }: { term: string; className?: string; children: ReactNode }) {
  return (
    <div className={cn('flex flex-col gap-1', className)}>
      <dt className="text-[0.6875rem] font-medium tracking-[0.12em] text-muted-foreground uppercase print:text-black">
        {term}
      </dt>
      <dd className="text-sm font-medium text-foreground print:text-black">{children}</dd>
    </div>
  )
}
