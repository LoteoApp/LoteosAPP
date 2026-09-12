import {
  memo,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent,
} from 'react'
import { Maximize2, Minus, Plus } from 'lucide-react'
import { Button } from '../../../shared/ui/button'
import { DXF_LAYER_LABELS } from '../lib/dxfLayerLabels'
import {
  fitViewBox,
  panViewBox,
  viewBoxToString,
  zoomViewBox,
  type SvgViewBox,
  type ZoomFocus,
} from '../lib/fitViewBox'
import { calleCaptionLayout } from '../lib/calleCaptionLayout'
import { loteCaptionLayout } from '../lib/loteCaptionLayout'
import { polygonToSvgPath } from '../lib/polygonToSvgPath'
import { LOT_STATE_LABELS, LOT_STATE_PAINT } from '../lib/lotStateVisuals'
import type { DxfLayer, DxfPolygon, LotState } from '../types'

const LAYER_PAINT: Record<DxfLayer, { fill: string; stroke: string }> = {
  LOTEO: { fill: 'none', stroke: 'var(--chart-1)' },
  MANZANA: { fill: 'var(--plan-block)', stroke: 'var(--plan-block-line)' },
  LOTES: { fill: 'var(--plan-block)', stroke: 'var(--plan-block-line)' },
  CALLE: { fill: 'var(--plan-street)', stroke: 'var(--plan-street-line)' },
}

const LAYER_FILL_OPACITY: Record<DxfLayer, number> = {
  LOTEO: 0,
  MANZANA: 1,
  LOTES: 1,
  CALLE: 1,
}

const LAYER_STROKE_WIDTH: Record<DxfLayer, number> = {
  LOTEO: 2.5,
  MANZANA: 1,
  LOTES: 1,
  CALLE: 0.6,
}

const ZOOM_IN = 1 / 1.2
const ZOOM_OUT = 1.2
const DRAG_THRESHOLD_PX = 8
const SELECTED_STROKE_WIDTH = 2.5
const LOT_STATE_FILL_OPACITY = 0.9
const SELECTED_FILL_OPACITY = 1
const LOT_STATE_STROKE_WIDTH = 1

type PointerPosition = { x: number; y: number }

function focusWithin(
  svg: SVGSVGElement,
  clientX: number,
  clientY: number,
): ZoomFocus | undefined {
  const rect = svg.getBoundingClientRect()
  if (rect.width === 0 || rect.height === 0) {
    return undefined
  }

  return {
    x: (clientX - rect.left) / rect.width,
    y: (clientY - rect.top) / rect.height,
  }
}

function distanceBetween(a: PointerPosition, b: PointerPosition): number {
  return Math.hypot(a.x - b.x, a.y - b.y)
}

function midpointOf(a: PointerPosition, b: PointerPosition): PointerPosition {
  return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 }
}

const DEFAULT_EMPTY_MESSAGE = 'El plano aparece acá cuando cargues un DXF.'

type DxfViewerProps = {
  polygons: DxfPolygon[]
  visibleLayers: ReadonlySet<DxfLayer>
  emptyMessage?: string
  selectedPolygonId?: string | null
  onSelectPolygon?: (polygonId: string | null) => void
  polygonLabels?: ReadonlyMap<string, string>
}

