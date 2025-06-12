package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"restapi/dto"

	"github.com/gin-gonic/gin"
	"github.com/phpdave11/gofpdf"
)

// Genera una factura en PDF para una cita específica
func GenerarFacturaPDF(c *gin.Context) {
	id := c.Param("id")
	rol, _ := c.Get("rol")

	if rol != "admin" && rol != "empleado" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo administradores o empleados pueden generar facturas"})
		return
	}

	var datos struct {
		Cliente   string
		Cedula    string
		Servicio  string
		Precio    float64
		FechaHora string
		Empleado  sql.NullString
	}

	err := dto.DB.QueryRow(`
		SELECT u.nombre, u.cedula, s.nombre, s.precio, c.fecha_hora, e.nombre
		FROM citas c
		JOIN usuarios u ON c.usuario_id = u.id
		JOIN servicios s ON c.servicio_id = s.id
		LEFT JOIN usuarios e ON c.empleado_id = e.id
		WHERE c.id = ? AND (c.estado = 'confirmada' OR c.estado = 'atendida')
	`, id).Scan(&datos.Cliente, &datos.Cedula, &datos.Servicio, &datos.Precio, &datos.FechaHora, &datos.Empleado)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se encontró la cita o no está confirmada/atendida"})
		return
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Factura de Servicio")

	pdf.Ln(12)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 10, fmt.Sprintf("Cliente: %s (%s)", datos.Cliente, datos.Cedula))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Servicio: %s", datos.Servicio))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Precio: ₡%.2f", datos.Precio))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Fecha: %s", datos.FechaHora))
	if datos.Empleado.Valid {
		pdf.Ln(8)
		pdf.Cell(0, 10, fmt.Sprintf("Empleado: %s", datos.Empleado.String))
	}

	c.Header("Content-Type", "application/pdf")
	_ = pdf.Output(c.Writer)
}
