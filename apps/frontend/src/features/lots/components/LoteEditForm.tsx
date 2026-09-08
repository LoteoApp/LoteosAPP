import { useState, type ChangeEvent, type FormEvent } from 'react'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Field, FieldDescription, FieldError, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { SaveNotice, useSaveNotice } from '../../../shared/ui/save-notice'
import { Textarea } from '../../../shared/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '../../../shared/ui/toggle-group'
import type { UpdateLoteState } from '../hooks/use-update-lote'
import {
  MAX_LOTE_FEATURES_LENGTH,
  currencyOptions,
  maskPrecioTyping,
  reformatPrecioInput,
  toLoteFormValues,
  validateLoteForm,
  type LoteFormErrors,
  type LoteFormValues,
  type UpdateLotePayload,
} from '../lib/loteFormValues'
import type { LoteoLote } from '../types'

type LoteEditFormProps = {
  lote: LoteoLote
  updateState: UpdateLoteState
  onSave: (payload: UpdateLotePayload) => Promise<boolean>
}

export default function LoteEditForm({ lote, updateState, onSave }: LoteEditFormProps) {
  const [values, setValues] = useState<LoteFormValues>(() => toLoteFormValues(lote))
  const [errors, setErrors] = useState<LoteFormErrors>({})
  const notice = useSaveNotice()
  const [trackedLoteId, setTrackedLoteId] = useState(lote.id)

  if (lote.id !== trackedLoteId) {
    setTrackedLoteId(lote.id)
    setValues(toLoteFormValues(lote))
    setErrors({})
    notice.clear()
  }

  const saving = updateState.status === 'saving'
  const serverMessage = updateState.status === 'error' ? updateState.message : undefined
  const serverField = updateState.status === 'error' ? updateState.field : undefined
  const currencies = currencyOptions(values.moneda)

  function update<Key extends keyof LoteFormValues>(key: Key, value: LoteFormValues[Key]) {
    setValues((current) => ({ ...current, [key]: value }))
    notice.clear()
  }

  function handlePrecioChange(event: ChangeEvent<HTMLInputElement>) {
    const input = event.currentTarget
    const masked = maskPrecioTyping(input.value, input.selectionStart ?? input.value.length)
    update('precio', masked.value)
    if (masked.value === '') {
      update('moneda', '')
    }
    queueMicrotask(() => {
      input.setSelectionRange(masked.cursor, masked.cursor)
    })
  }

  function formatPrecioOnBlur() {
    const formatted = reformatPrecioInput(values.precio)
    if (formatted !== values.precio) {
      update('precio', formatted)
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const result = validateLoteForm(values)
    if (!result.ok) {
      setErrors(result.errors)
      notice.clear()
      return
    }

    setErrors({})
    const ok = await onSave(result.payload)
    if (ok) {
      notice.show()
    } else {
      notice.clear()
    }
  }

  function fieldError(field: keyof LoteFormValues): string | undefined {
    return errors[field] ?? (serverField === field ? serverMessage : undefined)
  }

  const numeroError = fieldError('numero')
  const precioError = fieldError('precio')
  const monedaError = fieldError('moneda')
  const superficieError = fieldError('superficie')
  const caracteristicasError = fieldError('caracteristicas')
  const bannerMessage = serverMessage && !serverField ? serverMessage : undefined

  return (
    <form className="flex flex-col gap-3" onSubmit={handleSubmit}>
      <SaveNotice token={notice.token}>Lote guardado</SaveNotice>
      {bannerMessage && (
        <Alert variant="destructive">
          <AlertDescription>{bannerMessage}</AlertDescription>
        </Alert>
      )}

      <Field data-invalid={numeroError ? true : undefined}>
        <FieldLabel htmlFor="lote-numero">Número</FieldLabel>
        <Input
          id="lote-numero"
          name="numero"
          value={values.numero}
          onChange={(event) => update('numero', event.target.value)}
          autoComplete="off"
          aria-invalid={numeroError ? true : undefined}
        />
        <FieldError>{numeroError}</FieldError>
      </Field>

      <Field data-invalid={precioError ? true : undefined}>
        <FieldLabel htmlFor="lote-precio">Precio</FieldLabel>
        <Input
          id="lote-precio"
          name="precio"
          inputMode="decimal"
          value={values.precio}
          onChange={handlePrecioChange}
          onBlur={formatPrecioOnBlur}
          autoComplete="off"
          aria-invalid={precioError ? true : undefined}
        />
        <FieldError>{precioError}</FieldError>
      </Field>

      <Field data-invalid={monedaError ? true : undefined}>
        <FieldLabel id="lote-moneda-label">Moneda</FieldLabel>
        <ToggleGroup
          variant="outline"
          className="w-full md:w-fit"
          value={values.moneda === '' ? [] : [values.moneda]}
          onValueChange={(next) => {
            const selected = next[0]
            if (selected) {
              update('moneda', selected)
            }
          }}
          aria-labelledby="lote-moneda-label"
        >
          {currencies.map((currency) => (
            <ToggleGroupItem
              key={currency}
              value={currency}
              className="min-h-11 min-w-11 flex-1 touch-manipulation md:min-h-8 md:flex-none"
            >
              {currency}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
        <FieldError>{monedaError}</FieldError>
      </Field>

      <Field data-invalid={superficieError ? true : undefined}>
        <FieldLabel htmlFor="lote-superficie">Superficie (m²)</FieldLabel>
        <Input
          id="lote-superficie"
          name="superficie"
          inputMode="decimal"
          value={values.superficie}
          onChange={(event) => update('superficie', event.target.value)}
          autoComplete="off"
          aria-invalid={superficieError ? true : undefined}
        />
        {lote.superficie === null && values.superficie !== '' ? (
          <FieldDescription>Calculada del plano. Podés modificarla.</FieldDescription>
        ) : null}
        <FieldError>{superficieError}</FieldError>
      </Field>

      <Field data-invalid={caracteristicasError ? true : undefined}>
        <FieldLabel htmlFor="lote-caracteristicas">Características</FieldLabel>
        <Textarea
          id="lote-caracteristicas"
          name="caracteristicas"
          value={values.caracteristicas}
          onChange={(event) => update('caracteristicas', event.target.value)}
          rows={3}
          aria-invalid={caracteristicasError ? true : undefined}
        />
        <FieldDescription>
          {Array.from(values.caracteristicas).length} / {MAX_LOTE_FEATURES_LENGTH}
        </FieldDescription>
        <FieldError>{caracteristicasError}</FieldError>
      </Field>

      <Button type="submit" className="min-h-11 md:min-h-8" disabled={saving}>
        {saving ? 'Guardando…' : 'Guardar'}
      </Button>
    </form>
  )
}