export default function DxfViewer({
  polygons,
  visibleLayers,
  emptyMessage = DEFAULT_EMPTY_MESSAGE,
  selectedPolygonId = null,
  onSelectPolygon,
  polygonLabels,
}: DxfViewerProps) {
  const visiblePolygons = useMemo(
    () => polygons.filter((polygon) => visibleLayers.has(polygon.layer)),
    [polygons, visibleLayers],
  )
  const polygonPaths = useMemo(
    () => new Map(polygons.map((polygon) => [polygon.id, polygonToSvgPath(polygon.vertices)])),
    [polygons],
  )
  const fitted = useMemo(() => fitViewBox(polygons), [polygons])
  const [prevFitted, setPrevFitted] = useState(fitted)
  const [viewBox, setViewBox] = useState<SvgViewBox>(fitted)
  const svgRef = useRef<SVGSVGElement>(null)
  const dragRef = useRef<PointerPosition | null>(null)
  const originRef = useRef<PointerPosition | null>(null)
  const movedRef = useRef(false)
  const pointersRef = useRef(new Map<number, PointerPosition>())
  const pinchDistanceRef = useRef<number | null>(null)
  const interactive = onSelectPolygon !== undefined

  if (fitted !== prevFitted) {
    setPrevFitted(fitted)
    setViewBox(fitted)
  }

  const hasGeometry = polygons.length > 0

  useEffect(() => {
    const svg = svgRef.current
    if (!svg) {
      return
    }

    const handleWheel = (event: WheelEvent) => {
      event.preventDefault()
      const factor = event.deltaY > 0 ? ZOOM_OUT : ZOOM_IN
      const focus = focusWithin(svg, event.clientX, event.clientY)
      setViewBox((current) => zoomViewBox(current, factor, fitted, focus))
    }

    svg.addEventListener('wheel', handleWheel, { passive: false })
    return () => svg.removeEventListener('wheel', handleWheel)
  }, [hasGeometry, fitted])

  function handlePointerDown(event: PointerEvent<SVGSVGElement>) {
    pointersRef.current.set(event.pointerId, { x: event.clientX, y: event.clientY })

    if (pointersRef.current.size > 1) {
      dragRef.current = null
      pinchDistanceRef.current = null
      movedRef.current = true
      return
    }

    movedRef.current = false
    originRef.current = { x: event.clientX, y: event.clientY }
    dragRef.current = { x: event.clientX, y: event.clientY }
  }

  function handlePointerMove(event: PointerEvent<SVGSVGElement>) {
    const svg = svgRef.current
    if (!svg) {
      return
    }

    const pointers = pointersRef.current
    if (pointers.has(event.pointerId)) {
      pointers.set(event.pointerId, { x: event.clientX, y: event.clientY })
    }

    if (pointers.size > 1) {
      movedRef.current = true
      pinch(svg, [...pointers.values()])
      return
    }

    const drag = dragRef.current
    if (!drag) {
      return
    }

    const rect = svg.getBoundingClientRect()
    if (rect.width === 0 || rect.height === 0) {
      return
    }

    const origin = originRef.current
    if (origin && !movedRef.current) {
      const distance = distanceBetween(origin, { x: event.clientX, y: event.clientY })
      if (distance > DRAG_THRESHOLD_PX) {
        movedRef.current = true
        event.currentTarget.setPointerCapture(event.pointerId)
      }
    }

    const dxPx = event.clientX - drag.x
    const dyPx = event.clientY - drag.y
    dragRef.current = { x: event.clientX, y: event.clientY }
    setViewBox((current) => panViewBox(current, dxPx, dyPx, rect.width, rect.height))
  }

  function pinch(svg: SVGSVGElement, [first, second]: PointerPosition[]) {
    const distance = distanceBetween(first, second)
    if (distance === 0) {
      return
    }

    const previous = pinchDistanceRef.current
    pinchDistanceRef.current = distance
    if (previous === null || previous === 0) {
      return
    }

    const midpoint = midpointOf(first, second)
    const focus = focusWithin(svg, midpoint.x, midpoint.y)
    setViewBox((current) => zoomViewBox(current, previous / distance, fitted, focus))
  }

  function handlePointerUp(event: PointerEvent<SVGSVGElement>) {
    const pointers = pointersRef.current
    pointers.delete(event.pointerId)
    pinchDistanceRef.current = null

    const [remaining] = pointers.values()
    dragRef.current = remaining ? { ...remaining } : null

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
  }

  function handleBackgroundClick(event: MouseEvent<SVGSVGElement>) {
    if (!onSelectPolygon || movedRef.current) {
      return
    }
    if (event.target !== event.currentTarget) {
      return
    }
    onSelectPolygon(null)
  }

  function handlePathClick(event: MouseEvent<SVGPathElement>, polygonId: string) {
    event.stopPropagation()
    if (!onSelectPolygon || movedRef.current) {
      return
    }
    onSelectPolygon(polygonId)
  }

  function handlePathKeyDown(event: KeyboardEvent<SVGPathElement>, polygonId: string) {
    if (!onSelectPolygon) {
      return
    }
    if (event.key !== 'Enter' && event.key !== ' ') {
      return
    }
    event.preventDefault()
    onSelectPolygon(polygonId)
  }

  if (!hasGeometry) {
    return (
      <div className="flex h-[50dvh] min-h-64 items-center justify-center rounded-xl border border-dashed border-border bg-muted/30 px-4 text-center text-sm text-muted-foreground md:h-auto md:min-h-0 md:flex-1">
        {emptyMessage}
      </div>
    )
  }

  return (
    <div className="relative flex h-[50dvh] min-h-64 flex-col md:h-auto md:min-h-0 md:flex-1">
      <div className="absolute bottom-3 right-3 z-10 flex flex-col gap-1 rounded-lg border border-border bg-card/95 p-1 shadow-sm">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-11 md:size-8"
          aria-label="Acercar"
          onClick={() => setViewBox((current) => zoomViewBox(current, ZOOM_IN, fitted))}
        >
          <Plus />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-11 md:size-8"
          aria-label="Alejar"
          onClick={() => setViewBox((current) => zoomViewBox(current, ZOOM_OUT, fitted))}
        >
          <Minus />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-11 md:size-8"
          aria-label="Ajustar al plano"
          onClick={() => setViewBox(fitted)}
        >
          <Maximize2 />
        </Button>
      </div>

      <svg
        ref={svgRef}
        role={interactive ? 'group' : 'img'}
        aria-label="Plano del loteo"
        viewBox={viewBoxToString(viewBox)}
        className="min-h-0 w-full flex-1 touch-none touch-manipulation rounded-xl border border-border bg-card outline-none"
        style={{
          backgroundImage:
            'linear-gradient(var(--muted) 1px, transparent 1px), linear-gradient(90deg, var(--muted) 1px, transparent 1px)',
          backgroundSize: '24px 24px',
        }}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
        onClick={handleBackgroundClick}
      >
        {visiblePolygons.map((polygon) => {
          const selected = polygon.id === selectedPolygonId
          const style = polygonStyle(polygon, selected)
          const label = polygonLabels?.get(polygon.id) ?? DXF_LAYER_LABELS[polygon.layer]
          const state = lotStateOf(polygon)
          return (
            <g key={polygon.id} className="outline-none">
              <path
                d={polygonPaths.get(polygon.id) ?? ''}
                fill={style.fill}
                fillOpacity={style.fillOpacity}
                stroke={style.stroke}
                strokeWidth={style.strokeWidth}
                vectorEffect="non-scaling-stroke"
                className={
                  interactive
                    ? 'cursor-pointer outline-none focus-visible:stroke-ring focus-visible:stroke-[4]'
                    : 'outline-none'
                }
                role={interactive ? 'button' : undefined}
                tabIndex={interactive ? 0 : undefined}
                aria-pressed={interactive ? selected : undefined}
                aria-label={label}
                aria-description={state ? LOT_STATE_LABELS[state] : undefined}
                onClick={(event) => handlePathClick(event, polygon.id)}
                onKeyDown={(event) => handlePathKeyDown(event, polygon.id)}
              />
              <PolygonCaption polygon={polygon} />
            </g>
          )
        })}
      </svg>
    </div>
  )
}

