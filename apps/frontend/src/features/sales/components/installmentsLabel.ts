import { formatCurrency } from '../../../shared/lib/formatCurrency'

// "12 de US$ 383,33", plus "(la última de US$ 383,37)" when the last cuota
// absorbs the rounding remainder, so what is printed adds up to what is
// persisted without hiding the regular amount behind an arithmetic.
export function installmentsLabel(
  count: number,
  regular: number,
  last: number,
  currency: string,
): string {
  const base = `${count} de ${formatCurrency(regular, currency)}`
  if (count === 1 || last === regular) {
    return base
  }
  return `${base} (la última de ${formatCurrency(last, currency)})`
}
