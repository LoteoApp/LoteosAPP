import type { DxfPoint, LoteoLote } from '../types'
import { polygonArea } from './polygonGeometry'

export const MAX_LOTE_NUMBER_LENGTH = 32
export const MAX_LOTE_FEATURES_LENGTH = 2000
export const MAX_LOTE_PRICE = 999_999_999_999.99
export const MAX_LOTE_AREA = 99_999_999.9999
export const LOTE_PRICE_DECIMALS = 2
export const LOTE_AREA_DECIMALS = 4
export const DEFAULT_LOTE_CURRENCIES = ['ARS', 'USD'] as const

export type LoteFormValues = {
  numero: string
  precio: string
  moneda: string
  superficie: string
  caracteristicas: string
}

export type UpdateLotePayload = {
  numero: string
  precio: number | null
  moneda: string
  superficie: number | null
  caracteristicas: string
}

export type LoteFormErrors = Partial<Record<keyof LoteFormValues, string>>

const NUMERO_INVALID_MESSAGE =
  'El número de lote es obligatorio y no puede superar los 32 caracteres'
const PRECIO_INVALID_MESSAGE =
  'El precio debe ser un monto no negativo, de hasta 2 decimales y menor a 1.000.000.000.000'
const MONEDA_INVALID_MESSAGE = 'La moneda debe ser un código de tres letras'
const MONEDA_REQUIRED_MESSAGE = 'Indicá la moneda del precio.'
const SUPERFICIE_INVALID_MESSAGE =
  'La superficie debe ser mayor a cero, de hasta 4 decimales y menor a 100.000.000'
const FEATURES_TOO_LONG_MESSAGE =
  'Las características del lote no pueden superar los 2000 caracteres'

export function toLoteFormValues(lote: LoteoLote): LoteFormValues {
  return {
    numero: lote.numero,
    precio: lote.precio === null ? '' : formatPrecioInput(lote.precio),
    moneda: lote.precio === null ? '' : lote.moneda || 'ARS',
    superficie:
      lote.superficie === null
        ? (superficieFromPolygon(lote.poligono) ?? '')
        : String(lote.superficie),
    caracteristicas: lote.caracteristicas,
  }
}

export function formatPrecioInput(value: number): string {
  const negative = value < 0
  const [intRaw, fracRaw] = Math.abs(value).toFixed(LOTE_PRICE_DECIMALS).split('.')
  const grouped = groupThousands(intRaw)
  const frac = fracRaw.replace(/0+$/, '')
  const body = frac === '' ? grouped : `${grouped},${frac}`
  return negative ? `-${body}` : body
}

const MAX_PRECIO_INT_DIGITS = 12

export function maskPrecioInput(raw: string): string {
  const cleaned = raw.replace(/[^\d.,]/g, '')
  if (cleaned === '') {
    return ''
  }

  let working = cleaned
  const commaAt = working.indexOf(',')
  if (commaAt !== -1) {
    working = working.slice(0, commaAt + 1) + working.slice(commaAt + 1).replace(/[.,]/g, '')
  }

  let keepDecimal = working.includes(',')
  if (!keepDecimal && working.endsWith('.')) {
    keepDecimal = true
    working = `${working.slice(0, -1)},`
  }

  working = working.replace(/\./g, '')
  const comma = working.indexOf(',')
  let intDigits = stripLeadingZeros(
    (comma === -1 ? working : working.slice(0, comma)).replace(/\D/g, ''),
  ).slice(0, MAX_PRECIO_INT_DIGITS)
  const fracDigits = (comma === -1 ? '' : working.slice(comma + 1).replace(/\D/g, '')).slice(
    0,
    LOTE_PRICE_DECIMALS,
  )

  if (intDigits === '' && fracDigits === '' && !keepDecimal) {
    return ''
  }
  if (intDigits === '') {
    intDigits = '0'
  }

  const grouped = groupThousands(intDigits)
  return keepDecimal ? `${grouped},${fracDigits}` : grouped
}

export function maskPrecioTyping(
  raw: string,
  cursor: number,
): { value: string; cursor: number } {
  const significant = countPrecioSignificant(raw.slice(0, Math.max(0, cursor)))
  const value = maskPrecioInput(raw)
  return { value, cursor: cursorAfterSignificant(value, significant) }
}

function groupThousands(intDigits: string): string {
  return intDigits.replace(/\B(?=(\d{3})+(?!\d))/g, '.')
}

function stripLeadingZeros(digits: string): string {
  const stripped = digits.replace(/^0+/, '')
  if (stripped !== '') {
    return stripped
  }
  return digits === '' ? '' : '0'
}

function countPrecioSignificant(raw: string): number {
  const hasComma = raw.includes(',')
  let count = 0
  for (let i = 0; i < raw.length; i++) {
    const ch = raw[i]
    if (ch >= '0' && ch <= '9') {
      count += 1
    } else if (ch === ',') {
      count += 1
    } else if (ch === '.' && !hasComma && i === raw.length - 1) {
      count += 1
    }
  }
  return count
}

function cursorAfterSignificant(formatted: string, significant: number): number {
  if (significant <= 0) {
    return 0
  }
  let seen = 0
  for (let i = 0; i < formatted.length; i++) {
    if (formatted[i] === '.') {
      continue
    }
    seen += 1
    if (seen >= significant) {
      return i + 1
    }
  }
  return formatted.length
}

export function superficieFromPolygon(vertices: DxfPoint[]): string | null {
  const area = polygonArea(vertices)
  if (area === null) {
    return null
  }
  return area.toFixed(LOTE_AREA_DECIMALS).replace(/\.?0+$/, '')
}

