import { useEffect, useState } from 'react'
import { useAuth } from '../../auth/hooks/use-auth'
import { getUserRole, ROLE } from '../../auth/lib/getUserRole'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { Input } from '../../../shared/ui/input'
import { Label } from '../../../shared/ui/label'
import { createClient, deleteClient, listClients, updateClient } from '../api/clients'
import ClientForm from '../components/ClientForm'
import { resolveFormView, type FormState } from '../lib/resolveFormView'
import { toClienteFormValues, type Cliente, type ClienteFormValues } from '../types'

const DIACRITICS_PATTERN = /[\u0300-\u036f]/g

function normalize(value: string): string {
  return value.normalize('NFD').replace(DIACRITICS_PATTERN, '').toLowerCase()
}

function matchesSearch(cliente: Cliente, search: string): boolean {
  const nombreCompleto = normalize(`${cliente.nombre} ${cliente.apellido}`)
  return nombreCompleto.includes(search) || cliente.dni.toLowerCase().includes(search)
}

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export default function ClientsPage() {
  const { session, user } = useAuth()
  const token = session?.access_token ?? ''
  // Hiding the button is UX only, not access control: the backend restricts
  // the baja to administrador as well, and RLS is still pending.
  const isAdministrador = getUserRole(user) === ROLE.administrador
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [reloadCount, setReloadCount] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [formState, setFormState] = useState<FormState>({ mode: 'closed' })
  const [search, setSearch] = useState('')
  const [confirmingBajaId, setConfirmingBajaId] = useState<string | null>(null)

  const formView = resolveFormView(formState, clientes)

  const normalizedSearch = normalize(search.trim())
  const filteredClientes = normalizedSearch
    ? clientes.filter((cliente) => matchesSearch(cliente, normalizedSearch))
    : clientes

  useEffect(() => {
    let cancelled = false

    listClients(token)
      .then((loaded) => {
        if (cancelled) {
          return
        }
        setClientes(loaded)
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
  }, [token, reloadCount])

  function refresh() {
    setIsLoading(true)
    setReloadCount((count) => count + 1)
  }

  function validate(values: ClienteFormValues, excludeId?: string): string | null {
    if (!values.nombre || !values.apellido || !values.dni) {
      return 'Completá nombre, apellido y DNI.'
    }

    const dniTaken = clientes.some(
      (cliente) => cliente.dni === values.dni && cliente.id !== excludeId
    )
    if (dniTaken) {
      return 'Ya existe un cliente con ese DNI.'
    }

    return null
  }

  async function handleCreate(values: ClienteFormValues) {
    setIsSubmitting(true)
    try {
      await createClient(token, values)
      setFormState({ mode: 'closed' })
      setError(null)
      refresh()
    } catch (createError) {
      setError(messageOf(createError))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleUpdate(id: string, values: ClienteFormValues) {
    setIsSubmitting(true)
    try {
      await updateClient(token, id, values)
      setFormState({ mode: 'closed' })
      setError(null)
      refresh()
    } catch (updateError) {
      setError(messageOf(updateError))
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleBaja(cliente: Cliente) {
    setIsSubmitting(true)
    try {
      await deleteClient(token, cliente.id)
      setConfirmingBajaId(null)
      setError(null)
      refresh()
    } catch (bajaError) {
      setError(messageOf(bajaError))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Clientes</h1>
        {formView.mode === 'closed' && (
          <Button onClick={() => setFormState({ mode: 'create' })}>Nuevo cliente</Button>
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
            <CardTitle>{formView.mode === 'create' ? 'Nuevo cliente' : 'Editar cliente'}</CardTitle>
          </CardHeader>
          <CardContent>
            {formView.mode === 'create' ? (
              <ClientForm
                key="create"
                submitLabel="Crear cliente"
                onSubmit={handleCreate}
                onValidate={(values) => validate(values)}
                onCancel={() => setFormState({ mode: 'closed' })}
              />
            ) : (
              <ClientForm
                key={formView.client.id}
                initialValue={toClienteFormValues(formView.client)}
                submitLabel="Guardar cambios"
                onSubmit={(values) => handleUpdate(formView.client.id, values)}
                onValidate={(values) => validate(values, formView.client.id)}
                onCancel={() => setFormState({ mode: 'closed' })}
              />
            )}
          </CardContent>
        </Card>
      ) : (
        <>
          {isLoading && <p className="text-muted-foreground">Cargando clientes...</p>}

          {!isLoading && clientes.length === 0 && (
            <p className="text-muted-foreground">No hay clientes cargados todavía.</p>
          )}

          {clientes.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="buscar-cliente">Buscar</Label>
              <Input
                id="buscar-cliente"
                type="search"
                placeholder="Nombre, apellido o DNI"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                className="max-w-sm"
              />
            </div>
          )}

          {clientes.length > 0 && filteredClientes.length === 0 && (
            <p className="text-muted-foreground">No se encontraron clientes con esa búsqueda.</p>
          )}

          {filteredClientes.length > 0 && (
            <ul className="flex flex-col gap-3">
              {filteredClientes.map((cliente) => (
                <li key={cliente.id}>
                  <Card>
                    <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <div>
                        <p className="font-medium text-foreground">
                          {cliente.nombre} {cliente.apellido}
                        </p>
                        <p className="text-sm text-muted-foreground">DNI {cliente.dni}</p>
                        {(cliente.celular || cliente.email) && (
                          <p className="text-sm text-muted-foreground">
                            {[cliente.celular, cliente.email].filter(Boolean).join(' · ')}
                          </p>
                        )}
                      </div>
                      <div className="flex flex-wrap items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setFormState({ mode: 'edit', id: cliente.id })}
                        >
                          Editar
                        </Button>
                        {isAdministrador && confirmingBajaId === cliente.id ? (
                          <>
                            <span className="text-sm text-muted-foreground">
                              ¿Confirmar baja?
                            </span>
                            <Button
                              variant="destructive"
                              size="sm"
                              disabled={isSubmitting}
                              onClick={() => handleBaja(cliente)}
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
                          isAdministrador && (
                            <Button
                              variant="destructive"
                              size="sm"
                              onClick={() => setConfirmingBajaId(cliente.id)}
                            >
                              Dar de baja
                            </Button>
                          )
                        )}
                      </div>
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
