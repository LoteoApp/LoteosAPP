import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import type { DxfValidationIssue } from '../types'

const MAX_VISIBLE_ISSUES = 20

type DxfParseAlertsProps = {
  error: string | null
  issues: DxfValidationIssue[]
}

export default function DxfParseAlerts({ error, issues }: DxfParseAlertsProps) {
  if (!error && issues.length === 0) {
    return null
  }

  const visibleIssues = issues.slice(0, MAX_VISIBLE_ISSUES)
  const hiddenCount = issues.length - visibleIssues.length

  return (
    <div className="flex flex-col gap-2">
      {error ? (
        <Alert variant="destructive">
          <AlertTitle>No se pudo leer el DXF</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {visibleIssues.length > 0 ? (
        <Alert>
          <AlertTitle>Avisos de geometría</AlertTitle>
          <AlertDescription>
            <ul className="list-disc pl-4">
              {visibleIssues.map((issue, index) => (
                <li key={`${issue.code}-${issue.polygonId ?? index}`}>
                  {issue.message}
                </li>
              ))}
            </ul>
            {hiddenCount > 0 ? <p>Y {hiddenCount} avisos más.</p> : null}
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  )
}
