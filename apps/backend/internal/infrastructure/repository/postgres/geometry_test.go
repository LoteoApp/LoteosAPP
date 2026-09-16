package postgres

import (
	"testing"

	"loteosapp/backend/internal/business/domain"
)

func TestParsePolygonWKT(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		wkt  string
		want domain.Polygon
	}{
		"drops the repeated closing vertex": {
			wkt: "POLYGON((0 0,10 0,10 10,0 10,0 0))",
			want: domain.Polygon{
				{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10},
			},
		},
		"keeps an already open ring": {
			wkt:  "POLYGON((0 0,10 0,5 8))",
			want: domain.Polygon{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 8}},
		},
		"tolerates spaces after commas": {
			wkt:  "POLYGON ((0 0, 10 0, 5 8, 0 0))",
			want: domain.Polygon{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 8}},
		},
		"reads negative and decimal coordinates": {
			wkt:  "POLYGON((-1.5 -2.25,3 0,0 4.5,-1.5 -2.25))",
			want: domain.Polygon{{X: -1.5, Y: -2.25}, {X: 3, Y: 0}, {X: 0, Y: 4.5}},
		},
		"reads only the outer ring when interior rings are present": {
			wkt: "POLYGON((0 0,10 0,10 10,0 10,0 0),(2 2,4 2,4 4,2 4,2 2))",
			want: domain.Polygon{
				{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePolygonWKT(test.wkt)
			if err != nil {
				t.Fatalf("parsePolygonWKT(%q) error = %v", test.wkt, err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("parsePolygonWKT(%q) = %v, want %v", test.wkt, got, test.want)
			}
			for i := range test.want {
				if got[i] != test.want[i] {
					t.Errorf("vertex %d = %v, want %v", i, got[i], test.want[i])
				}
			}
		})
	}
}

func TestParsePolygonWKTEmptyGeometryIsNoRing(t *testing.T) {
	t.Parallel()

	got, err := parsePolygonWKT("POLYGON EMPTY")
	if err != nil {
		t.Fatalf("parsePolygonWKT() error = %v", err)
	}
	if got != nil {
		t.Errorf("parsePolygonWKT(\"POLYGON EMPTY\") = %v, want nil", got)
	}
}

func TestParsePolygonWKTRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	invalid := map[string]string{
		"not a polygon":        "LINESTRING(0 0,1 1)",
		"no ring":              "POLYGON",
		"missing a coordinate": "POLYGON((0 0,10,5 8,0 0))",
		"non-numeric":          "POLYGON((0 0,ten 0,5 8,0 0))",
		"too few vertices":     "POLYGON((0 0,10 0,0 0))",
	}

	for name, wkt := range invalid {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := parsePolygonWKT(wkt); err == nil {
				t.Errorf("parsePolygonWKT(%q) error = nil, want an error", wkt)
			}
		})
	}
}

func TestPolygonFromNullableWKT(t *testing.T) {
	t.Parallel()

	got, err := polygonFromNullableWKT(nil)
	if err != nil || got != nil {
		t.Fatalf("polygonFromNullableWKT(nil) = %v, %v, want nil, nil", got, err)
	}

	wkt := "POLYGON((0 0,1 0,1 1,0 0))"
	got, err = polygonFromNullableWKT(&wkt)
	if err != nil {
		t.Fatalf("polygonFromNullableWKT() error = %v", err)
	}
	if len(got) != 3 {
		t.Errorf("polygonFromNullableWKT() = %v, want 3 vertices", got)
	}
}
