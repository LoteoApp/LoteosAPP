import type { DxfPoint } from '../types'

// A fixed absolute epsilon breaks down once coordinates are large (Gauss-
// Krüger easting/northing run in the millions, where a double's own ULP is
// already ~1e-9): floating-point noise from the coordinate values themselves
// can exceed the epsilon. Scale it to the magnitude of the values being
// compared instead, matching the standard robust-orientation-predicate
// approach.
const RELATIVE_EPSILON = 1e-9

export function distance(a: DxfPoint, b: DxfPoint): number {
  return Math.hypot(a.x - b.x, a.y - b.y)
}

function orientation(a: DxfPoint, b: DxfPoint, c: DxfPoint): number {
  const term1 = (b.x - a.x) * (c.y - a.y)
  const term2 = (b.y - a.y) * (c.x - a.x)
  const value = term1 - term2
  const scale = Math.abs(term1) + Math.abs(term2)
  if (scale === 0 || Math.abs(value) <= RELATIVE_EPSILON * scale) return 0
  return value > 0 ? 1 : -1
}

function onSegment(a: DxfPoint, b: DxfPoint, p: DxfPoint): boolean {
  const scale = Math.max(Math.abs(a.x), Math.abs(a.y), Math.abs(b.x), Math.abs(b.y), 1)
  const eps = RELATIVE_EPSILON * scale
  return (
    Math.min(a.x, b.x) - eps <= p.x &&
    p.x <= Math.max(a.x, b.x) + eps &&
    Math.min(a.y, b.y) - eps <= p.y &&
    p.y <= Math.max(a.y, b.y) + eps
  )
}

// Counts touching (shared endpoint, collinear overlap) as an intersection.
// Used for self-intersection: a ring that merely touches itself at a point
// is still a degenerate, non-simple polygon.
function segmentsIntersect(p1: DxfPoint, p2: DxfPoint, p3: DxfPoint, p4: DxfPoint): boolean {
  const o1 = orientation(p1, p2, p3)
  const o2 = orientation(p1, p2, p4)
  const o3 = orientation(p3, p4, p1)
  const o4 = orientation(p3, p4, p2)

  if (o1 !== o2 && o3 !== o4) return true

  if (o1 === 0 && onSegment(p1, p2, p3)) return true
  if (o2 === 0 && onSegment(p1, p2, p4)) return true
  if (o3 === 0 && onSegment(p3, p4, p1)) return true
  if (o4 === 0 && onSegment(p3, p4, p2)) return true

  return false
}

function pointsCoincide(p: DxfPoint, q: DxfPoint): boolean {
  const scale = Math.max(Math.abs(p.x), Math.abs(p.y), Math.abs(q.x), Math.abs(q.y), 1)
  return distance(p, q) <= RELATIVE_EPSILON * scale
}

// Requires a genuine transversal crossing (all four orientations non-zero).
// Excludes shared edges/vertices so adjacent, boundary-sharing polygons
// (e.g. two lots that share a property line or a corner where three lots
// meet) aren't flagged as overlapping. The orientation epsilon alone isn't
// enough for a shared corner: when p3/p4 is a near-duplicate of p1/p2 (two
// independently-drawn vertices for what's meant to be the same point), the
// cross-product terms scale down together with the gap between them, so the
// relative-epsilon check in orientation() never zeroes out. Checking point
// coincidence directly catches it.
function segmentsCrossProperly(p1: DxfPoint, p2: DxfPoint, p3: DxfPoint, p4: DxfPoint): boolean {
  if (
    pointsCoincide(p1, p3) ||
    pointsCoincide(p1, p4) ||
    pointsCoincide(p2, p3) ||
    pointsCoincide(p2, p4)
  ) {
    return false
  }

  const o1 = orientation(p1, p2, p3)
  const o2 = orientation(p1, p2, p4)
  const o3 = orientation(p3, p4, p1)
  const o4 = orientation(p3, p4, p2)

  return o1 !== 0 && o2 !== 0 && o3 !== 0 && o4 !== 0 && o1 !== o2 && o3 !== o4
}

