import { Badge } from '../../../shared/ui/badge'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent } from '../../../shared/ui/card'
import { ROLE_LABELS, isActivo, type Usuario } from '../types'

type UserCardProps = {
  usuario: Usuario
  isSubmitting: boolean
  isConfirmingBaja: boolean
  onEdit: () => void
  onStartConfirmBaja: () => void
  onCancelConfirmBaja: () => void
  onConfirmBaja: () => void
  onReactivar: () => void
}

export default function UserCard({
  usuario,
  isSubmitting,
  isConfirmingBaja,
  onEdit,
  onStartConfirmBaja,
  onCancelConfirmBaja,
  onConfirmBaja,
  onReactivar,
}: UserCardProps) {
  const activo = isActivo(usuario)

  return (
    <li>
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
          {activo ? (
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="outline" size="sm" onClick={onEdit}>
                Editar
              </Button>
              {isConfirmingBaja ? (
                <>
                  <span className="text-sm text-muted-foreground">¿Confirmar baja?</span>
                  <Button variant="destructive" size="sm" disabled={isSubmitting} onClick={onConfirmBaja}>
                    Confirmar
                  </Button>
                  <Button variant="outline" size="sm" onClick={onCancelConfirmBaja}>
                    Cancelar
                  </Button>
                </>
              ) : (
                <Button variant="destructive" size="sm" onClick={onStartConfirmBaja}>
                  Dar de baja
                </Button>
              )}
            </div>
          ) : (
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="outline" size="sm" disabled={isSubmitting} onClick={onReactivar}>
                Reactivar
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </li>
  )
}
