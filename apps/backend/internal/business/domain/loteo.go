package domain

import (
	"math"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalidLoteoName     = &Error{Kind: KindInvalid, Code: "invalid_loteo_nombre", Message: "El nombre del loteo es obligatorio"}
	ErrInvalidGeometry      = &Error{Kind: KindInvalid, Code: "invalid_geometry", Message: "La geometría recibida no es un polígono cerrado válido"}
	ErrSelfIntersectingRing = &Error{Kind: KindInvalid, Code: "self_intersecting_geometry", Message: "La geometría recibida se cruza consigo misma"}
	ErrPlanTooLarge         = &Error{Kind: KindInvalid, Code: "plan_too_large", Message: "El plano supera la cantidad de polígonos admitida"}
	ErrUnknownManzana       = &Error{Kind: KindInvalid, Code: "unknown_manzana", Message: "Un lote referencia una manzana que no está en el plano"}
	ErrInvalidManzanaRef    = &Error{Kind: KindInvalid, Code: "invalid_manzana_ref", Message: "Cada manzana del plano necesita una referencia propia y no vacía"}
	ErrPlanWithoutLoteo     = &Error{Kind: KindInvalid, Code: "plan_without_loteo", Message: "El plano no tiene el polígono de la capa LOTEO"}
	ErrLoteNotFound         = &Error{Kind: KindNotFound, Code: "lote_not_found", Message: "El lote solicitado no existe"}
	ErrLoteNumberInUse      = &Error{Kind: KindConflict, Code: "lote_numero_in_use", Message: "Ya existe un lote con ese número en este loteo"}
	ErrInvalidLoteNumber    = &Error{Kind: KindInvalid, Code: "invalid_lote_numero", Message: "El número de lote es obligatorio y no puede superar los 32 caracteres"}
	ErrInvalidPrice         = &Error{Kind: KindInvalid, Code: "invalid_precio", Message: "El precio debe ser un monto no negativo, de hasta 2 decimales y menor a 1.000.000.000.000"}
	ErrInvalidCurrency      = &Error{Kind: KindInvalid, Code: "invalid_moneda", Message: "La moneda debe ser un código de tres letras"}
	ErrInvalidArea          = &Error{Kind: KindInvalid, Code: "invalid_superficie", Message: "La superficie debe ser mayor a cero, de hasta 4 decimales y menor a 100.000.000"}
	ErrLoteFeaturesTooLong  = &Error{Kind: KindInvalid, Code: "lote_caracteristicas_too_long", Message: "Las características del lote no pueden superar los 2000 caracteres"}
	ErrLoteoNotFound        = &Error{Kind: KindNotFound, Code: "loteo_not_found", Message: "El loteo solicitado no existe"}
	ErrInvalidDxfFile       = &Error{Kind: KindInvalid, Code: "invalid_dxf_file", Message: "El archivo DXF es inválido o supera el tamaño permitido"}
)

// A DXF plan arrives as JSON from a client we don't control, so the vertex
// count of a single ring, the number of rings in a plan and the vertices of
// the whole plan are all bounded before any per-polygon work runs. The limits
// sit far above a real cadastral plan (a large loteo is hundreds of lots of a
// few dozen vertices each) and keep the O(v²) simplicity check bounded.
const (
	MaxVerticesPerPolygon = 1_000
	MaxPolygonsPerPlan    = 25_000
	MaxVerticesPerPlan    = 250_000
)

// Limits of the columns a lot's manually loaded values are stored in:
// numero and caracteristicas are text, precio is NUMERIC(14,2) and
// superficie NUMERIC(12,4). A value past them would either be rejected by
// PostgreSQL as an unexpected failure or silently rounded.
const (
	MaxLoteNumberLength   = 32
	MaxLoteFeaturesLength = 2_000
	MaxLotePrice          = 999_999_999_999.99
	MaxLoteArea           = 99_999_999.9999
	lotePriceDecimals     = 2
	loteAreaDecimals      = 4
)

// MaxDxfFileBytes bounds the original DXF upload. It mirrors MAX_DXF_FILE_BYTES
// in apps/frontend/src/features/lots/lib/readDxfFile.ts; the backend can't
// trust the client's check.
const MaxDxfFileBytes = 20_000_000

// Point is a vertex in the DXF's own coordinate system. Georeferencing is a
// separate concern, so no spatial reference system is implied here.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Polygon is a closed ring listing each vertex once: the segment from the
// last vertex back to the first is implicit, which is the shape the DXF
// parser in the frontend emits.
type Polygon []Point

