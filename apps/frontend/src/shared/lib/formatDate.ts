// Takes the RFC 3339 string the API returns and renders it as a short es-AR
// date. Returns an empty string for an unparseable value instead of
// "Invalid Date".
export function formatDate(isoDate: string): string {
  const date = new Date(isoDate)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  return new Intl.DateTimeFormat('es-AR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  }).format(date)
}
