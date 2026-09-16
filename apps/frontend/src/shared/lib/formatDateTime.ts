export function formatDateTime(isoDate: string): string {
  const date = new Date(isoDate)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat('es-AR', {
    dateStyle: 'short',
    timeStyle: 'short',
    timeZone: 'America/Argentina/Buenos_Aires',
  }).format(date)
}
