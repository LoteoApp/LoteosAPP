import { useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../../../shared/ui/card'
import CreatedCredentialsAlert from '../components/CreatedCredentialsAlert'
import UserCard from '../components/UserCard'
import UserForm from '../components/UserForm'
import UsersFilters, { type EstadoFilter, type RolFilter } from '../components/UsersFilters'
import { useUsers } from '../hooks/use-users'
import { resolveFormView, type FormState } from '../lib/resolveFormView'
import { isActivo, toUsuarioUpdateValues, type Usuario, type UsuarioFormValues } from '../types'

function matchesRol(usuario: Usuario, filter: RolFilter): boolean {
  return filter === 'todos' || usuario.rol === filter
}

function matchesEstado(usuario: Usuario, filter: EstadoFilter): boolean {
  if (filter === 'todos') {
    return true
  }
  return filter === 'activos' ? isActivo(usuario) : !isActivo(usuario)
}

type UsersPageProps = {
  accessToken: string | null
}

// This route is admin-only, gated by RequireRole — every render here is
// for an administrador, so no further role check is needed on top of it.
export default function UsersPage({ accessToken }: UsersPageProps) {
  const token = accessToken ?? ''
  const { usuarios, isLoading, isSubmitting, error, clearError, create, update, deactivate, reactivate } =
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
              clearError()
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
        <CreatedCredentialsAlert
          email={createdCredentials.email}
          temporaryPassword={createdCredentials.temporaryPassword}
          onClose={() => setCreatedCredentials(null)}
        />
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
                onCancel={() => {
                  clearError()
                  setFormState({ mode: 'closed' })
                }}
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
                onCancel={() => {
                  clearError()
                  setFormState({ mode: 'closed' })
                }}
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
            <UsersFilters
              rolFilter={rolFilter}
              estadoFilter={estadoFilter}
              onRolFilterChange={setRolFilter}
              onEstadoFilterChange={setEstadoFilter}
            />
          )}

          {usuarios.length > 0 && filteredUsuarios.length === 0 && (
            <p className="text-muted-foreground">No se encontraron usuarios con esos filtros.</p>
          )}

          {filteredUsuarios.length > 0 && (
            <ul className="flex flex-col gap-3">
              {filteredUsuarios.map((usuario) => (
                <UserCard
                  key={usuario.id}
                  usuario={usuario}
                  isSubmitting={isSubmitting}
                  isConfirmingBaja={confirmingBajaId === usuario.id}
                  onEdit={() => {
                    setCreatedCredentials(null)
                    clearError()
                    setFormState({ mode: 'edit', id: usuario.id })
                  }}
                  onStartConfirmBaja={() => setConfirmingBajaId(usuario.id)}
                  onCancelConfirmBaja={() => {
                    clearError()
                    setConfirmingBajaId(null)
                  }}
                  onConfirmBaja={() => handleBaja(usuario)}
                  onReactivar={() => handleReactivar(usuario)}
                />
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  )
}
