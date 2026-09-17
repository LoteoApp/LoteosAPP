import { CircleCheck, CircleX } from 'lucide-react'
import { Badge } from '../../../shared/ui/badge'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent } from '../../../shared/ui/card'
import { ROLE_LABELS, isActivo, type Usuario } from '../types'

export type ResendStatus = 'success' | 'error' | undefined

// Success has no dedicated Button variant (it's the only place that needs
// one), so it's a border/text color override on top of "outline" instead of
// a new shared variant.
const RESEND_SUCCESS_CLASS = 'border-green-600 text-green-700 dark:border-green-500 dark:text-green-400'

type UserCardProps = {
  usuario: Usuario
  isSubmitting: boolean
  isConfirmingBaja: boolean
  resendStatus: ResendStatus
  onEdit: () => void
  onStartConfirmBaja: () => void
  onCancelConfirmBaja: () => void
  onConfirmBaja: () => void
  onReactivar: () => void
  onResendInvite: () => void
}

export default function UserCard({
  usuario,
  isSubmitting,
  isConfirmingBaja,
  resendStatus,
  onEdit,
  onStartConfirmBaja,
  onCancelConfirmBaja,
  onConfirmBaja,
  onReactivar,
  onResendInvite,
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
              {!isConfirmingBaja && (
                <Button
                  variant={resendStatus === 'error' ? 'destructive' : 'outline'}
                  size="sm"
                  disabled={isSubmitting}
                  onClick={onResendInvite}
                  className={resendStatus === 'success' ? RESEND_SUCCESS_CLASS : undefined}
                >
                  {resendStatus === 'success' && <CircleCheck aria-hidden />}
                  {resendStatus === 'error' && <CircleX aria-hidden />}
                  Reenviar invitación
                </Button>
              )}
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
