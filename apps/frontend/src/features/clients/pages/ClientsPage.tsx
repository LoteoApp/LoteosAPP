import { useState } from 'react'
import { useAuth } from '../../auth/hooks/use-auth'
import { getUserRole } from '../../auth/lib/getUserRole'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import ClientForm from '../components/ClientForm'
import { toClienteFormValues, type Cliente, type ClienteFormValues } from '../types'

type FormState = { mode: 'closed' } | { mode: 'create' } | { mode: 'edit'; id: string }

const searchFieldClassName =
  'h-10 w-full max-w-sm rounded-lg border border-border bg-background px-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50'

const DIACRITICS_PATTERN = /[̀-ͯ]/g

function normalize(value: string): string {
  return value.normalize('NFD').replace(DIACRITICS_PATTERN, '').toLowerCase()
}

function matchesSearch(cliente: Cliente, search: string): boolean {
  const nombreCompleto = normalize(`${cliente.nombre} ${cliente.apellido}`)
  return nombreCompleto.includes(search) || cliente.dni.toLowerCase().includes(search)
}

export default function ClientsPage() {
  const { user } = useAuth()
  const isAdministrador = getUserRole(user) === 'administrador'
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [formState, setFormState] = useState<FormState>({ mode: 'closed' })
  const [search, setSearch] = useState('')

  const editingClient =
    formState.mode === 'edit'
      ? clientes.find((cliente) => cliente.id === formState.id)
      : undefined

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
    const confirmed = window.confirm(`¿Dar de baja a ${cliente.nombre} ${cliente.apellido}?`)
    if (!confirmed) return

    setClientes((current) => current.filter((item) => item.id !== cliente.id))
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Clientes</h1>
        {formState.mode === 'closed' && (
          <Button onClick={() => setFormState({ mode: 'create' })}>Nuevo cliente</Button>
        )}
      </div>

      {formState.mode === 'create' && (
        <Card>
          <CardHeader>
            <CardTitle>Nuevo cliente</CardTitle>
          </CardHeader>
          <CardContent>
            <ClientForm
              key="create"
              submitLabel="Crear cliente"
              onSubmit={handleCreate}
              onValidate={(values) => validate(values)}
              onCancel={() => setFormState({ mode: 'closed' })}
            />
          </CardContent>
        </Card>
      )}

      {formState.mode === 'edit' && editingClient && (
        <Card>
          <CardHeader>
            <CardTitle>Editar cliente</CardTitle>
          </CardHeader>
          <CardContent>
            <ClientForm
              key={editingClient.id}
              initialValue={toClienteFormValues(editingClient)}
              submitLabel="Guardar cambios"
              onSubmit={(values) => handleUpdate(editingClient.id, values)}
              onValidate={(values) => validate(values, editingClient.id)}
              onCancel={() => setFormState({ mode: 'closed' })}
            />
          </CardContent>
        </Card>
      )}

      {formState.mode === 'closed' && clientes.length === 0 && (
        <p className="text-muted-foreground">No hay clientes cargados todavía.</p>
      )}

      {formState.mode === 'closed' && clientes.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <label htmlFor="buscar-cliente" className="text-sm font-medium">
            Buscar
          </label>
          <input
            id="buscar-cliente"
            type="search"
            placeholder="Nombre, apellido o DNI"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className={searchFieldClassName}
          />
        </div>
      )}

      {formState.mode === 'closed' && clientes.length > 0 && filteredClientes.length === 0 && (
        <p className="text-muted-foreground">No se encontraron clientes con esa búsqueda.</p>
      )}

      {formState.mode === 'closed' && filteredClientes.length > 0 && (
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
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setFormState({ mode: 'edit', id: cliente.id })}
                    >
                      Editar
                    </Button>
                    {isAdministrador && (
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => handleBaja(cliente)}
                      >
                        Dar de baja
                      </Button>
                    )}
                  </div>
                </CardContent>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
