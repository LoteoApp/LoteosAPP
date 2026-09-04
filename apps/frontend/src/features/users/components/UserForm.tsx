import { useState, type FormEvent } from 'react'
import { Button } from '../../../shared/ui/button'
import { Input } from '../../../shared/ui/input'
import { Label } from '../../../shared/ui/label'
import { Select, SelectContent, SelectItem, SelectList, SelectTrigger, SelectValue } from '../../../shared/ui/select'
import { GESTIONABLE_ROLES, ROLE_LABELS } from '../types'
import type { GestionableRol, UsuarioFormValues, UsuarioUpdateValues } from '../types'

type CreateUserFormProps = {
  mode: 'create'
  submitLabel: string
  isSubmitting?: boolean
  onSubmit: (values: UsuarioFormValues) => void
  onValidate: (values: UsuarioFormValues) => string | null
  onCancel: () => void
}

type EditUserFormProps = {
  mode: 'edit'
  email: string
  rol: GestionableRol
  initialValue: UsuarioUpdateValues
  submitLabel: string
  isSubmitting?: boolean
  onSubmit: (values: UsuarioUpdateValues) => void
  onValidate: (values: UsuarioUpdateValues) => string | null
  onCancel: () => void
}

type UserFormProps = CreateUserFormProps | EditUserFormProps

export default function UserForm(props: UserFormProps) {
  const { submitLabel, isSubmitting = false, onCancel } = props
  const [nombre, setNombre] = useState(props.mode === 'edit' ? props.initialValue.nombre : '')
  const [apellido, setApellido] = useState(props.mode === 'edit' ? props.initialValue.apellido : '')
  const [email, setEmail] = useState('')
  const [rol, setRol] = useState<GestionableRol>(GESTIONABLE_ROLES[0])
  const [error, setError] = useState<string | null>(null)

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    if (isSubmitting) {
      return
    }

    const trimmedNombre = nombre.trim()
    const trimmedApellido = apellido.trim()

    if (props.mode === 'create') {
      const values: UsuarioFormValues = {
        nombre: trimmedNombre,
        apellido: trimmedApellido,
        email: email.trim(),
        rol,
      }
      const validationError = props.onValidate(values)
      if (validationError) {
        setError(validationError)
        return
      }
      setError(null)
      props.onSubmit(values)
      return
    }

    const values: UsuarioUpdateValues = { nombre: trimmedNombre, apellido: trimmedApellido }
    const validationError = props.onValidate(values)
    if (validationError) {
      setError(validationError)
      return
    }
    setError(null)
    props.onSubmit(values)
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={handleSubmit} aria-label="Datos del usuario">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="nombre">Nombre</Label>
          <Input id="nombre" name="nombre" value={nombre} onChange={(e) => setNombre(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="apellido">Apellido</Label>
          <Input
            id="apellido"
            name="apellido"
            value={apellido}
            onChange={(e) => setApellido(e.target.value)}
          />
        </div>

        {props.mode === 'create' ? (
          <>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="email">Correo electrónico</Label>
              <Input
                id="email"
                name="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="rol">Rol</Label>
              <Select
                name="rol"
                value={rol}
                onValueChange={(value) => setRol(value as GestionableRol)}
              >
                <SelectTrigger id="rol">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectList>
                    {GESTIONABLE_ROLES.map((candidate) => (
                      <SelectItem key={candidate} value={candidate}>
                        {ROLE_LABELS[candidate]}
                      </SelectItem>
                    ))}
                  </SelectList>
                </SelectContent>
              </Select>
            </div>
          </>
        ) : (
          <div className="flex flex-col justify-end gap-1.5 sm:col-span-2">
            <p className="text-sm text-muted-foreground">
              {props.email} · {ROLE_LABELS[props.rol]}
            </p>
          </div>
        )}
      </div>

      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}

      <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
        <Button type="button" variant="outline" disabled={isSubmitting} onClick={onCancel}>
          Cancelar
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Guardando...' : submitLabel}
        </Button>
      </div>
    </form>
  )
}
