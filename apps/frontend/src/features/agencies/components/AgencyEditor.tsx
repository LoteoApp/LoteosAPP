import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import AgencyForm from './AgencyForm'
import { toAgencyFormValues, type Agency, type AgencyFormValues } from '../types'

type AgencyEditorProps = {
  // Absent means the editor is registering a new agency.
  agency?: Agency
  isSubmitting: boolean
  onSubmit: (values: AgencyFormValues) => void
  onValidate: (values: AgencyFormValues) => string | null
  onCancel: () => void
}

export default function AgencyEditor({
  agency,
  isSubmitting,
  onSubmit,
  onValidate,
  onCancel,
}: AgencyEditorProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{agency ? 'Editar inmobiliaria' : 'Nueva inmobiliaria'}</CardTitle>
      </CardHeader>
      <CardContent>
        <AgencyForm
          key={agency?.id ?? 'create'}
          initialValue={agency && toAgencyFormValues(agency)}
          submitLabel={agency ? 'Guardar cambios' : 'Crear inmobiliaria'}
          isSubmitting={isSubmitting}
          onSubmit={onSubmit}
          onValidate={onValidate}
          onCancel={onCancel}
        />
      </CardContent>
    </Card>
  )
}
