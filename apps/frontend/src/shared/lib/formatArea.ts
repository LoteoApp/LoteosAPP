export function formatArea(squareMeters: number): string {
  const number = new Intl.NumberFormat('es-AR', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(squareMeters)

  return `${number} m²`
}