export function currencyOptions(moneda: string): string[] {
  if (moneda === '' || DEFAULT_LOTE_CURRENCIES.includes(moneda as 'ARS' | 'USD')) {
    return [...DEFAULT_LOTE_CURRENCIES]
  }
  return [...DEFAULT_LOTE_CURRENCIES, moneda]
}

export function validateLoteForm(
  values: LoteFormValues,
): { ok: true; payload: UpdateLotePayload } | { ok: false; errors: LoteFormErrors } {
  const errors: LoteFormErrors = {}
  const numero = values.numero.trim()
  const moneda = values.moneda.trim().toUpperCase()
  const caracteristicas = values.caracteristicas.trim()

  if (numero === '' || runeCount(numero) > MAX_LOTE_NUMBER_LENGTH) {
    errors.numero = NUMERO_INVALID_MESSAGE
  }

  const precio = parsePrecio(values.precio)
  if (!precio.ok) {
    errors.precio = PRECIO_INVALID_MESSAGE
  } else if (precio.value !== null) {
    if (
      precio.value < 0 ||
      precio.value > MAX_LOTE_PRICE ||
      precio.decimals > LOTE_PRICE_DECIMALS
    ) {
      errors.precio = PRECIO_INVALID_MESSAGE
    }
  }

  if (moneda !== '' && !isCurrencyCode(moneda)) {
    errors.moneda = MONEDA_INVALID_MESSAGE
  } else if (precio.ok && precio.value !== null && moneda === '') {
    errors.moneda = MONEDA_REQUIRED_MESSAGE
  } else if (precio.ok && precio.value === null && moneda !== '') {
    errors.moneda = 'La moneda solo puede indicarse junto con un precio'
  }

  const superficie = parseAmount(values.superficie)
  if (!superficie.ok) {
    errors.superficie = SUPERFICIE_INVALID_MESSAGE
  } else if (superficie.value !== null) {
    if (
      superficie.value <= 0 ||
      superficie.value > MAX_LOTE_AREA ||
      superficie.decimals > LOTE_AREA_DECIMALS
    ) {
      errors.superficie = SUPERFICIE_INVALID_MESSAGE
    }
  }

  if (runeCount(caracteristicas) > MAX_LOTE_FEATURES_LENGTH) {
    errors.caracteristicas = FEATURES_TOO_LONG_MESSAGE
  }

  if (Object.keys(errors).length > 0) {
    return { ok: false, errors }
  }

  return {
    ok: true,
    payload: {
      numero,
      precio: precio.ok ? precio.value : null,
      moneda,
      superficie: superficie.ok ? superficie.value : null,
      caracteristicas,
    },
  }
}

function runeCount(value: string): number {
  return Array.from(value).length
}

function isCurrencyCode(currency: string): boolean {
  return /^[A-Z]{3}$/.test(currency)
}

function parseAmount(
  raw: string,
): { ok: true; value: number | null; decimals: number } | { ok: false } {
  const trimmed = raw.trim()
  if (trimmed === '') {
    return { ok: true, value: null, decimals: 0 }
  }

  const normalized =
    trimmed.includes(',') && !trimmed.includes('.')
      ? trimmed.replace(',', '.')
      : trimmed.replace(/,/g, '')

  return numberFromNormalized(normalized)
}

function parsePrecio(
  raw: string,
): { ok: true; value: number | null; decimals: number } | { ok: false } {
  const trimmed = raw.trim()
  if (trimmed === '') {
    return { ok: true, value: null, decimals: 0 }
  }

  const negative = trimmed.startsWith('-')
  const unsigned = negative ? trimmed.slice(1).trim() : trimmed
  if (unsigned === '') {
    return { ok: false }
  }

  let intDigits: string
  let fracDigits = ''

  if (unsigned.includes(',')) {
    const pieces = unsigned.split(',')
    if (pieces.length !== 2) {
      return { ok: false }
    }
    intDigits = pieces[0].replace(/\./g, '')
    fracDigits = pieces[1]
    if (fracDigits.includes('.')) {
      return { ok: false }
    }
  } else if (unsigned.includes('.')) {
    const pieces = unsigned.split('.')
    if (pieces.length > 2 || pieces[1].length === 3) {
      intDigits = pieces.join('')
    } else {
      intDigits = pieces[0]
      fracDigits = pieces[1]
    }
  } else {
    intDigits = unsigned
  }

  if (intDigits === '' || !/^\d+$/.test(intDigits) || (fracDigits !== '' && !/^\d+$/.test(fracDigits))) {
    return { ok: false }
  }

  const normalized = `${negative ? '-' : ''}${intDigits}${fracDigits === '' ? '' : `.${fracDigits}`}`
  return numberFromNormalized(normalized)
}

export function reformatPrecioInput(raw: string): string {
  const parsed = parsePrecio(raw)
  if (!parsed.ok || parsed.value === null) {
    return raw
  }
  return formatPrecioInput(parsed.value)
}

function numberFromNormalized(
  normalized: string,
): { ok: true; value: number | null; decimals: number } | { ok: false } {
  if (!/^-?\d+(\.\d+)?$/.test(normalized)) {
    return { ok: false }
  }

  const value = Number(normalized)
  if (!Number.isFinite(value)) {
    return { ok: false }
  }

  const dot = normalized.indexOf('.')
  const decimals = dot === -1 ? 0 : normalized.length - dot - 1
  return { ok: true, value, decimals }
}
