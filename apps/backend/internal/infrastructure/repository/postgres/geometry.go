package postgres

import (
	"fmt"
	"strconv"
	"strings"

	"loteosapp/backend/internal/business/domain"
)

// parsePolygonWKT reads the ring PostGIS returns from ST_AsText(geom) for a
// geometry(Polygon) column: POLYGON((x y,x y,...,x y)), the first vertex
// repeated as the last. It returns the outer ring with each vertex listed
// once, matching domain.Polygon, and nil for an empty geometry. The write
// path stores a single outer ring per entity; if a value ever carries
// interior rings (POLYGON((outer),(hole))) only the outer one is read.
func parsePolygonWKT(wkt string) (domain.Polygon, error) {
	text := strings.TrimSpace(wkt)

	rest, ok := cutPrefixFold(text, "POLYGON")
	if !ok {
		return nil, fmt.Errorf("geometry %q is not a POLYGON", wkt)
	}
	if strings.EqualFold(strings.TrimSpace(rest), "EMPTY") {
		return nil, nil
	}

	open := strings.Index(text, "((")
	if open < 0 {
		return nil, fmt.Errorf("geometry %q has no ring", wkt)
	}
	open += 2
	// The outer ring ends at its own closing paren, before any interior ring
	// or the polygon's final paren.
	span := strings.IndexByte(text[open:], ')')
	if span < 0 {
		return nil, fmt.Errorf("geometry %q has an unterminated ring", wkt)
	}

	vertices := strings.Split(text[open:open+span], ",")
	ring := make(domain.Polygon, 0, len(vertices))
	for _, vertex := range vertices {
		coords := strings.Fields(vertex)
		if len(coords) < 2 {
			return nil, fmt.Errorf("geometry %q has a vertex without two coordinates", wkt)
		}

		x, err := strconv.ParseFloat(coords[0], 64)
		if err != nil {
			return nil, fmt.Errorf("geometry %q has a non-numeric x: %w", wkt, err)
		}
		y, err := strconv.ParseFloat(coords[1], 64)
		if err != nil {
			return nil, fmt.Errorf("geometry %q has a non-numeric y: %w", wkt, err)
		}

		ring = append(ring, domain.Point{X: x, Y: y})
	}

	if len(ring) >= 2 && ring[0] == ring[len(ring)-1] {
		ring = ring[:len(ring)-1]
	}
	if len(ring) < 3 {
		return nil, fmt.Errorf("geometry %q has fewer than three distinct vertices", wkt)
	}

	return ring, nil
}

// polygonFromNullableWKT is parsePolygonWKT for a column that may be NULL,
// which is how an entity with no DXF ring yet comes back from a LEFT JOIN.
func polygonFromNullableWKT(wkt *string) (domain.Polygon, error) {
	if wkt == nil {
		return nil, nil
	}

	return parsePolygonWKT(*wkt)
}

func cutPrefixFold(s, prefix string) (rest string, ok bool) {
	if len(s) < len(prefix) || !strings.EqualFold(s[:len(prefix)], prefix) {
		return s, false
	}

	return s[len(prefix):], true
}
