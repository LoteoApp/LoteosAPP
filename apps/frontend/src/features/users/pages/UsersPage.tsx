import { useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Badge } from '../../../shared/ui/badge'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import { Label } from '../../../shared/ui/label'
import { useAuth } from '../../auth/hooks/use-auth'
import UserForm from '../components/UserForm'
import { useUsers } from '../hooks/use-users'
import { resolveFormView, type FormState } from '../lib/resolveFormView'
import {
  GESTIONABLE_ROLES,
  ROLE_LABELS,
  isActivo,
  toUsuarioUpdateValues,
  type GestionableRol,
  type Usuario,
  type UsuarioFormValues,
} from '../types'

type RolFilter = 'todos' | GestionableRol
type EstadoFilter = 'todos' | 'activos' | 'inactivos'

function matchesRol(usuario: Usuario, filter: RolFilter): boolean {
  return filter === 'todos' || usuario.rol === filter
}

function matchesEstado(usuario: Usuario, filter: EstadoFilter): boolean {
  if (filter === 'todos') {
    return true
  }
  return filter === 'activos' ? isActivo(usuario) : !isActivo(usuario)
}

// This route is admin-only, gated by RequireRole — every render here is
// for an administrador, so no further role check is needed on top of it.
export default function UsersPage() {
  const { session } = useAuth()
  const token = session?.access_token ?? ''
  const { usuarios, isLoading, isSubmitting, error, create, update, deactivate, reactivate } =
    useUsers(token)
  const [formState, setFormState] = useState<FormState>({ mode: 'closed' })
  const [rolFilter, setRolFilter] = useState<RolFilter>('todos')
  const [estadoFilter, setEstadoFilter] = useState<EstadoFilter>('todos')
  const [confirmingBajaId, setConfirmingBajaId] = useState<string | null>(null)
  const [createdCredentials, setCreatedCredentials] = useState<{
    email: string
    temporaryPassword: string
  } | null>(null)

  const formView = resolveFormView(formState, usuarios)

  const filteredUsuarios = usuarios.filter(
    (usuario) => matchesRol(usuario, rolFilter) && matchesEstado(usuario, estadoFilter),
  )

  function validateCreate(values: UsuarioFormValues): string | null {
    if (!values.nombre || !values.apellido || !values.email) {
      return 'Completá nombre, apellido y correo electrónico.'
    }
    if (usuarios.some((usuario) => usuario.email === values.email)) {
      return 'Ya existe un usuario con ese correo electrónico.'
    }
    return null
  }

  function validateEdit(values: { nombre: string; apellido: string }): string | null {
    if (!values.nombre || !values.apellido) {
      return 'Completá nombre y apellido.'
    }
    return null
  }

  async function handleCreate(values: UsuarioFormValues) {
    const temporaryPassword = await create(values)
    if (temporaryPassword) {
      setCreatedCredentials({ email: values.email, temporaryPassword })
      setFormState({ mode: 'closed' })
    }
  }

  async function handleUpdate(id: string, values: { nombre: string; apellido: string }) {
    if (await update(id, values)) {
      setFormState({ mode: 'closed' })
    }
  }

  async function handleBaja(usuario: Usuario) {
    if (await deactivate(usuario.id)) {
      setConfirmingBajaId(null)
    }
  }

  async function handleReactivar(usuario: Usuario) {
    await reactivate(usuario.id)
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Usuarios</h1>
        {formView.mode === 'closed' && (
          <Button
            onClick={() => {
              setCreatedCredentials(null)
              setFormState({ mode: 'create' })
            }}
          >
            Nuevo usuario
          </Button>
        )}
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertTitle>No se pudo completar la operación</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {createdCredentials && formView.mode === 'closed' && (
        <Alert>
          <AlertTitle>Usuario creado</AlertTitle>
          <AlertDescription>
            <p>
              Contraseña temporal para {createdCredentials.email}:{' '}
              <strong>{createdCredentials.temporaryPassword}</strong>
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="mt-2"
              onClick={() => setCreatedCredentials(null)}
            >
              Cerrar
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {formView.mode !== 'closed' ? (
        <Card>
          <CardHeader>
            <CardTitle>{formView.mode === 'create' ? 'Nuevo usuario' : 'Editar usuario'}</CardTitle>
          </CardHeader>
          <CardContent>
            {formView.mode === 'create' ? (
              <UserForm
                key="create"
                mode="create"
                submitLabel="Crear usuario"
                isSubmitting={isSubmitting}
                onSubmit={handleCreate}
                onValidate={validateCreate}
                onCancel={() => setFormState({ mode: 'closed' })}
              />
            ) : (
              <UserForm
                key={formView.usuario.id}
                mode="edit"
                email={formView.usuario.email}
                rol={formView.usuario.rol}
                initialValue={toUsuarioUpdateValues(formView.usuario)}
                submitLabel="Guardar cambios"
                isSubmitting={isSubmitting}
                onSubmit={(values) => handleUpdate(formView.usuario.id, values)}
                onValidate={validateEdit}
                onCancel={() => setFormState({ mode: 'closed' })}
              />
            )}
          </CardContent>
        </Card>
      ) : (
        <>
          {isLoading && <p className="text-muted-foreground">Cargando usuarios...</p>}

          {!isLoading && !error && usuarios.length === 0 && (
            <p className="text-muted-foreground">No hay usuarios cargados todavía.</p>
          )}

          {usuarios.length > 0 && (
            <div className="flex flex-col gap-3 sm:flex-row">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="filtro-rol">Rol</Label>
                <select
                  id="filtro-rol"
                  value={rolFilter}
                  onChange={(event) => setRolFilter(event.target.value as RolFilter)}
                  className="h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 sm:w-48 md:text-sm dark:bg-input/30"
                >
                  <option value="todos">Todos</option>
                  {GESTIONABLE_ROLES.map((candidate) => (
                    <option key={candidate} value={candidate}>
                      {ROLE_LABELS[candidate]}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="filtro-estado">Estado</Label>
                <select
                  id="filtro-estado"
                  value={estadoFilter}
                  onChange={(event) => setEstadoFilter(event.target.value as EstadoFilter)}
                  className="h-9 w-full rounded-lg border border-input bg-transparent px-3 py-1 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 sm:w-48 md:text-sm dark:bg-input/30"
                >
                  <option value="todos">Todos</option>
                  <option value="activos">Activos</option>
                  <option value="inactivos">Dados de baja</option>
                </select>
              </div>
            </div>
          )}

          {usuarios.length > 0 && filteredUsuarios.length === 0 && (
            <p className="text-muted-foreground">No se encontraron usuarios con esos filtros.</p>
          )}

          {filteredUsuarios.length > 0 && (
            <ul className="flex flex-col gap-3">
              {filteredUsuarios.map((usuario) => {
                const activo = isActivo(usuario)
                return (
                  <li key={usuario.id}>
                    <Card>
                      <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <div>
                          <p className="font-medium text-foreground">
                            {usuario.nombre} {usuario.apellido}
                          </p>
                          <p className="text-sm text-muted-foreground">{usuario.email}</p>
                          <div className="mt-1 flex gap-1.5">
                            <Badge variant="outline">{ROLE_LABELS[usuario.rol]}</Badge>
                            <Badge variant={activo ? 'default' : 'secondary'}>
                              {activo ? 'Activo' : 'Dado de baja'}
                            </Badge>
                          </div>
                        </div>
                        {activo && (
                          <div className="flex flex-wrap items-center gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => {
                                setCreatedCredentials(null)
                                setFormState({ mode: 'edit', id: usuario.id })
                              }}
                            >
                              Editar
                            </Button>
                            {confirmingBajaId === usuario.id ? (
                              <>
                                <span className="text-sm text-muted-foreground">¿Confirmar baja?</span>
                                <Button
                                  variant="destructive"
                                  size="sm"
                                  disabled={isSubmitting}
                                  onClick={() => handleBaja(usuario)}
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
                                onClick={() => setConfirmingBajaId(usuario.id)}
                              >
                                Dar de baja
                              </Button>
                            )}
                          </div>
                        )}
                        {!activo && (
                          <div className="flex flex-wrap items-center gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={isSubmitting}
                              onClick={() => handleReactivar(usuario)}
                            >
                              Reactivar
                            </Button>
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  </li>
                )
              })}
            </ul>
          )}
        </>
      )}
    </section>
  )
}