export function pointInPolygon(point: DxfPoint, vertices: DxfPoint[]): boolean {
  let inside = false
  for (let i = 0, j = vertices.length - 1; i < vertices.length; j = i++) {
    const vi = vertices[i]
    const vj = vertices[j]
    const crosses =
      vi.y > point.y !== vj.y > point.y &&
      point.x < ((vj.x - vi.x) * (point.y - vi.y)) / (vj.y - vi.y) + vi.x
    if (crosses) inside = !inside
  }
  return inside
}

// The average of the vertices, not a lone vertex: a vertex is often shared
// with the neighboring polygon's boundary (adjacent lots reuse each other's
// property line), which makes point-in-polygon undefined right where it's
// evaluated. The vertex average stays clear of that edge case for the
// convex, simple parcel shapes this validates.
// Gauss area formula (shoelace). Coordinates are metres (Gauss-Krüger), so
// the result is m². Returns null when the ring has no interior.
export function polygonArea(vertices: DxfPoint[]): number | null {
  const ring = openRing(vertices)
  if (ring.length < 3) {
    return null
  }

  let twice = 0
  for (let i = 0; i < ring.length; i++) {
    const current = ring[i]
    const next = ring[(i + 1) % ring.length]
    twice += current.x * next.y - next.x * current.y
  }

  if (Math.abs(twice) < 1e-9) {
    return null
  }

  return Math.abs(twice) / 2
}

function openRing(vertices: DxfPoint[]): DxfPoint[] {
  if (vertices.length < 2) {
    return vertices
  }
  const first = vertices[0]
  const last = vertices[vertices.length - 1]
  if (first.x === last.x && first.y === last.y) {
    return vertices.slice(0, -1)
  }
  return vertices
}

export function centroid(vertices: DxfPoint[]): DxfPoint {
  let sumX = 0
  let sumY = 0
  for (const { x, y } of vertices) {
    sumX += x
    sumY += y
  }
  return { x: sumX / vertices.length, y: sumY / vertices.length }
}

function boundingBoxesOverlap(a: DxfPoint[], b: DxfPoint[]): boolean {
  const boxA = boundingBox(a)
  const boxB = boundingBox(b)
  const scale = Math.max(
    Math.abs(boxA.minX),
    Math.abs(boxA.maxX),
    Math.abs(boxA.minY),
    Math.abs(boxA.maxY),
    Math.abs(boxB.minX),
    Math.abs(boxB.maxX),
    Math.abs(boxB.minY),
    Math.abs(boxB.maxY),
    1,
  )
  const eps = RELATIVE_EPSILON * scale
  return (
    boxA.minX - eps <= boxB.maxX &&
    boxB.minX - eps <= boxA.maxX &&
    boxA.minY - eps <= boxB.maxY &&
    boxB.minY - eps <= boxA.maxY
  )
}

function boundingBox(vertices: DxfPoint[]) {
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const { x, y } of vertices) {
    if (x < minX) minX = x
    if (y < minY) minY = y
    if (x > maxX) maxX = x
    if (y > maxY) maxY = y
  }
  return { minX, minY, maxX, maxY }
}

export function isSimplePolygon(vertices: DxfPoint[]): boolean {
  const n = vertices.length
  if (n < 3) return false

  for (let i = 0; i < n; i++) {
    const a1 = vertices[i]
    const a2 = vertices[(i + 1) % n]
    for (let j = i + 1; j < n; j++) {
      const adjacent = j === i + 1 || (i === 0 && j === n - 1)
      if (adjacent) continue

      const b1 = vertices[j]
      const b2 = vertices[(j + 1) % n]
      if (segmentsIntersect(a1, a2, b1, b2)) return false
    }
  }

  return true
}

// Polygons that only touch — a shared property line or corner between
// adjacent lots — are not considered overlapping.
export function polygonsOverlap(a: DxfPoint[], b: DxfPoint[]): boolean {
  if (!boundingBoxesOverlap(a, b)) return false

  for (let i = 0; i < a.length; i++) {
    const a1 = a[i]
    const a2 = a[(i + 1) % a.length]
    for (let j = 0; j < b.length; j++) {
      const b1 = b[j]
      const b2 = b[(j + 1) % b.length]
      if (segmentsCrossProperly(a1, a2, b1, b2)) return true
    }
  }

  if (pointInPolygon(centroid(a), b)) return true
  if (pointInPolygon(centroid(b), a)) return true

  return false
}
