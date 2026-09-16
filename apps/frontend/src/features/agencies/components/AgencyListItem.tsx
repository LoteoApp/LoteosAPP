import { useState } from 'react'
import { Button } from '../../../shared/ui/button'
import { Card, CardContent } from '../../../shared/ui/card'
import type { Agency } from '../types'

type AgencyListItemProps = {
  agency: Agency
  canManage: boolean
  isSubmitting: boolean
  onEdit: () => void
  onDeactivate: () => Promise<boolean>
}

export default function AgencyListItem({
  agency,
  canManage,
  isSubmitting,
  onEdit,
  onDeactivate,
}: AgencyListItemProps) {
  const [isConfirming, setIsConfirming] = useState(false)
  const contact = [agency.telefono, agency.email].filter(Boolean).join(' · ')

  async function handleDeactivate() {
    if (await onDeactivate()) {
      setIsConfirming(false)
    }
  }

  return (
    <li>
      <Card>
        <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="font-medium text-foreground">{agency.razonSocial}</p>
            {agency.cuit && (
              <p className="text-sm text-muted-foreground">CUIT {agency.cuit}</p>
            )}
            {contact && <p className="text-sm text-muted-foreground">{contact}</p>}
          </div>
          {canManage && (
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="outline" size="sm" onClick={onEdit}>
                Editar
              </Button>
              {isConfirming ? (
                <>
                  <span className="text-sm text-muted-foreground">¿Confirmar baja?</span>
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={isSubmitting}
                    onClick={handleDeactivate}
                  >
                    Confirmar
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => setIsConfirming(false)}>
                    Cancelar
                  </Button>
                </>
              ) : (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setIsConfirming(true)}
                >
                  Dar de baja
                </Button>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </li>
  )
}
