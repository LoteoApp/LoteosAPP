package handler

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"codeberg.org/go-fonts/dejavu/dejavusans"
	"codeberg.org/go-fonts/dejavu/dejavusansbold"
	"codeberg.org/go-pdf/fpdf"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/reservations"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

const (
	receiptContentLeft  = 18.0
	receiptContentWidth = 174.0
	receiptSketchHeight = 88.0
)

var receiptArgentinaLocation = time.FixedZone("UTC-3", -3*60*60)

type reservationReceiptField struct {
	label string
	value string
}

type ReservationReceiptHandler struct {
	getReservationReceipt reservations.GetReservationReceipt
}

func NewReservationReceiptHandler(getReservationReceipt reservations.GetReservationReceipt) *ReservationReceiptHandler {
	return &ReservationReceiptHandler{getReservationReceipt: getReservationReceipt}
}

func (handler *ReservationReceiptHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	receiptData, err := handler.getReservationReceipt.Execute(request.Context(), reservations.Actor{
		AuthProviderID: principal.Subject,
		Roles:          principal.Roles,
	}, request.PathValue("id"))
	if err != nil {
		return err
	}

	receipt, err := reservationReceiptPDF(receiptData)
	if err != nil {
		return fmt.Errorf("generate reservation receipt: %w", err)
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="comprobante-reserva-`+receiptData.Reservation.ID+`.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(receipt)
	return err
}

func reservationReceiptPDF(receipt reservations.ReservationReceipt) ([]byte, error) {
	reservation := receipt.Reservation
	agency := "Sin inmobiliaria informada"
	if reservation.Inmobiliaria != nil {
		agency = reservation.Inmobiliaria.BusinessName
	}
	lot := reservation.LoteNumero
	if lot == "" {
		lot = reservation.LoteID
	}
	client := strings.TrimSpace(reservation.Cliente.Nombre + " " + reservation.Cliente.Apellido)
	seller := strings.TrimSpace(reservation.Vendedor.Nombre + " " + reservation.Vendedor.Apellido)

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Comprobante de reserva", false)
	pdf.SetAuthor("LoteosApp", false)
	pdf.SetLang("es-AR")
	pdf.AddUTF8FontFromBytes("DejaVu", "", dejavusans.TTF)
	pdf.AddUTF8FontFromBytes("DejaVu", "B", dejavusansbold.TTF)
	if err := pdf.Error(); err != nil {
		return nil, err
	}
	pdf.SetMargins(receiptContentLeft, 40, receiptContentLeft)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetCompression(false)
	pdf.AliasNbPages("")
	pdf.SetHeaderFunc(func() {
		writeReservationReceiptHeader(pdf, receipt.IssuedAt)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-14)
		pdf.SetDrawColor(218, 222, 218)
		pdf.Line(receiptContentLeft, pdf.GetY(), receiptContentLeft+receiptContentWidth, pdf.GetY())
		pdf.SetY(-12)
		pdf.SetFont("DejaVu", "", 8)
		pdf.SetTextColor(105, 113, 108)
		pdf.CellFormat(0, 4, fmt.Sprintf("Documento informativo - Página %d/{nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()
	writeReservationReceiptSummary(pdf, reservation)
	writeReservationReceiptSectionTitle(pdf, "DATOS DE LA RESERVA", 16)

	fields := []reservationReceiptField{
		{"LOTEO", reservation.LoteoNombre},
		{"LOTE", lot},
		{"CLIENTE", client},
		{"DNI", reservation.Cliente.DNI},
		{"VENDEDOR", seller},
		{"INMOBILIARIA", agency},
		{"FECHA DE RESERVA", formatReservationReceiptDate(reservation.FechaCreacion)},
		{"VENCIMIENTO", formatReservationReceiptDate(reservation.FechaVencimiento)},
	}
	for index := 0; index < len(fields); index += 2 {
		if err := writeReservationReceiptFieldPair(pdf, fields[index], fields[index+1]); err != nil {
			return nil, err
		}
	}

	writeReservationReceiptSectionTitle(pdf, "UBICACIÓN DEL LOTE", receiptSketchHeight+2)
	writeReservationReceiptSketch(pdf, receipt)
	writeReservationReceiptConditions(pdf)
	if err := pdf.Error(); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func writeReservationReceiptFieldPair(pdf *fpdf.Fpdf, left, right reservationReceiptField) error {
	left.value = reservationReceiptFieldValue(left.value)
	right.value = reservationReceiptFieldValue(right.value)
	columnWidth := (receiptContentWidth - 3) / 2
	pdf.SetFont("DejaVu", "", 10.5)
	leftLines := max(1, len(pdf.SplitText(left.value, columnWidth-10)))
	rightLines := max(1, len(pdf.SplitText(right.value, columnWidth-10)))
	height := 9.5 + float64(max(leftLines, rightLines))*5.2
	if height > 55 {
		if err := writeReservationReceiptField(pdf, left.label, left.value); err != nil {
			return err
		}
		return writeReservationReceiptField(pdf, right.label, right.value)
	}

	ensureReservationReceiptSpace(pdf, height+2)
	x, y := receiptContentLeft, pdf.GetY()
	writeReservationReceiptFieldCard(pdf, x, y, columnWidth, height, left)
	writeReservationReceiptFieldCard(pdf, x+columnWidth+3, y, columnWidth, height, right)
	pdf.SetY(y + height + 2)
	return pdf.Error()
}

func writeReservationReceiptFieldCard(pdf *fpdf.Fpdf, x, y, width, height float64, field reservationReceiptField) {
	pdf.SetFillColor(247, 249, 248)
	pdf.SetDrawColor(225, 231, 227)
	pdf.Rect(x, y, width, height, "DF")
	pdf.SetXY(x+5, y+2.5)
	pdf.SetFont("DejaVu", "B", 7.5)
	pdf.SetTextColor(101, 113, 106)
	pdf.CellFormat(width-10, 4, field.label, "", 1, "L", false, 0, "")
	pdf.SetXY(x+5, y+7)
	pdf.SetFont("DejaVu", "", 10.5)
	pdf.SetTextColor(37, 48, 42)
	pdf.MultiCell(width-10, 5.2, field.value, "", "L", false)
}

func writeReservationReceiptHeader(pdf *fpdf.Fpdf, issuedAt time.Time) {
	pdf.SetFillColor(31, 57, 48)
	pdf.Rect(0, 0, 210, 34, "F")
	pdf.SetXY(receiptContentLeft, 7)
	pdf.SetFont("DejaVu", "B", 8)
	pdf.SetTextColor(205, 221, 211)
	pdf.CellFormat(0, 4, "LOTEOSAPP", "", 1, "L", false, 0, "")
	pdf.SetXY(receiptContentLeft, 14)
	pdf.SetFont("DejaVu", "B", 17)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(108, 9, "COMPROBANTE DE RESERVA", "", 0, "L", false, 0, "")
	pdf.SetXY(137, 8)
	pdf.SetFont("DejaVu", "B", 7.5)
	pdf.SetTextColor(205, 221, 211)
	pdf.CellFormat(55, 4, "FECHA DE EMISIÓN", "", 1, "R", false, 0, "")
	pdf.SetX(137)
	pdf.SetFont("DejaVu", "", 10)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(55, 5, formatReservationReceiptDate(issuedAt), "", 0, "R", false, 0, "")
	pdf.SetY(40)
}

func writeReservationReceiptSummary(pdf *fpdf.Fpdf, reservation domain.Reservation) {
	ensureReservationReceiptSpace(pdf, 18)
	x, y := receiptContentLeft, pdf.GetY()
	pdf.SetFillColor(239, 244, 241)
	pdf.SetDrawColor(210, 220, 214)
	pdf.Rect(x, y, receiptContentWidth, 16, "DF")
	pdf.SetXY(x+5, y+3)
	pdf.SetFont("DejaVu", "B", 7.5)
	pdf.SetTextColor(93, 107, 99)
	pdf.CellFormat(118, 3.5, "NÚMERO DE RESERVA", "", 0, "L", false, 0, "")
	pdf.CellFormat(46, 3.5, "ESTADO", "", 1, "R", false, 0, "")
	pdf.SetX(x + 5)
	pdf.SetFont("DejaVu", "B", 10.5)
	pdf.SetTextColor(31, 57, 48)
	pdf.CellFormat(118, 6, reservation.ID, "", 0, "L", false, 0, "")
	state := strings.ToUpper(string(reservation.Estado))
	if state == "" {
		state = "NO INFORMADO"
	}
	pdf.CellFormat(46, 6, state, "", 0, "R", false, 0, "")
	pdf.SetY(y + 20)
}

func writeReservationReceiptSectionTitle(pdf *fpdf.Fpdf, title string, contentHeight float64) {
	ensureReservationReceiptSpace(pdf, 8+contentHeight)
	x, y := receiptContentLeft, pdf.GetY()
	pdf.SetFont("DejaVu", "B", 9)
	pdf.SetTextColor(31, 57, 48)
	pdf.SetXY(x, y)
	pdf.CellFormat(receiptContentWidth, 5, title, "", 1, "L", false, 0, "")
	pdf.SetDrawColor(195, 207, 200)
	pdf.Line(x, y+6, x+receiptContentWidth, y+6)
	pdf.SetY(y + 9)
}

func writeReservationReceiptField(pdf *fpdf.Fpdf, label, value string) error {
	value = reservationReceiptFieldValue(value)

	pdf.SetFont("DejaVu", "", 10.5)
	lines := pdf.SplitText(value, receiptContentWidth-10)
	lineCount := max(1, len(lines))
	height := 9.5 + float64(lineCount)*5.2
	if height <= 220 {
		ensureReservationReceiptSpace(pdf, height+2)
		x, y := receiptContentLeft, pdf.GetY()
		pdf.SetFillColor(247, 249, 248)
		pdf.SetDrawColor(225, 231, 227)
		pdf.Rect(x, y, receiptContentWidth, height, "DF")
		pdf.SetXY(x+5, y+2.5)
		pdf.SetFont("DejaVu", "B", 7.5)
		pdf.SetTextColor(101, 113, 106)
		pdf.CellFormat(receiptContentWidth-10, 4, label, "", 1, "L", false, 0, "")
		pdf.SetXY(x+5, y+7)
		pdf.SetFont("DejaVu", "", 10.5)
		pdf.SetTextColor(37, 48, 42)
		pdf.MultiCell(receiptContentWidth-10, 5.2, value, "", "L", false)
		pdf.SetY(y + height + 2)
		return pdf.Error()
	}

	ensureReservationReceiptSpace(pdf, 14)
	pdf.SetFillColor(247, 249, 248)
	pdf.SetFont("DejaVu", "B", 7.5)
	pdf.SetTextColor(101, 113, 106)
	pdf.CellFormat(receiptContentWidth, 6, label, "TLR", 1, "L", true, 0, "")
	pdf.SetFont("DejaVu", "", 10.5)
	pdf.SetTextColor(37, 48, 42)
	pdf.MultiCell(receiptContentWidth, 5.2, value, "LRB", "L", true)
	pdf.Ln(2)
	return pdf.Error()
}

func reservationReceiptFieldValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "No informado"
	}
	return value
}

func writeReservationReceiptSketch(pdf *fpdf.Fpdf, receipt reservations.ReservationReceipt) {
	ensureReservationReceiptSpace(pdf, receiptSketchHeight+2)
	x, y := receiptContentLeft, pdf.GetY()
	pdf.SetFillColor(250, 251, 250)
	pdf.SetDrawColor(207, 216, 211)
	pdf.Rect(x, y, receiptContentWidth, receiptSketchHeight, "DF")
	pdf.SetXY(x+6, y+4)
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetTextColor(31, 57, 48)
	pdf.CellFormat(receiptContentWidth-12, 5, "Croquis de ubicación del lote", "", 0, "L", false, 0, "")

	selected, found := reservationReceiptSelectedLot(receipt)
	if !found || !reservationReceiptPolygonDrawable(selected.Polygon) {
		writeReservationReceiptSketchFallback(pdf, x, y, "El lote reservado no tiene geometría cargada.")
		pdf.SetY(y + receiptSketchHeight + 4)
		return
	}

	polygons := make([]domain.Polygon, 0, len(receipt.Loteo.Lotes)+1)
	if reservationReceiptPolygonDrawable(receipt.Loteo.Boundary) {
		polygons = append(polygons, receipt.Loteo.Boundary)
	}
	for _, lot := range receipt.Loteo.Lotes {
		if reservationReceiptPolygonDrawable(lot.Polygon) {
			polygons = append(polygons, lot.Polygon)
		}
	}
	bounds, ok := reservationReceiptGeometryBounds(polygons)
	if !ok {
		writeReservationReceiptSketchFallback(pdf, x, y, "No hay geometría suficiente para dibujar el croquis.")
		pdf.SetY(y + receiptSketchHeight + 4)
		return
	}

	plotX, plotY := x+7, y+14
	plotWidth, plotHeight := receiptContentWidth-14, receiptSketchHeight-25
	project := reservationReceiptProjector(bounds, plotX, plotY, plotWidth, plotHeight)
	if reservationReceiptPolygonDrawable(receipt.Loteo.Boundary) {
		pdf.SetDrawColor(50, 77, 65)
		pdf.SetLineWidth(0.65)
		pdf.Polygon(project(receipt.Loteo.Boundary), "D")
	}

	pdf.SetLineWidth(0.25)
	pdf.SetDrawColor(134, 145, 139)
	pdf.SetFillColor(255, 255, 255)
	for _, lot := range receipt.Loteo.Lotes {
		if lot.ID == selected.ID || !reservationReceiptPolygonDrawable(lot.Polygon) {
			continue
		}
		pdf.Polygon(project(lot.Polygon), "DF")
	}

	pdf.SetLineWidth(0.6)
	pdf.SetDrawColor(31, 57, 48)
	pdf.SetFillColor(233, 184, 78)
	selectedPoints := project(selected.Polygon)
	pdf.Polygon(selectedPoints, "DF")
	lotLabel := receipt.Reservation.LoteNumero
	if lotLabel == "" {
		lotLabel = receipt.Reservation.LoteID
	}
	if labelPoint, ok := reservationReceiptPolygonLabelPoint(selectedPoints); ok {
		pdf.SetFont("DejaVu", "B", 9)
		pdf.SetTextColor(31, 57, 48)
		for fontSize := 9.0; fontSize >= 5.5 && pdf.GetStringWidth(lotLabel) > 36; fontSize -= 0.5 {
			pdf.SetFont("DejaVu", "B", fontSize)
		}
		labelWidth := math.Min(math.Max(pdf.GetStringWidth(lotLabel)+4, 10), 40)
		pdf.SetXY(labelPoint.X-labelWidth/2, labelPoint.Y-2.5)
		pdf.CellFormat(labelWidth, 5, lotLabel, "", 0, "C", false, 0, "")
	}

	legendY := y + receiptSketchHeight - 7
	pdf.SetFillColor(233, 184, 78)
	pdf.SetDrawColor(31, 57, 48)
	pdf.Rect(x+7, legendY+0.5, 4, 4, "DF")
	pdf.SetXY(x+14, legendY)
	pdf.SetFont("DejaVu", "", 7.5)
	pdf.SetTextColor(74, 86, 79)
	pdf.CellFormat(80, 5, "Lote reservado", "", 0, "L", false, 0, "")
	pdf.SetXY(x+96, legendY)
	pdf.SetFont("DejaVu", "B", 7.5)
	pdf.CellFormat(receiptContentWidth-103, 5, "REFERENCIA - SIN ESCALA", "", 0, "R", false, 0, "")
	pdf.SetY(y + receiptSketchHeight + 4)
}

func writeReservationReceiptSketchFallback(pdf *fpdf.Fpdf, x, y float64, detail string) {
	pdf.SetXY(x+16, y+32)
	pdf.SetFont("DejaVu", "B", 11)
	pdf.SetTextColor(80, 94, 86)
	pdf.CellFormat(receiptContentWidth-32, 6, "Croquis no disponible", "", 1, "C", false, 0, "")
	pdf.SetX(x + 16)
	pdf.SetFont("DejaVu", "", 9)
	pdf.SetTextColor(111, 121, 115)
	pdf.MultiCell(receiptContentWidth-32, 5, detail, "", "C", false)
	pdf.SetXY(x+96, y+receiptSketchHeight-7)
	pdf.SetFont("DejaVu", "B", 7.5)
	pdf.SetTextColor(74, 86, 79)
	pdf.CellFormat(receiptContentWidth-103, 5, "REFERENCIA - SIN ESCALA", "", 0, "R", false, 0, "")
}

func writeReservationReceiptConditions(pdf *fpdf.Fpdf) {
	disclaimer := "Este documento es un comprobante de reserva y no implica una venta. La reserva puede ser cancelada por el cliente o por la inmobiliaria. Si en 15 días no se concreta la venta, la reserva se cancela automáticamente."
	pdf.SetFont("DejaVu", "", 9.5)
	lineCount := max(1, len(pdf.SplitText(disclaimer, receiptContentWidth-12)))
	height := 12 + float64(lineCount)*5.2
	ensureReservationReceiptSpace(pdf, height)
	x, y := receiptContentLeft, pdf.GetY()
	pdf.SetFillColor(248, 244, 234)
	pdf.SetDrawColor(225, 215, 190)
	pdf.SetLineWidth(0.25)
	pdf.Rect(x, y, receiptContentWidth, height, "DF")
	pdf.SetXY(x+6, y+3)
	pdf.SetFont("DejaVu", "B", 9)
	pdf.SetTextColor(92, 72, 35)
	pdf.CellFormat(receiptContentWidth-12, 5, "CONDICIONES DE LA RESERVA", "", 1, "L", false, 0, "")
	pdf.SetX(x + 6)
	pdf.SetFont("DejaVu", "", 9.5)
	pdf.SetTextColor(58, 67, 61)
	pdf.MultiCell(receiptContentWidth-12, 5.2, disclaimer, "", "L", false)
	pdf.SetY(y + height + 2)
}

func ensureReservationReceiptSpace(pdf *fpdf.Fpdf, height float64) {
	_, pageHeight := pdf.GetPageSize()
	_, bottomMargin := pdf.GetAutoPageBreak()
	if pdf.GetY()+height > pageHeight-bottomMargin {
		pdf.AddPage()
	}
}

func formatReservationReceiptDate(value time.Time) string {
	if value.IsZero() {
		return "No informada"
	}
	return value.In(receiptArgentinaLocation).Format("02/01/2006 15:04")
}

func reservationReceiptSelectedLot(receipt reservations.ReservationReceipt) (domain.Lote, bool) {
	for _, lot := range receipt.Loteo.Lotes {
		if lot.ID == receipt.Reservation.LoteID {
			return lot, true
		}
	}
	return domain.Lote{}, false
}

func reservationReceiptPolygonDrawable(polygon domain.Polygon) bool {
	if len(polygon) < 3 {
		return false
	}
	for _, point := range polygon {
		if math.IsNaN(point.X) || math.IsInf(point.X, 0) || math.IsNaN(point.Y) || math.IsInf(point.Y, 0) {
			return false
		}
	}
	return true
}

type reservationReceiptBounds struct {
	minX float64
	maxX float64
	minY float64
	maxY float64
}

func reservationReceiptGeometryBounds(polygons []domain.Polygon) (reservationReceiptBounds, bool) {
	bounds := reservationReceiptBounds{
		minX: math.Inf(1), maxX: math.Inf(-1), minY: math.Inf(1), maxY: math.Inf(-1),
	}
	for _, polygon := range polygons {
		for _, point := range polygon {
			bounds.minX = math.Min(bounds.minX, point.X)
			bounds.maxX = math.Max(bounds.maxX, point.X)
			bounds.minY = math.Min(bounds.minY, point.Y)
			bounds.maxY = math.Max(bounds.maxY, point.Y)
		}
	}
	return bounds, bounds.maxX > bounds.minX && bounds.maxY > bounds.minY
}

func reservationReceiptProjector(bounds reservationReceiptBounds, x, y, width, height float64) func(domain.Polygon) []fpdf.PointType {
	padding := 3.0
	availableWidth := width - 2*padding
	availableHeight := height - 2*padding
	scale := math.Min(availableWidth/(bounds.maxX-bounds.minX), availableHeight/(bounds.maxY-bounds.minY))
	drawnWidth := (bounds.maxX - bounds.minX) * scale
	drawnHeight := (bounds.maxY - bounds.minY) * scale
	offsetX := x + padding + (availableWidth-drawnWidth)/2
	offsetY := y + padding + (availableHeight-drawnHeight)/2

	return func(polygon domain.Polygon) []fpdf.PointType {
		points := make([]fpdf.PointType, len(polygon))
		for index, point := range polygon {
			points[index] = fpdf.PointType{
				X: offsetX + (point.X-bounds.minX)*scale,
				Y: offsetY + drawnHeight - (point.Y-bounds.minY)*scale,
			}
		}
		return points
	}
}

func reservationReceiptPolygonLabelPoint(points []fpdf.PointType) (fpdf.PointType, bool) {
	if centroid, ok := reservationReceiptPolygonCentroid(points); ok && reservationReceiptPointInPolygon(centroid, points) {
		return centroid, true
	}

	yValues := make([]float64, 0, len(points))
	for _, point := range points {
		yValues = append(yValues, point.Y)
	}
	sort.Float64s(yValues)

	bestWidth := 0.0
	var best fpdf.PointType
	for index := 0; index < len(yValues)-1; index++ {
		if math.Abs(yValues[index+1]-yValues[index]) < 1e-9 {
			continue
		}
		y := (yValues[index] + yValues[index+1]) / 2
		intersections := reservationReceiptScanlineIntersections(points, y)
		for intersection := 0; intersection+1 < len(intersections); intersection += 2 {
			width := intersections[intersection+1] - intersections[intersection]
			candidate := fpdf.PointType{X: (intersections[intersection] + intersections[intersection+1]) / 2, Y: y}
			if width > bestWidth && reservationReceiptPointInPolygon(candidate, points) {
				best = candidate
				bestWidth = width
			}
		}
	}
	return best, bestWidth > 0
}

func reservationReceiptPolygonCentroid(points []fpdf.PointType) (fpdf.PointType, bool) {
	var twiceArea, weightedX, weightedY float64
	for index, current := range points {
		next := points[(index+1)%len(points)]
		cross := current.X*next.Y - next.X*current.Y
		twiceArea += cross
		weightedX += (current.X + next.X) * cross
		weightedY += (current.Y + next.Y) * cross
	}
	if math.Abs(twiceArea) < 1e-9 {
		return fpdf.PointType{}, false
	}
	return fpdf.PointType{
		X: weightedX / (3 * twiceArea),
		Y: weightedY / (3 * twiceArea),
	}, true
}

func reservationReceiptScanlineIntersections(points []fpdf.PointType, y float64) []float64 {
	intersections := make([]float64, 0, len(points))
	for index, current := range points {
		next := points[(index+1)%len(points)]
		if (current.Y > y) == (next.Y > y) {
			continue
		}
		x := current.X + (y-current.Y)*(next.X-current.X)/(next.Y-current.Y)
		intersections = append(intersections, x)
	}
	sort.Float64s(intersections)
	return intersections
}

func reservationReceiptPointInPolygon(point fpdf.PointType, polygon []fpdf.PointType) bool {
	inside := false
	for index, current := range polygon {
		next := polygon[(index+1)%len(polygon)]
		if reservationReceiptPointOnSegment(point, current, next) {
			return true
		}
		if (current.Y > point.Y) != (next.Y > point.Y) &&
			point.X < (next.X-current.X)*(point.Y-current.Y)/(next.Y-current.Y)+current.X {
			inside = !inside
		}
	}
	return inside
}

func reservationReceiptPointOnSegment(point, start, end fpdf.PointType) bool {
	const tolerance = 1e-7
	cross := (point.Y-start.Y)*(end.X-start.X) - (point.X-start.X)*(end.Y-start.Y)
	if math.Abs(cross) > tolerance {
		return false
	}
	return point.X >= math.Min(start.X, end.X)-tolerance && point.X <= math.Max(start.X, end.X)+tolerance &&
		point.Y >= math.Min(start.Y, end.Y)-tolerance && point.Y <= math.Max(start.Y, end.Y)+tolerance
}
