import { useState } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../../../shared/ui/card'
import DxfFileField from '../components/DxfFileField'
import DxfLayerToggles from '../components/DxfLayerToggles'
import DxfParseAlerts from '../components/DxfParseAlerts'
import DxfViewer from '../components/DxfViewer'
import LoteoFields, { type LoteoFieldValues } from '../components/LoteoFields'
import { DXF_LAYERS, type DxfLayer, type DxfParseResult } from '../types'

const emptyFields: LoteoFieldValues = {
  nombre: '',
  ubicacion: '',
  descripcion: '',
}

export default function LotsPage() {
  const [fields, setFields] = useState<LoteoFieldValues>(emptyFields)
  const [parseResult, setParseResult] = useState<DxfParseResult | null>(null)
  const [parseError, setParseError] = useState<string | null>(null)
  const [fileName, setFileName] = useState<string | null>(null)
  const [visibleLayers, setVisibleLayers] = useState<ReadonlySet<DxfLayer>>(
    () => new Set(DXF_LAYERS),
  )

  const polygons = parseResult?.polygons ?? []
  const issues = parseResult?.issues ?? []

  const dxfField = (
    <DxfFileField
      fileName={fileName}
      onParsed={(result, name) => {
        setParseResult(result)
        setParseError(null)
        setFileName(name)
      }}
      onError={(message) => {
        setParseResult(null)
        setParseError(message)
        setFileName(null)
      }}
      onCleared={() => {
        setParseResult(null)
        setParseError(null)
        setFileName(null)
      }}
    />
  )

  return (
    <section className="grid min-h-0 flex-1 grid-cols-1 gap-3 md:grid-cols-[20rem_minmax(0,1fr)] md:grid-rows-[auto_auto_1fr] md:gap-x-6 md:gap-y-3">
      <header className="md:col-span-2">
        <h1 className="text-xl font-semibold md:text-2xl">Loteos</h1>
        <p className="text-sm text-muted-foreground">
          Cargá el DXF para ver el plano. El guardado en la base llega después.
        </p>
      </header>

      <div className="flex min-h-0 flex-col gap-2 md:col-start-1 md:row-start-3">
        {dxfField}
        <DxfParseAlerts error={parseError} issues={issues} />
      </div>

      <div className="flex min-h-0 flex-col gap-2 md:col-start-2 md:row-start-2 md:row-span-2">
        {polygons.length > 0 ? (
          <DxfLayerToggles
            visibleLayers={visibleLayers}
            onVisibleLayersChange={setVisibleLayers}
          />
        ) : null}
        <DxfViewer polygons={polygons} visibleLayers={visibleLayers} />
      </div>

      <Card size="sm" className="md:col-start-1 md:row-start-2 md:self-start">
        <CardHeader>
          <CardTitle>Datos del loteo</CardTitle>
          <CardDescription>
            Nombre, ubicación y descripción breve.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <LoteoFields values={fields} onChange={setFields} />
        </CardContent>
      </Card>
    </section>
  )
}
