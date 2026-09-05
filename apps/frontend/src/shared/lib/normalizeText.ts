const DIACRITICS_PATTERN = /\p{Diacritic}/gu

export function normalizeText(value: string): string {
  return value.normalize('NFD').replace(DIACRITICS_PATTERN, '').toLowerCase()
}
