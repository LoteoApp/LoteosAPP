import { useState } from 'react'
import { useAuth } from '../../auth/hooks/use-auth'
import { getUserRole, ROLE } from '../../../shared/auth/roles'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { Input } from '../../../shared/ui/input'
import { Label } from '../../../shared/ui/label'
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

export default function ClientsPage() {
  const { user } = useAuth()
  // Hiding the button is UX only, not access control: once the baja endpoint
  // exists it must also be enforced server-side (and via RLS) — see the API
  // integration issue.
  const isAdministrador = getUserRole(user) === ROLE.administrador
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [formState, setFormState] = useState<FormState>({ mode: 'closed' })
  const [search, setSearch] = useState('')
  const [confirmingBajaId, setConfirmingBajaId] = useState<string | null>(null)

  const formView = resolveFormView(formState, clientes)

  const normalizedSearch = normalize(search.trim())
  const filteredClientes = normalizedSearch
    ? clientes.filter((cliente) => matchesSearch(cliente, normalizedSearch))
    : clientes

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

  function handleCreate(values: ClienteFormValues) {
    setClientes((current) => [...current, { id: crypto.randomUUID(), ...values }])
    setFormState({ mode: 'closed' })
  }

  function handleUpdate(id: string, values: ClienteFormValues) {
    setClientes((current) =>
      current.map((cliente) => (cliente.id === id ? { id, ...values } : cliente))
    )
    setFormState({ mode: 'closed' })
  }

  function handleBaja(cliente: Cliente) {
    setClientes((current) => current.filter((item) => item.id !== cliente.id))
    setConfirmingBajaId(null)
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Clientes</h1>
        {formView.mode === 'closed' && (
          <Button onClick={() => setFormState({ mode: 'create' })}>Nuevo cliente</Button>
        )}
      </div>

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
          {clientes.length === 0 && (
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
