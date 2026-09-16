import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'

type CreatedCredentialsAlertProps = {
  email: string
  temporaryPassword: string
  onClose: () => void
}

export default function CreatedCredentialsAlert({
  email,
  temporaryPassword,
  onClose,
}: CreatedCredentialsAlertProps) {
  return (
    <Alert>
      <AlertTitle>Usuario creado</AlertTitle>
      <AlertDescription>
        <p>
          Contraseña temporal para {email}: <strong>{temporaryPassword}</strong>
        </p>
        <Button type="button" variant="outline" size="sm" className="mt-2" onClick={onClose}>
          Cerrar
        </Button>
      </AlertDescription>
    </Alert>
  )
}
