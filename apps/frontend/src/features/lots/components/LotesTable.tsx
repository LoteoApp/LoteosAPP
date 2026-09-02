import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../../../shared/ui/table'
import { formatArea } from '../../../shared/lib/formatArea'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import type { LoteoLote } from '../types'

type LotesTableProps = {
  lotes: LoteoLote[]
  manzanaNumberById: ReadonlyMap<string, string>
}

const EMPTY_VALUE = '—'

function priceOf(lote: LoteoLote): string {
  return lote.precio === null ? EMPTY_VALUE : formatCurrency(lote.precio, lote.moneda)
}

function areaOf(lote: LoteoLote): string {
  return lote.superficie === null ? EMPTY_VALUE : formatArea(lote.superficie)
}

export default function LotesTable({ lotes, manzanaNumberById }: LotesTableProps) {
  return (
    <Table>
      <TableCaption>
        {lotes.length === 1 ? '1 lote' : `${lotes.length} lotes`}
      </TableCaption>
      <TableHeader>
        <TableRow>
          <TableHead>Manzana</TableHead>
          <TableHead>Lote</TableHead>
          <TableHead className="text-right">Superficie</TableHead>
          <TableHead className="text-right">Precio</TableHead>
          <TableHead className="hidden md:table-cell">Características</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {lotes.length === 0 ? (
          <TableRow>
            <TableCell colSpan={5} className="text-muted-foreground">
              No hay lotes para mostrar.
            </TableCell>
          </TableRow>
        ) : (
          lotes.map((lote) => (
            <TableRow key={lote.id}>
              <TableCell>{manzanaNumberById.get(lote.manzanaId) ?? EMPTY_VALUE}</TableCell>
              <TableCell>{lote.numero || EMPTY_VALUE}</TableCell>
              <TableCell className="text-right tabular-nums">{areaOf(lote)}</TableCell>
              <TableCell className="text-right tabular-nums">{priceOf(lote)}</TableCell>
              <TableCell className="hidden max-w-xs truncate md:table-cell">
                {lote.caracteristicas || EMPTY_VALUE}
              </TableCell>
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  )
}
