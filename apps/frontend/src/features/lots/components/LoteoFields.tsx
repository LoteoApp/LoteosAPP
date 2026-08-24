import { Field, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { Textarea } from '../../../shared/ui/textarea'
import InmobiliariasField from './InmobiliariasField'

export type LoteoFieldValues = {
  nombre: string
  ubicacion: string
  descripcion: string
  inmobiliariaIds: string[]
}

type LoteoFieldsProps = {
  values: LoteoFieldValues
  onChange: (values: LoteoFieldValues) => void
}

export default function LoteoFields({ values, onChange }: LoteoFieldsProps) {
  function update<Key extends keyof LoteoFieldValues>(key: Key, value: LoteoFieldValues[Key]) {
    onChange({ ...values, [key]: value })
  }

  return (
    <div className="flex flex-col gap-3">
      <Field>
        <FieldLabel htmlFor="loteo-nombre">Nombre</FieldLabel>
        <Input
          id="loteo-nombre"
          name="nombre"
          value={values.nombre}
          onChange={(event) => update('nombre', event.target.value)}
          autoComplete="off"
        />
      </Field>

      <Field>
        <FieldLabel htmlFor="loteo-ubicacion">Ubicación/Ciudad</FieldLabel>
        <Input
          id="loteo-ubicacion"
          name="ubicacion"
          value={values.ubicacion}
          onChange={(event) => update('ubicacion', event.target.value)}
          autoComplete="off"
        />
      </Field>

      <InmobiliariasField
        selectedIds={values.inmobiliariaIds}
        onChange={(inmobiliariaIds) => update('inmobiliariaIds', inmobiliariaIds)}
      />

      <Field>
        <FieldLabel htmlFor="loteo-descripcion">Descripción</FieldLabel>
        <Textarea
          id="loteo-descripcion"
          name="descripcion"
          value={values.descripcion}
          onChange={(event) => update('descripcion', event.target.value)}
          rows={2}
        />
      </Field>
    </div>
  )
}
