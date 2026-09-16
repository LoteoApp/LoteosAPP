import { formatCurrency } from '../../../shared/lib/formatCurrency'

// "12 × US$ 100,00" when every cuota is the same, or
// "11 × US$ 33,33 + 1 × US$ 33,37" when the last one absorbs the rounding
// remainder, so what is printed adds up to what is persisted.
export function installmentsLabel(
  count: number,
  regular: number,
  last: number,
  currency: string,
): string {
  if (count === 1 || last === regular) {
    return `${count} × ${formatCurrency(regular, currency)}`
  }
  return `${count - 1} × ${formatCurrency(regular, currency)} + 1 × ${formatCurrency(last, currency)}`
}
