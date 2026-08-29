package domain_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"loteosapp/backend/internal/business/domain"
)

func square() domain.Polygon {
	return domain.Polygon{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}
}

func TestPolygonNormalizeAcceptsAClosedRing(t *testing.T) {
	normalized, err := square().Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if len(normalized) != 4 {
		t.Errorf("Normalize() kept %d vertices, want 4", len(normalized))
	}
}

func TestPolygonNormalizeDropsTheRepeatedClosingVertex(t *testing.T) {
	ring := append(square(), domain.Point{X: 0, Y: 0})

	normalized, err := ring.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if len(normalized) != 4 {
		t.Errorf("Normalize() kept %d vertices, want 4", len(normalized))
	}
	if normalized[len(normalized)-1] == normalized[0] {
		t.Error("Normalize() should not leave the closing vertex repeated")
	}
}

func TestPolygonNormalizeDoesNotAliasTheInput(t *testing.T) {
	ring := square()

	normalized, err := ring.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	ring[0] = domain.Point{X: 99, Y: 99}
	if normalized[0].X == 99 {
		t.Error("Normalize() should return a copy, not a view over the caller's slice")
	}
}

func TestPolygonNormalizeRejectsUnusableRings(t *testing.T) {
	tooManyVertices := make(domain.Polygon, domain.MaxVerticesPerPolygon+2)

	tests := map[string]domain.Polygon{
		"empty":                       {},
		"two vertices":                {{X: 0, Y: 0}, {X: 1, Y: 1}},
		"three with a repeated close": {{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 0}},
		"collinear":                   {{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 2, Y: 2}},
		"zero area":                   {{X: 5, Y: 5}, {X: 5, Y: 5}, {X: 5, Y: 5}},
		"repeated vertex in the middle": {
			{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10},
		},
		"not a number":      {{X: math.NaN(), Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}},
		"infinite":          {{X: math.Inf(1), Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}},
		"too many vertices": tooManyVertices,
	}

	for name, polygon := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := polygon.Normalize(); !errors.Is(err, domain.ErrInvalidGeometry) {
				t.Errorf("Normalize() error = %v, want %v", err, domain.ErrInvalidGeometry)
			}
		})
	}
}

func TestPolygonNormalizeRejectsRingsThatCrossThemselves(t *testing.T) {
	tests := map[string]domain.Polygon{
		"bowtie": {{X: 0, Y: 0}, {X: 10, Y: 10}, {X: 10, Y: 0}, {X: 0, Y: 10}},
		"an edge that goes back through an earlier one": {
			{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 5, Y: -5}, {X: 0, Y: 10},
		},
		"a vertex sitting on a far edge": {
			{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 0}, {X: 5, Y: 10},
		},
	}

	for name, polygon := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := polygon.Normalize(); !errors.Is(err, domain.ErrSelfIntersectingRing) {
				t.Errorf("Normalize() error = %v, want %v", err, domain.ErrSelfIntersectingRing)
			}
		})
	}
}

func TestPolygonNormalizeAcceptsAConcaveRing(t *testing.T) {
	// An L-shaped parcel is concave but perfectly valid: the simplicity check
	// must not confuse a reflex vertex with a crossing.
	ring := domain.Polygon{
		{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 4}, {X: 4, Y: 4}, {X: 4, Y: 10}, {X: 0, Y: 10},
	}

	if _, err := ring.Normalize(); err != nil {
		t.Fatalf("Normalize() error = %v, want an L-shaped parcel to be accepted", err)
	}
}

func TestPolygonNormalizeAcceptsTheMaximumVertexCount(t *testing.T) {
	// A ring of exactly the limit, plus the optional repeated closing vertex,
	// is the largest legitimate polygon and must not be rejected.
	ring := make(domain.Polygon, 0, domain.MaxVerticesPerPolygon+1)
	for i := range domain.MaxVerticesPerPolygon {
		angle := 2 * math.Pi * float64(i) / float64(domain.MaxVerticesPerPolygon)
		ring = append(ring, domain.Point{X: math.Cos(angle), Y: math.Sin(angle)})
	}
	ring = append(ring, ring[0])

	normalized, err := ring.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if len(normalized) != domain.MaxVerticesPerPolygon {
		t.Errorf("Normalize() kept %d vertices, want %d", len(normalized), domain.MaxVerticesPerPolygon)
	}
}

