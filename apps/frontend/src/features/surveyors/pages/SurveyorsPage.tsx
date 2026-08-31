import { useEffect, useState } from 'react'
import { useAuth } from '../../auth/hooks/use-auth'
import { getUserRole, ROLE } from '../../../shared/auth/roles'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Badge } from '../../../shared/ui/badge'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { Label } from '../../../shared/ui/label'
import {
  createSurveyor,
  deactivateSurveyor,
  listSurveyors,
  updateSurveyor,
} from '../api/surveyors'
import SurveyorForm from '../components/SurveyorForm'
import { resolveFormView, type FormState } from '../lib/resolveFormView'
import {
  fullName,
  isActive,
  toSurveyorFormValues,
  type Surveyor,
  type SurveyorFormValues,
} from '../types'

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export default function SurveyorsPage() {
  const { session, user } = useAuth()
  const token = session?.access_token ?? ''
  const isAdministrador = getUserRole(user) === ROLE.administrador

  const [surveyors, setSurveyors] = useState<Surveyor[]>([])
  const [includeInactive, setIncludeInactive] = useState(false)
  const [reloadCount, setReloadCount] = useState(0)
  const [isLoading, setIsLoading] = useState(isAdministrador)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [temporaryPassword, setTemporaryPassword] = useState<string | null>(null)
  const [formState, setFormState] = useState<FormState>({ mode: 'closed' })
  const [confirmingBajaId, setConfirmingBajaId] = useState<string | null>(null)

  const formView = resolveFormView(formState, surveyors)

  useEffect(() => {
    if (!isAdministrador) {
      return
    }

    let cancelled = false

    listSurveyors(token, includeInactive)
      .then((loaded) => {
        if (cancelled) {
          return
        }
        setSurveyors(loaded)
        setError(null)
      })
      .catch((loadError: unknown) => {
        if (!cancelled) {
          setError(messageOf(loadError))
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [isAdministrador, token, includeInactive, reloadCount])

  function refresh() {
    setIsLoading(true)
    setReloadCount((count) => count + 1)
  }

  async function handleCreate(values: SurveyorFormValues) {
    setIsSubmitting(true)
    try {
      const created = await createSurveyor(token, values)
      setTemporaryPassword(created.temporaryPassword)
      setFormState({ mode: 'closed' })
      setError(null)
      refresh()
    } catch (createError) {
      setError(messageOf(createError))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleUpdate(id: string, values: SurveyorFormValues) {
    setIsSubmitting(true)
    try {
      await updateSurveyor(token, id, { nombre: values.nombre, apellido: values.apellido })
      setFormState({ mode: 'closed' })
      setError(null)
      refresh()
    } catch (updateError) {
      setError(messageOf(updateError))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleBaja(id: string) {
    setIsSubmitting(true)
    try {
      await deactivateSurveyor(token, id)
      setConfirmingBajaId(null)
      setError(null)
      refresh()
    } catch (bajaError) {
      setError(messageOf(bajaError))
    } finally {
      setIsSubmitting(false)
    }
  }

  if (!isAdministrador) {
    return (
      <section className="flex flex-col gap-4">
        <h1 className="text-2xl font-semibold text-foreground">Agrimensores</h1>
        <Alert variant="destructive">
          <AlertTitle>No tenés permisos para esta sección</AlertTitle>
          <AlertDescription>
            Solo el administrador puede gestionar agrimensores.
          </AlertDescription>
        </Alert>
      </section>
    )
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Agrimensores</h1>
        {formView.mode === 'closed' && (
          <Button onClick={() => setFormState({ mode: 'create' })}>Nuevo agrimensor</Button>
        )}
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>No se pudo completar la acción</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {temporaryPassword && (
        <Alert>
          <AlertTitle>Agrimensor creado</AlertTitle>
          <AlertDescription>
            <p>Contraseña temporal: {temporaryPassword}</p>
            <Button variant="outline" size="sm" onClick={() => setTemporaryPassword(null)}>
              Entendido
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {formView.mode !== 'closed' ? (
        <Card>
          <CardHeader>
            <CardTitle>
              {formView.mode === 'create' ? 'Nuevo agrimensor' : 'Editar agrimensor'}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {formView.mode === 'create' ? (
              <SurveyorForm
                key="create"
                submitLabel="Crear agrimensor"
                isEmailEditable
                isSubmitting={isSubmitting}
                onSubmit={handleCreate}
                onCancel={() => setFormState({ mode: 'closed' })}
              />
            ) : (
              <SurveyorForm
                key={formView.surveyor.id}
                initialValue={toSurveyorFormValues(formView.surveyor)}
                submitLabel="Guardar cambios"
                isEmailEditable={false}
                isSubmitting={isSubmitting}
                onSubmit={(values) => handleUpdate(formView.surveyor.id, values)}
                onCancel={() => setFormState({ mode: 'closed' })}
              />
            )}
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="flex items-center gap-2">
            <input
              id="incluir-bajas"
              type="checkbox"
              className="size-4 accent-primary"
              checked={includeInactive}
              onChange={(event) => {
                setIsLoading(true)
                setIncludeInactive(event.target.checked)
              }}
            />
            <Label htmlFor="incluir-bajas">Mostrar dados de baja</Label>
          </div>

          {isLoading && <p className="text-muted-foreground">Cargando agrimensores…</p>}

          {!isLoading && surveyors.length === 0 && (
            <p className="text-muted-foreground">No hay agrimensores cargados todavía.</p>
          )}

          {surveyors.length > 0 && (
            <ul className="flex flex-col gap-3">
              {surveyors.map((surveyor) => (
                <li key={surveyor.id}>
                  <Card>
                    <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <div className="flex flex-col items-start gap-1">
                        <p className="font-medium text-foreground">{fullName(surveyor)}</p>
                        <p className="text-sm text-muted-foreground">{surveyor.email}</p>
                        <Badge variant={isActive(surveyor) ? 'secondary' : 'destructive'}>
                          {isActive(surveyor) ? 'Activo' : 'Dado de baja'}
                        </Badge>
                      </div>

                      {isActive(surveyor) && (
                        <div className="flex flex-wrap items-center gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setFormState({ mode: 'edit', id: surveyor.id })}
                          >
                            Editar
                          </Button>
                          {confirmingBajaId === surveyor.id ? (
                            <>
                              <span className="text-sm text-muted-foreground">
                                ¿Confirmar baja?
                              </span>
                              <Button
                                variant="destructive"
                                size="sm"
                                disabled={isSubmitting}
                                onClick={() => void handleBaja(surveyor.id)}
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
                              onClick={() => setConfirmingBajaId(surveyor.id)}
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
