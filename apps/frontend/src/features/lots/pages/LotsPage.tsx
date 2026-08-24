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

type DxfState =
  | { status: 'empty' }
  | { status: 'loaded'; result: DxfParseResult; fileName: string }
  | { status: 'error'; message: string }

const emptyFields: LoteoFieldValues = {
  nombre: '',
  ubicacion: '',
  descripcion: '',
}

export default function LotsPage() {
  const [fields, setFields] = useState<LoteoFieldValues>(emptyFields)
  const [dxf, setDxf] = useState<DxfState>({ status: 'empty' })
  const [visibleLayers, setVisibleLayers] = useState<ReadonlySet<DxfLayer>>(
    () => new Set(DXF_LAYERS),
  )

  const polygons = dxf.status === 'loaded' ? dxf.result.polygons : []
  const issues = dxf.status === 'loaded' ? dxf.result.issues : []

  return (
    <section className="grid min-h-0 flex-1 grid-cols-1 gap-3 md:grid-cols-[20rem_minmax(0,1fr)] md:grid-rows-[auto_auto_1fr] md:gap-x-6 md:gap-y-3">
      <header className="md:col-span-2">
        <h1 className="text-xl font-semibold md:text-2xl">Loteos</h1>
        <p className="text-sm text-muted-foreground">
          Cargá el DXF para ver el plano. El guardado en la base llega después.
        </p>
      </header>

      <div className="flex min-h-0 flex-col gap-2 md:col-start-1 md:row-start-3">
        <DxfFileField
          fileName={dxf.status === 'loaded' ? dxf.fileName : null}
          onParsed={(result, fileName) => setDxf({ status: 'loaded', result, fileName })}
          onError={(message) => setDxf({ status: 'error', message })}
          onCleared={() => setDxf({ status: 'empty' })}
        />
        <DxfParseAlerts
          error={dxf.status === 'error' ? dxf.message : null}
          issues={issues}
        />
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
