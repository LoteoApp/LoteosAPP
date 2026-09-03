import { useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Input } from '../../../shared/ui/input'
import { Label } from '../../../shared/ui/label'
import { useAgencies } from '../hooks/use-agencies'
import AgencyEditor from '../components/AgencyEditor'
import AgencyList from '../components/AgencyList'
import { isValidCuit, normalizeCuit } from '../lib/cuit'
import { resolveFormView, type FormState } from '../lib/resolveFormView'
import type { Agency, AgencyFormValues } from '../types'

const COMBINING_MARKS = /\p{M}/gu

function normalize(value: string): string {
  return value.normalize('NFD').replace(COMBINING_MARKS, '').toLowerCase()
}

function matchesSearch(agency: Agency, search: string): boolean {
  if (normalize(agency.razonSocial).includes(search)) {
    return true
  }

  // A search made only of separators normalizes to an empty CUIT, and every
  // string contains the empty string — matching on it would list every
  // agency instead of none.
  const cuit = normalizeCuit(search)
  return cuit !== '' && agency.cuit.includes(cuit)
}

type AgenciesPageProps = {
  accessToken: string | null
  // Hiding the write actions is UX only, not access control: the backend
  // restricts every write to administrador as well, and RLS is still pending.
  isAdmin: boolean
}

export default function AgenciesPage({ accessToken, isAdmin }: AgenciesPageProps) {
  const { agencies, isLoading, isSubmitting, error, create, update, remove } = useAgencies(
    accessToken ?? '',
  )
  const [formState, setFormState] = useState<FormState>({ mode: 'closed' })
  const [search, setSearch] = useState('')

  const formView = resolveFormView(formState, agencies)
  const editing = formView.mode === 'edit' ? formView.agency : undefined

  const normalizedSearch = normalize(search.trim())
  const filteredAgencies = normalizedSearch
    ? agencies.filter((agency) => matchesSearch(agency, normalizedSearch))
    : agencies

  function validate(values: AgencyFormValues): string | null {
    if (!values.razonSocial) {
      return 'Completá la razón social.'
    }

    const cuit = normalizeCuit(values.cuit)
    if (values.cuit && !isValidCuit(cuit)) {
      return 'El CUIT debe tener 11 dígitos.'
    }

    const cuitTaken =
      cuit !== '' &&
      agencies.some((agency) => agency.cuit === cuit && agency.id !== editing?.id)
    if (cuitTaken) {
      return 'Ya existe una inmobiliaria con ese CUIT.'
    }

    return null
  }

  async function handleSubmit(values: AgencyFormValues) {
    const saved = editing ? await update(editing.id, values) : await create(values)
    if (saved) {
      setFormState({ mode: 'closed' })
    }
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Inmobiliarias</h1>
        {isAdmin && formView.mode === 'closed' && (
          <Button onClick={() => setFormState({ mode: 'create' })}>Nueva inmobiliaria</Button>
        )}
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>No se pudo completar la operación</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {formView.mode !== 'closed' ? (
        <AgencyEditor
          agency={editing}
          isSubmitting={isSubmitting}
          onSubmit={handleSubmit}
          onValidate={validate}
          onCancel={() => setFormState({ mode: 'closed' })}
        />
      ) : (
        <>
          {isLoading && <p className="text-muted-foreground">Cargando inmobiliarias...</p>}

          {!isLoading && !error && agencies.length === 0 && (
            <p className="text-muted-foreground">No hay inmobiliarias cargadas todavía.</p>
          )}

          {agencies.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="buscar-inmobiliaria">Buscar</Label>
              <Input
                id="buscar-inmobiliaria"
                type="search"
                placeholder="Razón social o CUIT"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                className="max-w-sm"
              />
            </div>
          )}

          {agencies.length > 0 && filteredAgencies.length === 0 && (
            <p className="text-muted-foreground">
              No se encontraron inmobiliarias con esa búsqueda.
            </p>
          )}

          {filteredAgencies.length > 0 && (
            <AgencyList
              agencies={filteredAgencies}
              canManage={isAdmin}
              isSubmitting={isSubmitting}
              onEdit={(id) => setFormState({ mode: 'edit', id })}
              onDeactivate={remove}
            />
          )}
        </>
      )}
    </section>
  )
}
