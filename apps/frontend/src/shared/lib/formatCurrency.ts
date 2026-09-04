// `moneda` is free text in the database (lotes.moneda has no CHECK), so an
// unknown code makes Intl.NumberFormat throw a RangeError. Fall back to
// "<code> <number>" then, and to the bare number when no currency is set.
export function formatCurrency(amount: number, currency: string): string {
  const code = currency.trim().toUpperCase()

  if (code !== '') {
    try {
      return new Intl.NumberFormat('es-AR', {
        style: 'currency',
        currency: code,
      }).format(amount)
    } catch {
      // Not a valid ISO 4217 code; fall through to the plain formatting.
    }
  }

  const number = new Intl.NumberFormat('es-AR', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)

  return code === '' ? number : `${code} ${number}`
}
