// Generación de reportes para administración (citas diarias, ingresos).

package api

import (
	"database/sql"
	"net/http"
	"restapi/dto"
	"time"

	"github.com/gin-gonic/gin"
)

func ReporteCitasPorFechas(c *gin.Context) {
	rol, _ := c.Get("rol")
	usuarioID, _ := c.Get("usuarioID")

	fechaInicio := c.Query("inicio") // formato YYYY-MM-DD
	fechaFin := c.Query("fin")       // formato YYYY-MM-DD
	empleadoFiltro := c.Query("empleado_id")

	// Validar fechas
	layout := "2006-01-02"
	start, err1 := time.Parse(layout, fechaInicio)
	end, err2 := time.Parse(layout, fechaFin)

	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido. Use YYYY-MM-DD"})
		return
	}

	// Query base
	query := `
		SELECT c.id, u.nombre AS cliente, u.cedula, s.nombre AS servicio, s.precio, 
		       c.fecha_hora, c.estado, e.nombre AS empleado
		FROM citas c
		JOIN usuarios u ON u.id = c.usuario_id
		JOIN servicios s ON s.id = c.servicio_id
		LEFT JOIN usuarios e ON e.id = c.empleado_id
		WHERE DATE(c.fecha_hora) BETWEEN ? AND ?
		  AND (c.estado = 'confirmada' OR c.estado = 'atendida')
	`

	var rows *sql.Rows
	var err error

	if rol == "admin" {
		if empleadoFiltro != "" {
			query += " AND c.empleado_id = ? ORDER BY c.fecha_hora"
			rows, err = dto.DB.Query(query, start, end, empleadoFiltro)
		} else {
			rows, err = dto.DB.Query(query+" ORDER BY c.fecha_hora", start, end)
		}
	} else if rol == "empleado" {
		query += " AND c.empleado_id = ? ORDER BY c.fecha_hora"
		rows, err = dto.DB.Query(query, start, end, usuarioID)
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Acceso denegado"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar reporte"})
		return
	}
	defer rows.Close()

	var reporte []map[string]interface{}
	for rows.Next() {
		var (
			id             int
			nombreCliente  string
			cedula         string
			nombreServicio string
			precio         float64
			fechaHora      time.Time
			estado         string
			nombreEmpleado sql.NullString
		)

		if err := rows.Scan(&id, &nombreCliente, &cedula, &nombreServicio, &precio, &fechaHora, &estado, &nombreEmpleado); err != nil {
			continue
		}

		reporte = append(reporte, gin.H{
			"id":       id,
			"cliente":  nombreCliente,
			"cedula":   cedula,
			"servicio": nombreServicio,
			"precio":   precio,
			"fecha":    fechaHora.Format("2006-01-02"),
			"hora":     fechaHora.Format("15:04"),
			"estado":   estado,
			"empleado": nombreEmpleado.String,
		})
	}

	c.JSON(http.StatusOK, reporte)
}
