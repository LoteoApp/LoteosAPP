import { formatCurrency } from '../../../shared/lib/formatCurrency'

// "12 de US$ 383,33": every cuota is the same amount.
export function installmentsLabel(count: number, amount: number, currency: string): string {
  return `${count} de ${formatCurrency(amount, currency)}`
}
