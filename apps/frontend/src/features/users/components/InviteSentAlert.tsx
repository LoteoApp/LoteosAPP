import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'

type InviteSentAlertProps = {
  email: string
  invitacionEnviada: boolean
  isResending: boolean
  onResend: () => void
  onClose: () => void
}

export default function InviteSentAlert({
  email,
  invitacionEnviada,
  isResending,
  onResend,
  onClose,
}: InviteSentAlertProps) {
  return (
    <Alert variant={invitacionEnviada ? 'default' : 'destructive'}>
      <AlertTitle>Usuario creado</AlertTitle>
      <AlertDescription>
        {invitacionEnviada ? (
          <p>Se envió un mail de invitación a {email} para que active su cuenta.</p>
        ) : (
          <p>No se pudo enviar el mail de invitación a {email}. Reintentá el envío.</p>
        )}
        <div className="mt-2 flex flex-wrap gap-2">
          {!invitacionEnviada && (
            <Button type="button" variant="outline" size="sm" onClick={onResend} disabled={isResending}>
              {isResending ? 'Reenviando…' : 'Reenviar invitación'}
            </Button>
          )}
          <Button type="button" variant="outline" size="sm" onClick={onClose}>
            Cerrar
          </Button>
        </div>
      </AlertDescription>
    </Alert>
  )
}