// Normalize checks that the ring is a usable simple polygon and returns a
// copy in canonical form, dropping the closing vertex when the client
// repeated it. Whether this ring overlaps another one of the same layer is
// not decided here: that's a relationship between entities, not a property of
// a ring (see docs/domain.md).
func (polygon Polygon) Normalize() (Polygon, error) {
	if len(polygon) > MaxVerticesPerPolygon+1 {
		return nil, ErrInvalidGeometry
	}

	for _, point := range polygon {
		if math.IsNaN(point.X) || math.IsInf(point.X, 0) ||
			math.IsNaN(point.Y) || math.IsInf(point.Y, 0) {
			return nil, ErrInvalidGeometry
		}
	}

	ring := polygon
	if len(ring) >= 2 && ring[0] == ring[len(ring)-1] {
		ring = ring[:len(ring)-1]
	}
	if len(ring) < 3 {
		return nil, ErrInvalidGeometry
	}
	if hasZeroLengthEdge(ring) {
		return nil, ErrInvalidGeometry
	}
	// Simplicity is checked before area because a ring that crosses itself can
	// enclose zero net area (a symmetric bowtie does), and the crossing is the
	// more useful thing to report back.
	if !isSimple(ring) {
		return nil, ErrSelfIntersectingRing
	}
	if isDegenerate(ring) {
		return nil, ErrInvalidGeometry
	}

	normalized := make(Polygon, len(ring))
	copy(normalized, ring)

	return normalized, nil
}

func hasZeroLengthEdge(ring Polygon) bool {
	for i, current := range ring {
		if current == ring[(i+1)%len(ring)] {
			return true
		}
	}

	return false
}

// isDegenerate reports a ring whose vertices are all collinear, which has no
// interior and would be stored as a polygon that encloses nothing. The
// shoelace sum of such a ring is zero in exact arithmetic and a rounding
// residue in floating point, so it's compared against a tolerance far below
// the area of any real parcel (coordinates are metres).
func isDegenerate(ring Polygon) bool {
	const minimumTwiceArea = 1e-9

	var total float64
	for i, current := range ring {
		next := ring[(i+1)%len(ring)]
		total += current.X*next.Y - next.X*current.Y
	}

	return math.Abs(total) < minimumTwiceArea
}

// isSimple reports whether the ring's edges only meet where they're supposed
// to: at the vertex two consecutive edges share. A ring that crosses itself
// is a valid list of points but not a parcel, and PostGIS would store it as
// an invalid polygon that later spatial queries can't be trusted on.
func isSimple(ring Polygon) bool {
	total := len(ring)

	for i := range total {
		for j := i + 1; j < total; j++ {
			if j == i+1 || (i == 0 && j == total-1) {
				continue
			}
			if segmentsIntersect(ring[i], ring[i+1], ring[j], ring[(j+1)%total]) {
				return false
			}
		}
	}

	return true
}

func segmentsIntersect(a, b, c, d Point) bool {
	abc := orientation(a, b, c)
	abd := orientation(a, b, d)
	cda := orientation(c, d, a)
	cdb := orientation(c, d, b)

	if abc != abd && cda != cdb {
		return true
	}

	return (abc == 0 && isBetween(a, b, c)) ||
		(abd == 0 && isBetween(a, b, d)) ||
		(cda == 0 && isBetween(c, d, a)) ||
		(cdb == 0 && isBetween(c, d, b))
}

// orientation reports the turn direction of a -> b -> c: 1 counter-clockwise,
// -1 clockwise, 0 collinear.
func orientation(a, b, c Point) int {
	cross := (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)

	switch {
	case cross > 0:
		return 1
	case cross < 0:
		return -1
	default:
		return 0
	}
}

// isBetween reports whether point, already known to be collinear with the
// segment a-b, lies inside its bounding box and therefore on the segment.
func isBetween(a, b, point Point) bool {
	return point.X >= math.Min(a.X, b.X) && point.X <= math.Max(a.X, b.X) &&
		point.Y >= math.Min(a.Y, b.Y) && point.Y <= math.Max(a.Y, b.Y)
}

// DxfEntity is one polygon read from a DXF layer. Handle is the DXF entity
// handle when the file carried one; it's kept for traceability back to the
// source drawing and is not an identifier the system relies on.
type DxfEntity struct {
	Handle  string
	Polygon Polygon
}

// DxfLote is a lot polygon together with the position, in DxfPlan.Manzanas,
// of the manzana it belongs to. The client sends that relationship by
// reference and the use case resolves it to an index, so persistence never
// handles a client-supplied key.
type DxfLote struct {
	DxfEntity
	ManzanaIndex int
}

// DxfPlan is the geometry of a loteo as extracted by the frontend parser.
// The backend validates and stores it; it never parses the DXF itself.
type DxfPlan struct {
	Loteo    DxfEntity
	Manzanas []DxfEntity
	Lotes    []DxfLote
	Calles   []DxfEntity
}

// NewLoteo is a loteo about to be created. Plan is nil when the loteo is
// registered before its DXF is available, which the domain allows.
type NewLoteo struct {
	Name        string
	Location    string
	Description string
	Plan        *DxfPlan
}

