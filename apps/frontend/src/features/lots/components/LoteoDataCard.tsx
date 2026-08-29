import { cn } from '../../../shared/lib/utils'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../../../shared/ui/card'
import LoteoFields, { type LoteoFieldValues } from './LoteoFields'

type LoteoDataCardProps = {
  values: LoteoFieldValues
  onChange: (values: LoteoFieldValues) => void
  className?: string
}

export default function LoteoDataCard({ values, onChange, className }: LoteoDataCardProps) {
  return (
    <Card size="sm" className={cn(className)}>
      <CardHeader>
        <CardTitle>Datos del loteo</CardTitle>
        <CardDescription>Nombre, ubicación, inmobiliarias y descripción breve.</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <LoteoFields values={values} onChange={onChange} />
      </CardContent>
    </Card>
  )
}
