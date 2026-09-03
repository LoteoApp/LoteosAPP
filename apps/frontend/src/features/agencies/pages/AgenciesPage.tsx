import { useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { Input } from '../../../shared/ui/input'
import { Label } from '../../../shared/ui/label'
import { useAgencies } from '../hooks/use-agencies'
import AgencyForm from '../components/AgencyForm'
import { isValidCuit, normalizeCuit } from '../lib/cuit'
import { resolveFormView, type FormState } from '../lib/resolveFormView'
import {
  toInmobiliariaFormValues,
  type Inmobiliaria,
  type InmobiliariaFormValues,
} from '../types'

const COMBINING_MARKS = /\p{M}/gu

function normalize(value: string): string {
  return value.normalize('NFD').replace(COMBINING_MARKS, '').toLowerCase()
}

function matchesSearch(inmobiliaria: Inmobiliaria, search: string): boolean {
  if (normalize(inmobiliaria.razonSocial).includes(search)) {
    return true
  }

  // A search made only of separators normalizes to an empty CUIT, and every
  // string contains the empty string — matching on it would list every
  // agency instead of none.
  const cuit = normalizeCuit(search)
  return cuit !== '' && inmobiliaria.cuit.includes(cuit)
}

type AgenciesPageProps = {
  accessToken: string | null
  // Hiding the write actions is UX only, not access control: the backend
  // restricts every write to administrador as well, and RLS is still pending.
  isAdministrador: boolean
}

export default function AgenciesPage({ accessToken, isAdministrador }: AgenciesPageProps) {
  const { inmobiliarias, isLoading, isSubmitting, error, create, update, remove } =
    useAgencies(accessToken ?? '')
  const [formState, setFormState] = useState<FormState>({ mode: 'closed' })
  const [search, setSearch] = useState('')
  const [confirmingBajaId, setConfirmingBajaId] = useState<string | null>(null)

  const formView = resolveFormView(formState, inmobiliarias)

  const normalizedSearch = normalize(search.trim())
  const filteredInmobiliarias = normalizedSearch
    ? inmobiliarias.filter((inmobiliaria) => matchesSearch(inmobiliaria, normalizedSearch))
    : inmobiliarias

  function validate(values: InmobiliariaFormValues, excludeId?: string): string | null {
    if (!values.razonSocial) {
      return 'Completá la razón social.'
    }

    if (values.cuit && !isValidCuit(normalizeCuit(values.cuit))) {
      return 'El CUIT debe tener 11 dígitos.'
    }

    const cuit = normalizeCuit(values.cuit)
    const cuitTaken =
      cuit !== '' &&
      inmobiliarias.some(
        (inmobiliaria) => inmobiliaria.cuit === cuit && inmobiliaria.id !== excludeId,
      )
    if (cuitTaken) {
      return 'Ya existe una inmobiliaria con ese CUIT.'
    }

    return null
  }

  async function handleCreate(values: InmobiliariaFormValues) {
    if (await create(values)) {
      setFormState({ mode: 'closed' })
    }
  }

  async function handleUpdate(id: string, values: InmobiliariaFormValues) {
    if (await update(id, values)) {
      setFormState({ mode: 'closed' })
    }
  }

  async function handleBaja(inmobiliaria: Inmobiliaria) {
    if (await remove(inmobiliaria.id)) {
      setConfirmingBajaId(null)
    }
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Inmobiliarias</h1>
        {isAdministrador && formView.mode === 'closed' && (
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
        <Card>
          <CardHeader>
            <CardTitle>
              {formView.mode === 'create' ? 'Nueva inmobiliaria' : 'Editar inmobiliaria'}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {formView.mode === 'create' ? (
              <AgencyForm
                key="create"
                submitLabel="Crear inmobiliaria"
                isSubmitting={isSubmitting}
                onSubmit={handleCreate}
                onValidate={(values) => validate(values)}
                onCancel={() => setFormState({ mode: 'closed' })}
              />
            ) : (
              <AgencyForm
                key={formView.agency.id}
                initialValue={toInmobiliariaFormValues(formView.agency)}
                submitLabel="Guardar cambios"
                isSubmitting={isSubmitting}
                onSubmit={(values) => handleUpdate(formView.agency.id, values)}
                onValidate={(values) => validate(values, formView.agency.id)}
                onCancel={() => setFormState({ mode: 'closed' })}
              />
            )}
          </CardContent>
        </Card>
      ) : (
        <>
          {isLoading && <p className="text-muted-foreground">Cargando inmobiliarias...</p>}

          {!isLoading && !error && inmobiliarias.length === 0 && (
            <p className="text-muted-foreground">No hay inmobiliarias cargadas todavía.</p>
          )}

          {inmobiliarias.length > 0 && (
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

          {inmobiliarias.length > 0 && filteredInmobiliarias.length === 0 && (
            <p className="text-muted-foreground">
              No se encontraron inmobiliarias con esa búsqueda.
            </p>
          )}

          {filteredInmobiliarias.length > 0 && (
            <ul className="flex flex-col gap-3">
              {filteredInmobiliarias.map((inmobiliaria) => (
                <li key={inmobiliaria.id}>
                  <Card>
                    <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <div>
                        <p className="font-medium text-foreground">
                          {inmobiliaria.razonSocial}
                        </p>
                        {inmobiliaria.cuit && (
                          <p className="text-sm text-muted-foreground">
                            CUIT {inmobiliaria.cuit}
                          </p>
                        )}
                        {(inmobiliaria.telefono || inmobiliaria.email) && (
                          <p className="text-sm text-muted-foreground">
                            {[inmobiliaria.telefono, inmobiliaria.email]
                              .filter(Boolean)
                              .join(' · ')}
                          </p>
                        )}
                      </div>
                      {isAdministrador && (
                        <div className="flex flex-wrap items-center gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() =>
                              setFormState({ mode: 'edit', id: inmobiliaria.id })
                            }
                          >
                            Editar
                          </Button>
                          {confirmingBajaId === inmobiliaria.id ? (
                            <>
                              <span className="text-sm text-muted-foreground">
                                ¿Confirmar baja?
                              </span>
                              <Button
                                variant="destructive"
                                size="sm"
                                disabled={isSubmitting}
                                onClick={() => handleBaja(inmobiliaria)}
                              >
                                Confirmar
                              </Button>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setConfirmingBajaId(null)}
                              >
                                Cancelar
                              </Button>
                            </>
                          ) : (
                            <Button
                              variant="destructive"
                              size="sm"
                              onClick={() => setConfirmingBajaId(inmobiliaria.id)}
                            >
                              Dar de baja
                            </Button>
                          )}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  )
}
