import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import type { DxfValidationIssue } from '../types'

type DxfParseAlertsProps = {
  error: string | null
  issues: DxfValidationIssue[]
}

export default function DxfParseAlerts({ error, issues }: DxfParseAlertsProps) {
  return (
    <div className="flex flex-col gap-2">
      {error ? (
        <Alert variant="destructive">
          <AlertTitle>No se pudo leer el DXF</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {issues.length > 0 ? (
        <Alert>
          <AlertTitle>Avisos de geometría</AlertTitle>
          <AlertDescription>
            <ul className="list-disc pl-4">
              {issues.map((issue, index) => (
                <li key={`${issue.code}-${issue.polygonId ?? index}`}>
                  {issue.message}
                </li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  )
}
