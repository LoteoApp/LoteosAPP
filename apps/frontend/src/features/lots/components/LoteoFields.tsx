import { Field, FieldLabel } from '../../../shared/ui/field'
import { Input } from '../../../shared/ui/input'
import { Textarea } from '../../../shared/ui/textarea'
import AgenciesField from './AgenciesField'

export type LoteoFieldValues = {
  name: string
  location: string
  description: string
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
        <FieldLabel htmlFor="loteo-name">Nombre</FieldLabel>
        <Input
          id="loteo-name"
          name="name"
          value={values.name}
          onChange={(event) => update('name', event.target.value)}
          autoComplete="off"
        />
      </Field>

      <Field>
        <FieldLabel htmlFor="loteo-location">Ubicación/Ciudad</FieldLabel>
        <Input
          id="loteo-location"
          name="location"
          value={values.location}
          onChange={(event) => update('location', event.target.value)}
          autoComplete="off"
        />
      </Field>

      <AgenciesField selectedIds={[]} disabled />

      <Field>
        <FieldLabel htmlFor="loteo-description">Descripción</FieldLabel>
        <Textarea
          id="loteo-description"
          name="description"
          value={values.description}
          onChange={(event) => update('description', event.target.value)}
          rows={2}
        />
      </Field>
    </div>
  )
}
