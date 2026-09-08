import { useState, type FormEvent } from 'react'
import { Alert, AlertDescription } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Field, FieldError, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { SaveNotice, useSaveNotice } from '../../../shared/ui/save-notice'
import { ToggleGroup, ToggleGroupItem } from '../../../shared/ui/toggle-group'
import type { UpdateCallePayload } from '../api/update-calle'
import type { UpdateCalleState } from '../hooks/use-update-calle'
import type { LoteoCalle } from '../types'

const CALLE_TIPOS = [
  { value: 'asfalto', label: 'Asfalto' },
  { value: 'tierra', label: 'Tierra' },
  { value: 'brosa', label: 'Brosa' },
  { value: 'granito', label: 'Granito' },
] as const

const MAX_CALLE_NAME_LENGTH = 64

type CalleFormValues = {
  nombre: string
  tipo: string
}

type CalleEditFormProps = {
  calle: LoteoCalle
  updateState: UpdateCalleState
  onSave: (payload: UpdateCallePayload) => Promise<boolean>
}

export default function CalleEditForm({ calle, updateState, onSave }: CalleEditFormProps) {
  const [values, setValues] = useState<CalleFormValues>(() => toValues(calle))
  const [nombreError, setNombreError] = useState<string | undefined>()
  const notice = useSaveNotice()
  const [trackedId, setTrackedId] = useState(calle.id)

  if (calle.id !== trackedId) {
    setTrackedId(calle.id)
    setValues(toValues(calle))
    setNombreError(undefined)
    notice.clear()
  }

  const saving = updateState.status === 'saving'
  const serverMessage = updateState.status === 'error' ? updateState.message : undefined
  const serverField = updateState.status === 'error' ? updateState.field : undefined
  const nombreFieldError =
    nombreError ?? (serverField === 'nombre' ? serverMessage : undefined)
  const tipoFieldError = serverField === 'tipo' ? serverMessage : undefined
  const bannerMessage = serverMessage && !serverField ? serverMessage : undefined

  function update<Key extends keyof CalleFormValues>(key: Key, value: CalleFormValues[Key]) {
    setValues((current) => ({ ...current, [key]: value }))
    notice.clear()
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const nombre = values.nombre.trim()
    if (nombre === '' || Array.from(nombre).length > MAX_CALLE_NAME_LENGTH) {
      setNombreError('El nombre de la calle es obligatorio y no puede superar los 64 caracteres')
      notice.clear()
      return
    }

    setNombreError(undefined)
    const ok = await onSave({ nombre, tipo: values.tipo })
    if (ok) {
      notice.show()
    } else {
      notice.clear()
    }
  }

  return (
    <form className="flex flex-col gap-3" onSubmit={handleSubmit}>
      <SaveNotice token={notice.token}>Calle guardada</SaveNotice>
      {bannerMessage && (
        <Alert variant="destructive">
          <AlertDescription>{bannerMessage}</AlertDescription>
        </Alert>
      )}

      <Field data-invalid={nombreFieldError ? true : undefined}>
        <FieldLabel htmlFor="calle-nombre">Nombre</FieldLabel>
        <Input
          id="calle-nombre"
          name="nombre"
          value={values.nombre}
          onChange={(event) => update('nombre', event.target.value)}
          autoComplete="off"
          aria-invalid={nombreFieldError ? true : undefined}
        />
        <FieldError>{nombreFieldError}</FieldError>
      </Field>

      <Field data-invalid={tipoFieldError ? true : undefined}>
        <FieldLabel id="calle-tipo-label">Tipo</FieldLabel>
        <ToggleGroup
          variant="outline"
          className="w-full"
          value={values.tipo === '' ? [] : [values.tipo]}
          onValueChange={(next) => {
            update('tipo', next[0] ?? '')
          }}
          aria-labelledby="calle-tipo-label"
        >
          {CALLE_TIPOS.map((option) => (
            <ToggleGroupItem
              key={option.value}
              value={option.value}
              className="min-h-11 min-w-11 flex-1 touch-manipulation md:min-h-8"
            >
              {option.label}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
        <FieldError>{tipoFieldError}</FieldError>
      </Field>

      <Button type="submit" className="min-h-11 md:min-h-8" disabled={saving}>
        {saving ? 'Guardando…' : 'Guardar'}
      </Button>
    </form>
  )
}

function toValues(calle: LoteoCalle): CalleFormValues {
  return {
    nombre: calle.nombre,
    tipo: calle.tipo,
  }
}
