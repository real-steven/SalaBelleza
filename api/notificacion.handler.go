// Simulación de envío de notificaciones para recordatorio de citas.

package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"restapi/dto"
	"time"

	"github.com/gin-gonic/gin"
)

// EnviarNotificacion simula el envío de un recordatorio de cita por correo electrónico.
func EnviarNotificacion(c *gin.Context) {
	id := c.Param("id")

	// Verificamos que el rol sea empleado o admin
	rol, _ := c.Get("rol")
	if rol != "empleado" && rol != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo empleados o administradores pueden enviar recordatorios"})
		return
	}

	var correo string
	var fecha time.Time

	// Consulta para obtener el correo del cliente y la fecha de la cita
	err := dto.DB.QueryRow(`
		SELECT u.correo, c.fecha_hora 
		FROM citas c 
		JOIN usuarios u ON u.id = c.usuario_id 
		WHERE c.id = ?`, id).Scan(&correo, &fecha)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Cita no encontrada"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la cita"})
		}
		return
	}

	// Simulación de envío de correo
	fmt.Printf("📧 Simulando envío de recordatorio a %s para su cita el %s\n", correo, fecha.Format("2006-01-02 15:04"))

	// Respuesta mock
	c.JSON(http.StatusOK, gin.H{
		"mensaje": fmt.Sprintf("Notificación enviada a %s para su cita el %s", correo, fecha.Format("2006-01-02 15:04")),
	})
}
