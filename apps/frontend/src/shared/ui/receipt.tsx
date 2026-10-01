import type { ReactNode } from 'react'
import { Button } from './button'
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from './dialog'
import { cn } from '../lib/utils'
import { formatDate } from '../lib/formatDate'

const labelClassName = 'text-[0.6875rem] font-semibold tracking-[0.12em] uppercase'

type ReceiptDialogProps = {
  open: boolean
  title: string
  description: string
  documentTitle: string
  issuedAt: string
  printLabel: string
  onClose: () => void
  children: ReactNode
}

// Only the [data-print-area] article gets printed: index.css hides the rest of the page.
export function ReceiptDialog({
  open,
  title,
  description,
  documentTitle,
  issuedAt,
  printLabel,
  onClose,
  children,
}: ReceiptDialogProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onClose()
        }
      }}
    >
      <DialogContent className="max-w-3xl p-4 sm:p-6 print:static print:max-h-none print:w-full print:max-w-none print:translate-none print:overflow-visible print:rounded-none print:border-0 print:bg-white print:p-0 print:shadow-none">
        <DialogTitle className="print:hidden">{title}</DialogTitle>
        <DialogDescription className="mt-1 print:hidden">{description}</DialogDescription>
        <article
          data-print-area
          className="mt-4 overflow-hidden rounded-xl border border-receipt-field-line bg-white text-receipt-text print:mt-0 print:rounded-none print:border-0"
        >
          <header className="flex flex-col gap-3 bg-receipt-brand px-4 py-5 sm:flex-row sm:items-start sm:justify-between sm:px-6">
            <div className="flex flex-col gap-1.5">
              <p className="text-[0.6875rem] font-semibold tracking-[0.24em] text-receipt-brand-soft uppercase">LoteosAPP</p>
              <h2 className="text-lg font-bold tracking-wide text-white uppercase sm:text-xl">{documentTitle}</h2>
            </div>
            <div className="flex flex-col gap-1 sm:items-end">
              <p className={cn(labelClassName, 'text-receipt-brand-soft')}>Fecha de emisión</p>
              <p className="text-sm font-medium text-white">{formatDate(issuedAt)}</p>
            </div>
          </header>
          <div className="flex flex-col gap-5 px-4 py-5 sm:px-6">{children}</div>
          <footer className="mx-4 border-t border-receipt-rule py-3 sm:mx-6">
            <p className="text-xs text-receipt-label sm:text-right">Documento generado automáticamente por LoteosAPP.</p>
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

type ReceiptSummaryProps = {
  label: string
  asideLabel: string
  aside: ReactNode
  children: ReactNode
}

export function ReceiptSummary({ label, asideLabel, aside, children }: ReceiptSummaryProps) {
  return (
    <div className="flex flex-col gap-3 rounded-lg border border-receipt-summary-line bg-receipt-summary px-4 py-4 sm:flex-row sm:items-end sm:justify-between sm:px-5">
      <div className="flex flex-col gap-1">
        <p className={cn(labelClassName, 'text-receipt-label')}>{label}</p>
        <div className="text-2xl font-bold tracking-tight text-receipt-brand tabular-nums sm:text-3xl">{children}</div>
      </div>
      <div className="flex flex-col gap-1 sm:items-end">
        <p className={cn(labelClassName, 'text-receipt-label')}>{asideLabel}</p>
        <div className="text-base font-bold text-receipt-brand tabular-nums">{aside}</div>
      </div>
    </div>
  )
}

function ReceiptSectionTitle({ children }: { children: ReactNode }) {
  return (
    <span className="block border-b border-receipt-rule pb-1.5 text-left text-xs font-bold tracking-[0.08em] text-receipt-brand uppercase">
      {children}
    </span>
  )
}

export function ReceiptSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      <h3>
        <ReceiptSectionTitle>{title}</ReceiptSectionTitle>
      </h3>
      {children}
    </section>
  )
}

export function ReceiptFields({ label, children }: { label?: string; children: ReactNode }) {
  return (
    <dl aria-label={label} className="grid grid-cols-1 gap-2 sm:grid-cols-2">
      {children}
    </dl>
  )
}

export function ReceiptField({ term, wide = false, children }: { term: string; wide?: boolean; children: ReactNode }) {
  return (
    <div
      className={cn(
        'flex flex-col gap-1 border border-receipt-field-line bg-receipt-field px-4 py-2.5',
        wide && 'sm:col-span-2',
      )}
    >
      <dt className={cn(labelClassName, 'text-receipt-label')}>{term}</dt>
      <dd className="text-sm font-medium break-words text-receipt-text">{children}</dd>
    </div>
  )
}

type ReceiptTableColumn = {
  label: string
  numeric?: boolean
}

type ReceiptTableProps = {
  caption: string
  columns: ReceiptTableColumn[]
  children: ReactNode
}

export function ReceiptTable({ caption, columns, children }: ReceiptTableProps) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <caption className="pb-3">
          <ReceiptSectionTitle>{caption}</ReceiptSectionTitle>
        </caption>
        <thead>
          <tr className="bg-receipt-summary">
            {columns.map((column) => (
              <th
                key={column.label}
                scope="col"
                className={cn(
                  labelClassName,
                  'px-3 py-2 whitespace-nowrap text-receipt-label',
                  column.numeric === true ? 'text-right' : 'text-left',
                )}
              >
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  )
}

export function ReceiptTableRow({ children }: { children: ReactNode }) {
  return <tr className="border-b border-receipt-field-line">{children}</tr>
}

export function ReceiptTableCell({ numeric = false, children }: { numeric?: boolean; children: ReactNode }) {
  return (
    <td className={cn('px-3 py-2 text-receipt-text', numeric && 'text-right whitespace-nowrap tabular-nums')}>
      {children}
    </td>
  )
}

export function ReceiptSignatures({ labels }: { labels: [string, string] }) {
  return (
    <div className="grid grid-cols-1 gap-6 pt-4 sm:grid-cols-2 sm:gap-10">
      {labels.map((label) => (
        <div key={label} className="flex flex-col">
          <div className="h-12 border-b border-receipt-brand" />
          <p className="pt-2 text-xs text-receipt-label">{label}</p>
        </div>
      ))}
    </div>
  )
}