func pointerTo[T any](value T) *T {
	return &value
}

func TestLoteDataValidate(t *testing.T) {
	tests := map[string]struct {
		data domain.LoteData
		want error
	}{
		"only the number": {
			data: domain.LoteData{Number: "12"},
		},
		"every value": {
			data: domain.LoteData{
				Number: "12", Price: pointerTo(150000.0), Currency: "ARS",
				Area: pointerTo(300.5), Features: "esquina",
			},
		},
		"a free lot is priced at zero": {
			data: domain.LoteData{Number: "12", Price: pointerTo(0.0), Currency: "USD"},
		},
		"the largest storable price": {
			data: domain.LoteData{Number: "12", Price: pointerTo(domain.MaxLotePrice), Currency: "ARS"},
		},
		"the largest storable area": {
			data: domain.LoteData{Number: "12", Area: pointerTo(domain.MaxLoteArea)},
		},
		"missing number": {
			data: domain.LoteData{Number: ""},
			want: domain.ErrInvalidLoteNumber,
		},
		"number past the column length": {
			data: domain.LoteData{Number: strings.Repeat("1", domain.MaxLoteNumberLength+1)},
			want: domain.ErrInvalidLoteNumber,
		},
		"negative price": {
			data: domain.LoteData{Number: "12", Price: pointerTo(-1.0), Currency: "ARS"},
			want: domain.ErrInvalidPrice,
		},
		"price that is not a number": {
			data: domain.LoteData{Number: "12", Price: pointerTo(math.NaN()), Currency: "ARS"},
			want: domain.ErrInvalidPrice,
		},
		"price past what the column holds": {
			data: domain.LoteData{Number: "12", Price: pointerTo(domain.MaxLotePrice * 10), Currency: "ARS"},
			want: domain.ErrInvalidPrice,
		},
		"price with more decimals than the column keeps": {
			data: domain.LoteData{Number: "12", Price: pointerTo(1000.005), Currency: "ARS"},
			want: domain.ErrInvalidPrice,
		},
		"price without a currency": {
			data: domain.LoteData{Number: "12", Price: pointerTo(1000.0)},
			want: domain.ErrInvalidCurrency,
		},
		"currency that is not a three letter code": {
			data: domain.LoteData{Number: "12", Currency: "PESOS"},
			want: domain.ErrInvalidCurrency,
		},
		"currency with a symbol": {
			data: domain.LoteData{Number: "12", Currency: "AR$"},
			want: domain.ErrInvalidCurrency,
		},
		"area of zero": {
			data: domain.LoteData{Number: "12", Area: pointerTo(0.0)},
			want: domain.ErrInvalidArea,
		},
		"negative area": {
			data: domain.LoteData{Number: "12", Area: pointerTo(-5.0)},
			want: domain.ErrInvalidArea,
		},
		"infinite area": {
			data: domain.LoteData{Number: "12", Area: pointerTo(math.Inf(1))},
			want: domain.ErrInvalidArea,
		},
		"area past what the column holds": {
			data: domain.LoteData{Number: "12", Area: pointerTo(domain.MaxLoteArea * 10)},
			want: domain.ErrInvalidArea,
		},
		"area with more decimals than the column keeps": {
			data: domain.LoteData{Number: "12", Area: pointerTo(300.00005)},
			want: domain.ErrInvalidArea,
		},
		"features past the accepted length": {
			data: domain.LoteData{Number: "12", Features: strings.Repeat("a", domain.MaxLoteFeaturesLength+1)},
			want: domain.ErrLoteFeaturesTooLong,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.data.Validate()
			if test.want == nil {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestHasRole(t *testing.T) {
	if !domain.HasRole([]string{domain.RolAgrimensor, domain.RolAdministrador}, domain.RolAdministrador) {
		t.Error("HasRole() should find a role present in the list")
	}
	if domain.HasRole([]string{domain.RolAgrimensor}, domain.RolAdministrador) {
		t.Error("HasRole() should not find a role absent from the list")
	}
	if domain.HasRole(nil, domain.RolAdministrador) {
		t.Error("HasRole() should be false for an actor with no roles")
	}
}
