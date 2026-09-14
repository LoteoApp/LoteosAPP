package handler

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"codeberg.org/go-pdf/fpdf"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/reservations"
)

func TestReservationReceiptPDFRendersStructuredContentAndOnlyLabelsReservedLot(t *testing.T) {
	receipt := reservationReceiptFixture()
	pdf, err := reservationReceiptPDF(receipt)
	if err != nil {
		t.Fatalf("reservationReceiptPDF() error = %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("PDF has invalid header: %q", pdf[:min(len(pdf), 8)])
	}
	if outputPath := os.Getenv("LOTEOS_RESERVATION_RECEIPT_PDF_PATH"); outputPath != "" {
		if err := os.WriteFile(outputPath, pdf, 0o600); err != nil {
			t.Fatalf("write receipt review PDF: %v", err)
		}
	}

	for _, text := range []string{
		"LOTEOSAPP",
		"COMPROBANTE DE RESERVA",
		"FECHA DE EMISIÓN",
		"11/09/2026 12:30",
		"NÚMERO DE RESERVA",
		"DATOS DE LA RESERVA",
		"Urbanización Ñandú de Córdoba",
		"Łukasz Dvořák",
		"José Muñoz",
		"06/09/2026 09:00",
		"UBICACIÓN DEL LOTE",
		"Croquis de ubicación del lote",
		"Lote reservado",
		"REFERENCIA - SIN ESCALA",
		"CONDICIONES DE LA RESERVA",
		"automáticamente",
		"Página 1/1",
	} {
		if !bytes.Contains(pdf, reservationReceiptUTF16Bytes(text)) {
			t.Errorf("PDF does not preserve %q", text)
		}
	}
	if bytes.Contains(pdf, reservationReceiptUTF16Bytes("LOTE-AJENO-13")) {
		t.Error("PDF labels a lot other than the reserved lot")
	}
	if pages := bytes.Count(pdf, []byte("/Type /Page\n")); pages != 1 {
		t.Fatalf("regular receipt produced %d pages, want 1", pages)
	}
}

func TestReservationReceiptPDFShowsGeometryFallback(t *testing.T) {
	receipt := reservationReceiptFixture()
	receipt.Loteo.Lotes[0].Polygon = nil

	pdf, err := reservationReceiptPDF(receipt)
	if err != nil {
		t.Fatalf("reservationReceiptPDF() error = %v", err)
	}
	for _, text := range []string{"Croquis no disponible", "El lote reservado no tiene geometría cargada.", "REFERENCIA - SIN ESCALA"} {
		if !bytes.Contains(pdf, reservationReceiptUTF16Bytes(text)) {
			t.Errorf("PDF does not include fallback %q", text)
		}
	}
}

func TestReservationReceiptPDFKeepsFixedSectionsTogetherAcrossPages(t *testing.T) {
	receipt := reservationReceiptFixture()
	receipt.Reservation.LoteoNombre = strings.Repeat("Urbanización Ñandú de Córdoba ", 120) + "Último sector"

	pdf, err := reservationReceiptPDF(receipt)
	if err != nil {
		t.Fatalf("reservationReceiptPDF() error = %v", err)
	}
	pages := bytes.Count(pdf, []byte("/Type /Page\n"))
	if pages < 2 {
		t.Fatalf("long values produced %d page objects, want at least 2", pages)
	}
	for _, text := range []string{"Último", "sector", "Croquis de ubicación del lote", "CONDICIONES DE LA RESERVA"} {
		if !bytes.Contains(pdf, reservationReceiptUTF16Bytes(text)) {
			t.Errorf("multipage PDF does not preserve %q", text)
		}
	}
	for page := 1; page <= pages; page++ {
		footer := "Página " + string(rune('0'+page)) + "/" + string(rune('0'+pages))
		if page <= 9 && pages <= 9 && !bytes.Contains(pdf, reservationReceiptUTF16Bytes(footer)) {
			t.Errorf("multipage PDF does not include footer %q", footer)
		}
	}
}

func TestReservationReceiptPDFRasterHighlightsSelectedLot(t *testing.T) {
	pdftoppm, err := exec.LookPath("pdftoppm")
	if err != nil {
		t.Skip("pdftoppm is not installed")
	}

	leftReceipt := reservationReceiptFixture()
	leftPDF, err := reservationReceiptPDF(leftReceipt)
	if err != nil {
		t.Fatalf("left reservationReceiptPDF() error = %v", err)
	}
	leftCentroid := reservationReceiptAmberCentroid(t, pdftoppm, leftPDF, "left")

	rightReceipt := reservationReceiptFixture()
	rightReceipt.Reservation.LoteID = "lot-13"
	rightReceipt.Reservation.LoteNumero = "13"
	rightPDF, err := reservationReceiptPDF(rightReceipt)
	if err != nil {
		t.Fatalf("right reservationReceiptPDF() error = %v", err)
	}
	rightCentroid := reservationReceiptAmberCentroid(t, pdftoppm, rightPDF, "right")

	if rightCentroid <= leftCentroid+80 {
		t.Errorf("highlight centroid did not move to selected lot: left %.1f, right %.1f", leftCentroid, rightCentroid)
	}
}

func TestReservationReceiptPDFPlacesLabelInsideConcaveLot(t *testing.T) {
	concaveLot := domain.Polygon{
		{X: 0, Y: 0},
		{X: 10, Y: 0},
		{X: 10, Y: 2},
		{X: 2, Y: 2},
		{X: 2, Y: 10},
		{X: 0, Y: 10},
	}
	bounds, ok := reservationReceiptGeometryBounds([]domain.Polygon{concaveLot})
	if !ok {
		t.Fatal("concave fixture has no drawable bounds")
	}
	points := reservationReceiptProjector(bounds, 0, 0, 100, 100)(concaveLot)

	var vertexAverage fpdf.PointType
	for _, point := range points {
		vertexAverage.X += point.X
		vertexAverage.Y += point.Y
	}
	vertexAverage.X /= float64(len(points))
	vertexAverage.Y /= float64(len(points))
	if reservationReceiptPointInPolygon(vertexAverage, points) {
		t.Fatal("concave fixture does not demonstrate the vertex-average regression")
	}

	labelPoint, ok := reservationReceiptPolygonLabelPoint(points)
	if !ok || !reservationReceiptPointInPolygon(labelPoint, points) {
		t.Fatalf("label point = %#v, inside = %t", labelPoint, reservationReceiptPointInPolygon(labelPoint, points))
	}

	receipt := reservationReceiptFixture()
	receipt.Loteo.Boundary = domain.Polygon{{X: -1, Y: -1}, {X: 11, Y: -1}, {X: 11, Y: 11}, {X: -1, Y: 11}}
	receipt.Loteo.Lotes = []domain.Lote{{ID: "lot-7", Number: "7", Polygon: concaveLot}}
	pdf, err := reservationReceiptPDF(receipt)
	if err != nil {
		t.Fatalf("reservationReceiptPDF() error = %v", err)
	}
	if !bytes.Contains(pdf, reservationReceiptUTF16Bytes("7")) {
		t.Error("concave lot PDF does not include the reserved lot label")
	}
	if outputPath := os.Getenv("LOTEOS_RESERVATION_RECEIPT_CONCAVE_PDF_PATH"); outputPath != "" {
		if err := os.WriteFile(outputPath, pdf, 0o600); err != nil {
			t.Fatalf("write concave receipt review PDF: %v", err)
		}
	}
}

func reservationReceiptAmberCentroid(t *testing.T, pdftoppm string, data []byte, name string) float64 {
	t.Helper()
	directory := t.TempDir()
	inputPath := filepath.Join(directory, name+".pdf")
	outputPrefix := filepath.Join(directory, name)
	if err := os.WriteFile(inputPath, data, 0o600); err != nil {
		t.Fatalf("write raster input: %v", err)
	}
	command := exec.Command(pdftoppm, "-f", "1", "-singlefile", "-r", "96", "-png", inputPath, outputPrefix)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("pdftoppm: %v: %s", err, output)
	}
	imageFile, err := os.Open(outputPrefix + ".png")
	if err != nil {
		t.Fatalf("open raster output: %v", err)
	}
	defer imageFile.Close()
	image, err := png.Decode(imageFile)
	if err != nil {
		t.Fatalf("decode raster output: %v", err)
	}

	var weightedX, pixels float64
	for y := image.Bounds().Min.Y; y < image.Bounds().Max.Y; y++ {
		for x := image.Bounds().Min.X; x < image.Bounds().Max.X; x++ {
			r, g, b, _ := image.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			if r8 >= 220 && r8 <= 245 && g8 >= 165 && g8 <= 200 && b8 >= 55 && b8 <= 105 {
				weightedX += float64(x)
				pixels++
			}
		}
	}
	if pixels < 500 {
		t.Fatalf("rendered PDF has only %.0f highlighted pixels", pixels)
	}
	return weightedX / pixels
}

