export function formatPercent(value: number): string {
  const number = new Intl.NumberFormat('es-AR', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 4,
  }).format(value)

  return `${number} %`
}
