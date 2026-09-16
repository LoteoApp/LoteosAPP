const CUIT_SEPARATORS = /[-\s.]/g
const ELEVEN_DIGITS = /^\d{11}$/

// Mirrors domain.NormalizarCUIT in the backend: the CUIT is stored as plain
// digits, so "30-71234567-8" and "30712345678" are the same agency.
export function normalizeCuit(cuit: string): string {
  return cuit.replace(CUIT_SEPARATORS, '')
}

export function isValidCuit(cuit: string): boolean {
  return ELEVEN_DIGITS.test(cuit)
}
