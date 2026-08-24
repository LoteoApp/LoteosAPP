import { useEffect, useMemo, useRef, useState, type PointerEvent } from 'react'
import { Maximize2, Minus, Plus } from 'lucide-react'
import { Button } from '../../../shared/ui/button'
import {
  fitViewBox,
  panViewBox,
  viewBoxToString,
  zoomViewBox,
  type SvgViewBox,
} from '../lib/fitViewBox'
import { polygonToSvgPath } from '../lib/polygonToSvgPath'
import { DXF_LAYER_LABELS, type DxfLayer, type DxfPolygon } from '../types'

const LAYER_PAINT: Record<DxfLayer, { fill: string; stroke: string }> = {
  LOTEO: { fill: 'var(--chart-1)', stroke: 'var(--chart-1)' },
  MANZANA: { fill: 'var(--chart-2)', stroke: 'var(--chart-2)' },
  LOTES: { fill: 'var(--chart-3)', stroke: 'var(--chart-3)' },
  CALLE: { fill: 'var(--chart-4)', stroke: 'var(--chart-4)' },
}

const ZOOM_IN = 1 / 1.2
const ZOOM_OUT = 1.2

type DxfViewerProps = {
  polygons: DxfPolygon[]
  visibleLayers: ReadonlySet<DxfLayer>
}

export default function DxfViewer({ polygons, visibleLayers }: DxfViewerProps) {
  const visiblePolygons = useMemo(
    () => polygons.filter((polygon) => visibleLayers.has(polygon.layer)),
    [polygons, visibleLayers],
  )
  const fitted = useMemo(() => fitViewBox(visiblePolygons), [visiblePolygons])
  const [prevFitted, setPrevFitted] = useState(fitted)
  const [viewBox, setViewBox] = useState<SvgViewBox>(fitted)
  const svgRef = useRef<SVGSVGElement>(null)
  const dragRef = useRef<{ x: number; y: number } | null>(null)

  if (fitted !== prevFitted) {
    setPrevFitted(fitted)
    setViewBox(fitted)
  }

  useEffect(() => {
    const svg = svgRef.current
    if (!svg) {
      return
    }

    function handleWheel(event: WheelEvent) {
      event.preventDefault()
      const factor = event.deltaY > 0 ? ZOOM_OUT : ZOOM_IN
      setViewBox((current) => zoomViewBox(current, factor))
    }

    svg.addEventListener('wheel', handleWheel, { passive: false })
    return () => svg.removeEventListener('wheel', handleWheel)
  }, [])

  function handlePointerDown(event: PointerEvent<SVGSVGElement>) {
    event.currentTarget.setPointerCapture(event.pointerId)
    dragRef.current = { x: event.clientX, y: event.clientY }
  }

  function handlePointerMove(event: PointerEvent<SVGSVGElement>) {
    const drag = dragRef.current
    const svg = svgRef.current
    if (!drag || !svg) {
      return
    }

    const rect = svg.getBoundingClientRect()
    if (rect.width === 0 || rect.height === 0) {
      return
    }

    const dxPx = event.clientX - drag.x
    const dyPx = event.clientY - drag.y
    dragRef.current = { x: event.clientX, y: event.clientY }
    setViewBox((current) => panViewBox(current, dxPx, dyPx, rect.width, rect.height))
  }

  function handlePointerUp(event: PointerEvent<SVGSVGElement>) {
    dragRef.current = null
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
  }

  if (polygons.length === 0) {
    return (
      <div className="flex h-[50dvh] min-h-64 items-center justify-center rounded-xl border border-dashed border-border bg-muted/30 px-4 text-center text-sm text-muted-foreground md:h-auto md:min-h-0 md:flex-1">
        El plano aparece acá cuando cargues un DXF.
      </div>
    )
  }

  return (
    <div className="flex h-[50dvh] min-h-64 flex-col gap-2 md:h-auto md:min-h-0 md:flex-1">
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="size-11 md:size-8"
          aria-label="Acercar"
          onClick={() => setViewBox((current) => zoomViewBox(current, ZOOM_IN))}
        >
          <Plus />
        </Button>
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="size-11 md:size-8"
          aria-label="Alejar"
          onClick={() => setViewBox((current) => zoomViewBox(current, ZOOM_OUT))}
        >
          <Minus />
        </Button>
        <Button
          type="button"
          variant="outline"
          size="lg"
          className="min-h-11 px-3 md:h-8 md:min-h-8 md:px-2.5 md:text-[0.8rem]"
          aria-label="Ajustar al plano"
          onClick={() => setViewBox(fitted)}
        >
          <Maximize2 />
          Ajustar
        </Button>
      </div>

      <svg
        ref={svgRef}
        role="img"
        aria-label="Plano del loteo"
        viewBox={viewBoxToString(viewBox)}
        className="min-h-0 w-full flex-1 touch-none touch-manipulation rounded-xl border border-border bg-card"
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
      >
        {visiblePolygons.map((polygon) => {
          const paint = LAYER_PAINT[polygon.layer]
          return (
            <path
              key={polygon.id}
              d={polygonToSvgPath(polygon.vertices)}
              fill={paint.fill}
              fillOpacity={0.28}
              stroke={paint.stroke}
              strokeWidth={1.5}
              vectorEffect="non-scaling-stroke"
              aria-label={DXF_LAYER_LABELS[polygon.layer]}
            />
          )
        })}
      </svg>
    </div>
  )
}
