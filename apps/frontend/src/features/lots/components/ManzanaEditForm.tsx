import { useState, type FormEvent } from 'react'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Field, FieldDescription, FieldError, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { SaveNotice, useSaveNotice } from '../../../shared/ui/save-notice'
import { ToggleGroup, ToggleGroupItem } from '../../../shared/ui/toggle-group'
import type { UpdateManzanaPayload } from '../api/update-manzana'
import type { UpdateManzanaState } from '../hooks/use-update-manzana'
import type { LoteoCalle, LoteoManzana } from '../types'

const SERVICE_OPTIONS = [
  { key: 'tieneAgua', label: 'Agua' },
  { key: 'tieneCloaca', label: 'Cloaca' },
  { key: 'tieneLuz', label: 'Luz' },
  { key: 'tieneGas', label: 'Gas' },
] as const

const MAX_MANZANA_CALLES = 4

type ManzanaFormValues = {
  numero: string
  tieneAgua: boolean
  tieneCloaca: boolean
  tieneLuz: boolean
  tieneGas: boolean
  calleIds: string[]
}

type ManzanaEditFormProps = {
  manzana: LoteoManzana
  calles: LoteoCalle[]
  loteCount: number
  updateState: UpdateManzanaState
  onSave: (payload: UpdateManzanaPayload) => Promise<boolean>
}

export default function ManzanaEditForm({
  manzana,
  calles,
  loteCount,
  updateState,
  onSave,
}: ManzanaEditFormProps) {
  const [values, setValues] = useState<ManzanaFormValues>(() => toValues(manzana))
  const [numeroError, setNumeroError] = useState<string | undefined>()
  const notice = useSaveNotice()
  const [trackedId, setTrackedId] = useState(manzana.id)

  if (manzana.id !== trackedId) {
    setTrackedId(manzana.id)
    setValues(toValues(manzana))
    setNumeroError(undefined)
    notice.clear()
  }

  const saving = updateState.status === 'saving'
  const serverMessage = updateState.status === 'error' ? updateState.message : undefined
  const serverField = updateState.status === 'error' ? updateState.field : undefined
  const numeroFieldError =
    numeroError ?? (serverField === 'numero' ? serverMessage : undefined)
  const callesFieldError = serverField === 'calleIds' ? serverMessage : undefined
  const bannerMessage = serverMessage && !serverField ? serverMessage : undefined

  function update<Key extends keyof ManzanaFormValues>(key: Key, value: ManzanaFormValues[Key]) {
    setValues((current) => ({ ...current, [key]: value }))
    notice.clear()
  }

  function toggleCalle(calleId: string) {
    const selected = values.calleIds.includes(calleId)
    if (selected) {
      update(
        'calleIds',
        values.calleIds.filter((id) => id !== calleId),
      )
      return
    }
    update('calleIds', [...values.calleIds, calleId])
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const numero = values.numero.trim()
    if (numero === '' || Array.from(numero).length > 32) {
      setNumeroError('El número de manzana es obligatorio y no puede superar los 32 caracteres')
      notice.clear()
      return
    }

    setNumeroError(undefined)
    const ok = await onSave({
      numero,
      tieneAgua: values.tieneAgua,
      tieneCloaca: values.tieneCloaca,
      tieneLuz: values.tieneLuz,
      tieneGas: values.tieneGas,
      calleIds: values.calleIds,
    })
    if (ok) {
      notice.show()
    } else {
      notice.clear()
    }
  }

  const pressedServices = SERVICE_OPTIONS.filter((option) => values[option.key]).map(
    (option) => option.key,
  )

  return (
    <form className="flex flex-col gap-3" onSubmit={handleSubmit}>
      <SaveNotice token={notice.token}>Manzana guardada</SaveNotice>
      {bannerMessage && (
        <Alert variant="destructive">
          <AlertDescription>{bannerMessage}</AlertDescription>
        </Alert>
      )}

      <p className="text-sm text-muted-foreground">
        {loteCount === 1 ? '1 lote' : `${loteCount} lotes`} en esta manzana.
      </p>

      <Field data-invalid={numeroFieldError ? true : undefined}>
        <FieldLabel htmlFor="manzana-numero">Número</FieldLabel>
        <Input
          id="manzana-numero"
          name="numero"
          value={values.numero}
          onChange={(event) => update('numero', event.target.value)}
          autoComplete="off"
          aria-invalid={numeroFieldError ? true : undefined}
        />
        <FieldError>{numeroFieldError}</FieldError>
      </Field>

      <Field>
        <FieldLabel id="manzana-servicios-label">Servicios</FieldLabel>
        <ToggleGroup
          multiple
          variant="outline"
          className="w-full"
          value={pressedServices}
          onValueChange={(next) => {
            update('tieneAgua', next.includes('tieneAgua'))
            update('tieneCloaca', next.includes('tieneCloaca'))
            update('tieneLuz', next.includes('tieneLuz'))
            update('tieneGas', next.includes('tieneGas'))
          }}
          aria-labelledby="manzana-servicios-label"
        >
          {SERVICE_OPTIONS.map((option) => (
            <ToggleGroupItem
              key={option.key}
              value={option.key}
              className="min-h-11 min-w-11 flex-1 touch-manipulation md:min-h-8"
            >
              {option.label}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </Field>

      <Field data-invalid={callesFieldError ? true : undefined}>
        <FieldLabel>Calles que la rodean</FieldLabel>
        <FieldDescription>Hasta 4 calles del loteo.</FieldDescription>
        {calles.length === 0 ? (
          <p className="text-sm text-muted-foreground">Este loteo todavía no tiene calles.</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {calles.map((calle) => {
              const checked = values.calleIds.includes(calle.id)
              const disabled = !checked && values.calleIds.length >= MAX_MANZANA_CALLES
              return (
                <li key={calle.id}>
                  <label className="flex min-h-11 items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      className="size-4"
                      checked={checked}
                      disabled={disabled}
                      onChange={() => toggleCalle(calle.id)}
                    />
                    {calle.nombre || 'Calle sin nombre'}
                  </label>
                </li>
              )
            })}
          </ul>
        )}
        <FieldError>{callesFieldError}</FieldError>
      </Field>

      <Button type="submit" className="min-h-11 md:min-h-8" disabled={saving}>
        {saving ? 'Guardando…' : 'Guardar'}
      </Button>
    </form>
  )
}

function toValues(manzana: LoteoManzana): ManzanaFormValues {
  return {
    numero: manzana.numero,
    tieneAgua: manzana.tieneAgua,
    tieneCloaca: manzana.tieneCloaca,
    tieneLuz: manzana.tieneLuz,
    tieneGas: manzana.tieneGas,
    calleIds: [...manzana.calleIds],
  }
}