// LoteData are the per-lot values loaded manually after the plan is drawn:
// the DXF layers carry geometry only, with no text.
type LoteData struct {
	Number   string
	Price    *float64
	Currency string
	Area     *float64
	Features string
}

// Validate checks the values a caller may set on a lot. Price, Currency and
// Area are optional, but a price without a currency is meaningless, so they're
// required together.
func (data LoteData) Validate() error {
	if data.Number == "" || utf8.RuneCountInString(data.Number) > MaxLoteNumberLength {
		return ErrInvalidLoteNumber
	}
	if data.Price != nil {
		if !isStorableAmount(*data.Price, MaxLotePrice, lotePriceDecimals) || *data.Price < 0 {
			return ErrInvalidPrice
		}
		if data.Currency == "" {
			return ErrInvalidCurrency
		}
	}
	if data.Currency != "" && !isCurrencyCode(data.Currency) {
		return ErrInvalidCurrency
	}
	if data.Area != nil {
		if !isStorableAmount(*data.Area, MaxLoteArea, loteAreaDecimals) || *data.Area <= 0 {
			return ErrInvalidArea
		}
	}
	if utf8.RuneCountInString(data.Features) > MaxLoteFeaturesLength {
		return ErrLoteFeaturesTooLong
	}

	return nil
}

// isStorableAmount reports whether value fits the NUMERIC column it's headed
// for: within its magnitude and with no decimals past its scale, which
// PostgreSQL would round away without saying so.
func isStorableAmount(value, maximum float64, decimals int) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > maximum {
		return false
	}

	scaled := value * math.Pow(10, float64(decimals))
	// The tolerance grows with the magnitude because a float64 can't hold a
	// residue smaller than its own ulp at that scale.
	tolerance := math.Max(1e-6, math.Abs(scaled)*1e-12)

	return math.Abs(scaled-math.Round(scaled)) <= tolerance
}

func isCurrencyCode(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for _, letter := range currency {
		if letter < 'A' || letter > 'Z' {
			return false
		}
	}

	return true
}

// Loteo lists its manzanas, lotes and calles flat, the way the tables do:
// a lote names its manzana through ManzanaID. On a create it keeps the order
// the plan was submitted in, so a client can match each row back to the
// polygon it sent. Boundary and the per-entity Polygon are filled when the
// loteo is read back with its geometry, and omitted otherwise.
type Loteo struct {
	ID          string    `json:"id"`
	Name        string    `json:"nombre"`
	Location    string    `json:"ubicacion"`
	Description string    `json:"descripcion"`
	Boundary    Polygon   `json:"contorno,omitempty"`
	Manzanas    []Manzana `json:"manzanas"`
	Lotes       []Lote    `json:"lotes"`
	Calles      []Calle   `json:"calles"`
	CreatedAt   time.Time `json:"fechaCreacion"`
}

type Manzana struct {
	ID      string  `json:"id"`
	Number  string  `json:"numero"`
	Polygon Polygon `json:"poligono,omitempty"`
}

type Lote struct {
	ID        string   `json:"id"`
	ManzanaID string   `json:"manzanaId"`
	Number    string   `json:"numero"`
	Price     *float64 `json:"precio"`
	Currency  string   `json:"moneda"`
	Area      *float64 `json:"superficie"`
	Features  string   `json:"caracteristicas"`
	Polygon   Polygon  `json:"poligono,omitempty"`
}

type Calle struct {
	ID      string  `json:"id"`
	Name    string  `json:"nombre"`
	Type    string  `json:"tipo"`
	Polygon Polygon `json:"poligono,omitempty"`
}

// LoteoSummary is a loteo as it appears in a listing: identity, how much of a
// plan it already carries, and whether its original DXF is on file. It never
// carries geometry — the list stays cheap to build and small on the wire.
type LoteoSummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"nombre"`
	Location     string    `json:"ubicacion"`
	Description  string    `json:"descripcion"`
	ManzanaCount int       `json:"cantidadManzanas"`
	LoteCount    int       `json:"cantidadLotes"`
	CalleCount   int       `json:"cantidadCalles"`
	HasPlan      bool      `json:"tienePlano"`
	HasDxfFile   bool      `json:"tieneDxf"`
	CreatedAt    time.Time `json:"fechaCreacion"`
}

// NewLoteoDxfFile is the original DXF about to be recorded for a loteo, once
// its bytes are stored in object storage under StorageKey.
type NewLoteoDxfFile struct {
	StorageKey   string
	OriginalName string
	MimeType     string
	Sha256       string
}

// LoteoDxfFile is the recorded original DXF of a loteo.
type LoteoDxfFile struct {
	ID           string
	StorageKey   string
	OriginalName string
	MimeType     string
	Sha256       string
	CreatedAt    time.Time
}