func reservationReceiptFixture() reservations.ReservationReceipt {
	return reservations.ReservationReceipt{
		Reservation: domain.Reservation{
			ID:            "b519169a-a41e-4c04-a82d-c56e92948c49",
			LoteoID:       "loteo-1",
			LoteoNombre:   "Urbanización Ñandú de Córdoba",
			LoteID:        "lot-7",
			LoteNumero:    "7",
			Estado:        domain.ReservationStateActive,
			Cliente:       domain.Cliente{Nombre: "Łukasz", Apellido: "Dvořák", DNI: "30111222"},
			Vendedor:      domain.ReservationActor{Nombre: "José", Apellido: "Muñoz"},
			Inmobiliaria:  &domain.ReservationAgency{BusinessName: "Inmobiliaria del Sur"},
			FechaCreacion: time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC),
			FechaVencimiento: time.Date(
				2026, time.September, 21, 12, 0, 0, 0, time.UTC,
			),
		},
		Loteo: domain.Loteo{
			ID:       "loteo-1",
			Boundary: domain.Polygon{{X: 0, Y: 0}, {X: 200, Y: 0}, {X: 200, Y: 100}, {X: 0, Y: 100}},
			Lotes: []domain.Lote{
				{ID: "lot-7", Number: "7", Polygon: domain.Polygon{{X: 10, Y: 10}, {X: 90, Y: 10}, {X: 90, Y: 90}, {X: 10, Y: 90}}},
				{ID: "lot-13", Number: "LOTE-AJENO-13", Polygon: domain.Polygon{{X: 110, Y: 10}, {X: 190, Y: 10}, {X: 190, Y: 90}, {X: 110, Y: 90}}},
			},
		},
		IssuedAt: time.Date(2026, time.September, 11, 15, 30, 0, 0, time.UTC),
	}
}

func reservationReceiptUTF16Bytes(value string) []byte {
	units := utf16.Encode([]rune(value))
	encoded := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.BigEndian.PutUint16(encoded[index*2:], unit)
	}
	return encoded
}
