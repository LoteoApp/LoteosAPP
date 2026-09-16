import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'

type CreatedCredentialsAlertProps = {
  email: string
  temporaryPassword: string
  invitacionEnviada: boolean
  isResending: boolean
  onResend: () => void
  onClose: () => void
}

export default function CreatedCredentialsAlert({
  email,
  temporaryPassword,
  invitacionEnviada,
  isResending,
  onResend,
  onClose,
}: CreatedCredentialsAlertProps) {
  return (
    <Alert variant={invitacionEnviada ? 'default' : 'destructive'}>
      <AlertTitle>Usuario creado</AlertTitle>
      <AlertDescription>
        <p>
          Contraseña temporal para {email}: <strong>{temporaryPassword}</strong>
        </p>
        {!invitacionEnviada && (
          <p>No se pudo enviar el mail de invitación. Comunicá la contraseña o reenvialo.</p>
        )}
        <div className="mt-2 flex flex-wrap gap-2">
          {!invitacionEnviada && (
            <Button type="button" variant="outline" size="sm" onClick={onResend} disabled={isResending}>
              {isResending ? 'Reenviando…' : 'Reenviar credenciales'}
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