function lotStateOf(polygon: DxfPolygon): LotState | null {
  return polygon.layer === 'LOTES' && polygon.lotState ? polygon.lotState : null
}

function polygonStyle(polygon: DxfPolygon, selected: boolean) {
  const state = lotStateOf(polygon)
  if (state) {
    return {
      fill: LOT_STATE_PAINT[state].fill,
      stroke: selected ? 'var(--foreground)' : LOT_STATE_PAINT[state].stroke,
      fillOpacity: selected ? SELECTED_FILL_OPACITY : LOT_STATE_FILL_OPACITY,
      strokeWidth: selected ? SELECTED_STROKE_WIDTH : LOT_STATE_STROKE_WIDTH,
    }
  }

  return {
    fill: LAYER_PAINT[polygon.layer].fill,
    stroke: selected ? 'var(--foreground)' : LAYER_PAINT[polygon.layer].stroke,
    fillOpacity: LAYER_FILL_OPACITY[polygon.layer],
    strokeWidth: selected ? SELECTED_STROKE_WIDTH : LAYER_STROKE_WIDTH[polygon.layer],
  }
}

const PolygonCaption = memo(function PolygonCaption({ polygon }: { polygon: DxfPolygon }) {
  if (!polygon.caption) {
    return null
  }

  if (polygon.layer === 'CALLE') {
    const layout = calleCaptionLayout(polygon.vertices, polygon.caption)
    if (!layout) {
      return null
    }

    const clipId = `${polygon.id}-caption-clip`
    return (
      <g style={{ pointerEvents: 'none' }} aria-hidden>
        <clipPath id={clipId}>
          <path d={polygonToSvgPath(polygon.vertices)} />
        </clipPath>
        <g clipPath={`url(#${clipId})`}>
          <text
            x={layout.x}
            y={layout.y}
            textAnchor="middle"
            dominantBaseline="middle"
            fontSize={layout.fontSize}
            fontWeight={700}
            fill="var(--foreground)"
            stroke="var(--card)"
            strokeWidth={layout.fontSize * 0.22}
            strokeLinejoin="round"
            paintOrder="stroke"
            transform={`rotate(${layout.rotate} ${layout.x} ${layout.y})`}
          >
            {polygon.caption}
          </text>
        </g>
      </g>
    )
  }

  const layout = loteCaptionLayout(polygon.vertices)
  if (!layout) {
    return null
  }

  return (
    <g style={{ pointerEvents: 'none' }} aria-hidden>
      <circle
        cx={layout.x}
        cy={layout.y}
        r={layout.fontSize * (0.62 + 0.18 * polygon.caption.length)}
        fill="var(--card)"
        fillOpacity={0.94}
        stroke="var(--foreground)"
        strokeWidth={layout.fontSize * 0.08}
      />
      <text
        x={layout.x}
        y={layout.y}
        textAnchor="middle"
        dominantBaseline="middle"
        fontSize={layout.fontSize}
        fontWeight={700}
        fill="var(--foreground)"
      >
        {polygon.caption}
      </text>
    </g>
  )
})
