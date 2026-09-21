import { Printer } from 'lucide-react'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card'
import { formatCurrency } from '../../../shared/lib/formatCurrency'
import { formatDate } from '../../../shared/lib/formatDate'
import { PAYMENT_MEDIUM_LABELS, PAYMENT_TYPE_LABELS, clientLabel, paymentItemsLabel, type Payment } from '../types'

type PaymentsHistoryProps = {
  cobros: Payment[]
  onPrint: (payment: Payment) => void
}

export default function PaymentsHistory({ cobros, onPrint }: PaymentsHistoryProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Cobros registrados</CardTitle>
        <CardDescription>
          {cobros.length === 0
            ? 'Todavía no se registraron cobros sobre esta venta.'
            : `${cobros.length} ${cobros.length === 1 ? 'cobro' : 'cobros'}, del más reciente al más antiguo.`}
        </CardDescription>
      </CardHeader>
      {cobros.length > 0 && (
        <CardContent>
          <ul className="grid gap-3">
            {cobros.map((cobro) => (
              <li
                key={cobro.id}
                className="grid gap-2 rounded-lg border border-border px-4 py-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
              >
                <div className="grid gap-0.5">
                  <p className="font-medium tabular-nums">{formatCurrency(cobro.monto, cobro.moneda)}</p>
                  <p className="text-sm">{paymentItemsLabel(cobro)}</p>
                  <p className="text-sm text-muted-foreground">
                    {formatDate(cobro.fechaPago)} · {PAYMENT_MEDIUM_LABELS[cobro.medioPago]} · {PAYMENT_TYPE_LABELS[cobro.tipo]} ·
                    Cobró {clientLabel(cobro.usuarioAlta)}
                  </p>
                  {cobro.observacion !== undefined && cobro.observacion !== '' && (
                    <p className="text-sm text-muted-foreground">{cobro.observacion}</p>
                  )}
                </div>
                <Button
                  type="button"
                  variant="outline"
                  className="w-fit"
                  onClick={() => onPrint(cobro)}
                  aria-label={`Imprimir recibo del cobro del ${formatDate(cobro.fechaPago)} por ${formatCurrency(cobro.monto, cobro.moneda)}`}
                >
                  <Printer aria-hidden />
                  Recibo
                </Button>
              </li>
            ))}
          </ul>
        </CardContent>
      )}
    </Card>
  )
}
