import type { DxfPolygon } from '../types'

export type SvgViewBox = {
  x: number
  y: number
  width: number
  height: number
}

const DEFAULT_SPAN = 1
const PADDING_RATIO = 0.08

export function viewBoxToString(viewBox: SvgViewBox): string {
  return `${viewBox.x} ${viewBox.y} ${viewBox.width} ${viewBox.height}`
}

export function zoomViewBox(viewBox: SvgViewBox, factor: number): SvgViewBox {
  const width = viewBox.width * factor
  const height = viewBox.height * factor
  return {
    x: viewBox.x + (viewBox.width - width) / 2,
    y: viewBox.y + (viewBox.height - height) / 2,
    width,
    height,
  }
}

export function panViewBox(
  viewBox: SvgViewBox,
  dxPx: number,
  dyPx: number,
  svgWidth: number,
  svgHeight: number,
): SvgViewBox {
  if (svgWidth === 0 || svgHeight === 0) {
    return viewBox
  }

  return {
    ...viewBox,
    x: viewBox.x - (dxPx * viewBox.width) / svgWidth,
    y: viewBox.y - (dyPx * viewBox.height) / svgHeight,
  }
}

export function fitViewBox(polygons: DxfPolygon[]): SvgViewBox {
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity

  for (const polygon of polygons) {
    for (const vertex of polygon.vertices) {
      if (vertex.x < minX) minX = vertex.x
      if (vertex.x > maxX) maxX = vertex.x
      if (vertex.y < minY) minY = vertex.y
      if (vertex.y > maxY) maxY = vertex.y
    }
  }

  if (!Number.isFinite(minX)) {
    return { x: 0, y: 0, width: DEFAULT_SPAN, height: DEFAULT_SPAN }
  }

  const rawWidth = maxX - minX
  const rawHeight = maxY - minY
  const width = rawWidth === 0 ? DEFAULT_SPAN : rawWidth
  const height = rawHeight === 0 ? DEFAULT_SPAN : rawHeight
  const padX = width * PADDING_RATIO
  const padY = height * PADDING_RATIO

  return {
    x: minX - padX,
    y: -maxY - padY,
    width: width + padX * 2,
    height: height + padY * 2,
  }
}
