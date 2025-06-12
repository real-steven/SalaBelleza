// Manejador de operaciones relacionadas con citas (crear, cancelar, modificar).

package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"restapi/dto"
	"time"

	"github.com/gin-gonic/gin"
)

type CrearCitaInput struct {
	ServicioID int32     `json:"servicio_id"`
	FechaHora  time.Time `json:"fecha_hora"`
}

func CancelarCita(c *gin.Context) {
	id := c.Param("id")

	// 🔐 Recuperar datos del token con conversión segura
	rolRaw, _ := c.Get("rol")
	usuarioIDRaw, _ := c.Get("usuarioID")

	rol, ok := rolRaw.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Rol inválido"})
		return
	}

	usuarioIDFloat, ok := usuarioIDRaw.(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ID de usuario inválido"})
		return
	}
	usuarioID := int(usuarioIDFloat)

	// 🕒 Obtener cita
	var fechaHora time.Time
	var dueñoID sql.NullInt64
	err := dto.DB.QueryRow("SELECT fecha_hora, usuario_id FROM citas WHERE id=?", id).Scan(&fechaHora, &dueñoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cita no encontrada"})
		return
	}

	// Leer motivo desde el body (esperamos JSON)
	var input struct {
		Motivo string `json:"motivo"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Motivo requerido"})
		return
	}

	// Admin puede cancelar siempre
	if rol == "admin" {
		_, err := dto.DB.Exec("UPDATE citas SET estado='cancelada', cancelacion_motivo=? WHERE id=?", input.Motivo, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cancelar cita"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"mensaje": "Cita cancelada correctamente por administrador"})
		return
	}

	// Cliente puede cancelar solo su propia cita (con 12h de anticipación)
	if rol == "cliente" && dueñoID.Valid && int(dueñoID.Int64) == usuarioID {
		if time.Until(fechaHora) < 12*time.Hour {
			c.JSON(http.StatusForbidden, gin.H{"error": "Solo puede cancelar con al menos 12h de antelación"})
			return
		}
		_, err := dto.DB.Exec("UPDATE citas SET estado='cancelada', cancelacion_motivo=? WHERE id=?", input.Motivo, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cancelar cita"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"mensaje": "Cita cancelada correctamente"})
		return
	}

	// Si no cumple ninguna condición
	c.JSON(http.StatusForbidden, gin.H{"error": "No tiene permiso para cancelar esta cita"})
}

func ConfirmarCita(c *gin.Context) {
	id := c.Param("id")
	rol, _ := c.Get("rol")

	if rol != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo administradores pueden confirmar citas"})
		return
	}

	var input struct {
		EmpleadoID int32 `json:"empleado_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.EmpleadoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe enviar un empleado_id válido"})
		return
	}

	// Verificar si la cita existe y está pendiente
	var estado string
	err := dto.DB.QueryRow("SELECT estado FROM citas WHERE id = ?", id).Scan(&estado)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cita no encontrada"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar cita"})
		return
	}
	if estado != "pendiente" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solo se pueden confirmar citas pendientes"})
		return
	}

	// Validar que el empleado exista
	var existe int
	err = dto.DB.QueryRow("SELECT COUNT(*) FROM usuarios WHERE id = ? AND rol = 'empleado'", input.EmpleadoID).Scan(&existe)
	if err != nil || existe == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empleado no válido"})
		return
	}

	res, err := dto.DB.Exec("UPDATE citas SET estado='confirmada', empleado_id=? WHERE id=?", input.EmpleadoID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar cita"})
		return
	}

	afectadas, _ := res.RowsAffected()
	if afectadas == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se actualizó ninguna cita"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Cita confirmada y empleado asignado"})
}

func RechazarCita(c *gin.Context) {
	id := c.Param("id")
	rol, _ := c.Get("rol")

	if rol != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo administradores pueden rechazar citas"})
		return
	}

	// Verificar que la cita exista y esté pendiente
	var estado string
	err := dto.DB.QueryRow("SELECT estado FROM citas WHERE id = ?", id).Scan(&estado)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cita no encontrada"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar cita"})
		return
	}
	if estado != "pendiente" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solo se pueden rechazar citas pendientes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Cita rechazada correctamente"})
}

func MisCitasCliente(c *gin.Context) {
	usuarioID, existe := c.Get("usuarioID")
	if !existe {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}
	rol, _ := c.Get("rol")

	query := `
		SELECT c.id, c.servicio_id, c.fecha_hora, c.estado, c.empleado_id,
		       c.creado_en, c.actualizado_en, s.nombre AS nombre_servicio,
		       s.precio, u.cedula, u.nombre AS nombre_cliente
		FROM citas c
		JOIN servicios s ON c.servicio_id = s.id
		JOIN usuarios u ON c.usuario_id = u.id`

	var rows *sql.Rows
	var err error

	if rol == "admin" {
		rows, err = dto.DB.Query(query + " ORDER BY c.fecha_hora DESC")
	} else {
		rows, err = dto.DB.Query(query+" WHERE c.usuario_id = ? ORDER BY c.fecha_hora DESC", usuarioID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudieron obtener las citas"})
		return
	}
	defer rows.Close()

	var citas []map[string]interface{}

	for rows.Next() {
		var (
			id, servicioID                        int
			empleadoID                            sql.NullInt32
			fechaHora, estado                     string
			creadoEn, actualizadoEn               sql.NullTime
			nombreServicio, nombreCliente, cedula string
			precio                                float64
		)

		err := rows.Scan(&id, &servicioID, &fechaHora, &estado, &empleadoID,
			&creadoEn, &actualizadoEn, &nombreServicio, &precio, &cedula, &nombreCliente)
		if err != nil {
			fmt.Println("❌ Error en Scan:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar cita"})
			return
		}

		cita := map[string]interface{}{
			"id":         id,
			"fecha_hora": fechaHora,
			"estado":     estado,
			"servicio": map[string]interface{}{
				"id":     servicioID,
				"nombre": nombreServicio,
				"precio": precio,
			},
			"cliente": map[string]interface{}{
				"nombre": nombreCliente,
				"cedula": cedula,
				"id":     usuarioID,
			},
			"creado_en":      creadoEn.Time,
			"actualizado_en": actualizadoEn.Time,
		}

		// Solo agregamos empleado_id si es válido
		if empleadoID.Valid {
			cita["empleado_id"] = empleadoID.Int32
		} else {
			cita["empleado_id"] = nil
		}

		citas = append(citas, cita)
	}

	c.JSON(http.StatusOK, citas)
}
func CrearCita(c *gin.Context) {
	var input CrearCitaInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Verificar que la fecha no esté en el pasado
	if input.FechaHora.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La fecha debe ser futura"})
		return
	}

	// Verificar que el servicio exista
	var exists int
	err := dto.DB.QueryRow("SELECT COUNT(*) FROM servicios WHERE id = ?", input.ServicioID).Scan(&exists)
	if err != nil || exists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El servicio no existe"})
		return
	}

	usuarioID, _ := c.Get("usuarioID")

	_, err = dto.DB.Exec("INSERT INTO citas (usuario_id, servicio_id, fecha_hora, estado) VALUES (?, ?, ?, ?)",
		usuarioID, input.ServicioID, input.FechaHora, "pendiente")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear cita"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mensaje": "Cita creada exitosamente"})
}

func ObtenerCita(c *gin.Context) {
	id := c.Param("id")

	var cita struct {
		ID         int32     `json:"id"`
		UsuarioID  int32     `json:"usuario_id"`
		ServicioID int32     `json:"servicio_id"`
		FechaHora  time.Time `json:"fecha_hora"`
		Estado     string    `json:"estado"`
	}

	err := dto.DB.QueryRow("SELECT id, usuario_id, servicio_id, fecha_hora, estado FROM citas WHERE id = ?", id).
		Scan(&cita.ID, &cita.UsuarioID, &cita.ServicioID, &cita.FechaHora, &cita.Estado)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Cita no encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al buscar cita"})
		return
	}

	c.JSON(http.StatusOK, cita)
}

func ActualizarCita(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo administradores pueden actualizar citas"})
		return
	}

	id := c.Param("id")

	var input struct {
		ServicioID int32     `json:"servicio_id"`
		FechaHora  time.Time `json:"fecha_hora"`
		Estado     string    `json:"estado"`
		EmpleadoID *int32    `json:"empleado_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Validar estado permitido
	estadosPermitidos := map[string]bool{
		"pendiente":  true,
		"confirmada": true,
		"cancelada":  true,
		"rechazada":  true,
		"atendida":   true,
	}
	if !estadosPermitidos[input.Estado] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Estado inválido"})
		return
	}

	// Ejecutar actualización
	_, err := dto.DB.Exec(`
		UPDATE citas 
		SET servicio_id=?, fecha_hora=?, estado=?, empleado_id=?
		WHERE id=?`,
		input.ServicioID, input.FechaHora, input.Estado, input.EmpleadoID, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar cita"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Cita actualizada correctamente"})
}

func CrearCitaInvitado(c *gin.Context) {
	var input struct {
		Nombre     string `json:"nombre_invitado"`
		Cedula     string `json:"cedula_invitado"`
		Telefono   string `json:"telefono_invitado"`
		ServicioID int    `json:"servicio_id"`
		FechaHora  string `json:"fecha_hora"` // ahora como string para mayor control
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		fmt.Println("❌ Error al bindear JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	if input.Nombre == "" || input.Cedula == "" || input.Telefono == "" || input.FechaHora == "" || input.ServicioID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Todos los campos son obligatorios"})
		return
	}

	// Parsear fecha_hora
	fechaHora, err := time.Parse("2006-01-02T15:04:05", input.FechaHora)
	if err != nil {
		fmt.Println("❌ Error al parsear fecha_hora:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido. Use: YYYY-MM-DDTHH:MM:SS"})
		return
	}

	if fechaHora.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La fecha debe ser futura"})
		return
	}

	// Verificar que el servicio existe
	var exists int
	err = dto.DB.QueryRow("SELECT COUNT(*) FROM servicios WHERE id = ?", input.ServicioID).Scan(&exists)
	if err != nil || exists == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El servicio no existe"})
		return
	}

	// Insertar la cita
	_, err = dto.DB.Exec(`
		INSERT INTO citas (servicio_id, fecha_hora, estado, nombre_invitado, cedula_invitado, telefono_invitado)
		VALUES (?, ?, 'pendiente', ?, ?, ?)`,
		input.ServicioID, fechaHora, input.Nombre, input.Cedula, input.Telefono)

	if err != nil {
		fmt.Println("❌ Error al insertar en la base de datos:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al registrar la cita"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mensaje": "Cita registrada exitosamente como invitado"})
}

func ListarTodasLasCitas(c *gin.Context) {
	// Verificamos que sea admin
	rol, _ := c.Get("rol")
	if rol != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "No autorizado"})
		return
	}

	var todasLasCitas []map[string]interface{}

	// 1️⃣ Consultamos las citas de usuarios logueados
	queryUsuarios := `
		SELECT c.id, c.servicio_id, c.fecha_hora, c.estado, c.empleado_id,
		       c.creado_en, c.actualizado_en, s.nombre AS nombre_servicio,
		       s.precio, u.cedula, u.nombre AS nombre_cliente
		FROM citas c
		JOIN servicios s ON c.servicio_id = s.id
		JOIN usuarios u ON c.usuario_id = u.id
		ORDER BY c.fecha_hora DESC
	`

	rowsUsuarios, err := dto.DB.Query(queryUsuarios)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener citas de usuarios"})
		return
	}
	defer rowsUsuarios.Close()

	for rowsUsuarios.Next() {
		var (
			id, servicioID                        int
			empleadoID                            sql.NullInt32
			fechaHora, estado                     string
			creadoEn, actualizadoEn               sql.NullTime
			nombreServicio, nombreCliente, cedula string
			precio                                float64
		)

		err := rowsUsuarios.Scan(&id, &servicioID, &fechaHora, &estado, &empleadoID,
			&creadoEn, &actualizadoEn, &nombreServicio, &precio, &cedula, &nombreCliente)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar citas de usuarios"})
			return
		}

		cita := map[string]interface{}{
			"id":         id,
			"fecha_hora": fechaHora,
			"estado":     estado,
			"servicio": map[string]interface{}{
				"id":     servicioID,
				"nombre": nombreServicio,
				"precio": precio,
			},
			"cliente": map[string]interface{}{
				"nombre": nombreCliente,
				"cedula": cedula,
			},
			"creado_en":      creadoEn.Time,
			"actualizado_en": actualizadoEn.Time,
			"empleado_id":    nil,
			"tipo":           "usuario", // <-- identificador de tipo
		}

		if empleadoID.Valid {
			cita["empleado_id"] = empleadoID.Int32
		}

		todasLasCitas = append(todasLasCitas, cita)
	}

	// 2️⃣ Consultamos las citas de invitados
	queryInvitados := `
		SELECT c.id, c.fecha_hora, c.estado, c.servicio_id, 
		       c.nombre_invitado, c.cedula_invitado, c.telefono_invitado
		FROM citas c
		WHERE c.usuario_id IS NULL
		ORDER BY c.fecha_hora DESC
	`

	rowsInvitados, err := dto.DB.Query(queryInvitados)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener citas de invitados"})
		return
	}
	defer rowsInvitados.Close()

	for rowsInvitados.Next() {
		var (
			id, servicioID                 int
			fechaHora, estado              string
			nombre, cedulaInv, telefonoInv string
		)

		err := rowsInvitados.Scan(&id, &fechaHora, &estado, &servicioID, &nombre, &cedulaInv, &telefonoInv)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar citas de invitados"})
			return
		}

		cita := map[string]interface{}{
			"id":                id,
			"fecha_hora":        fechaHora,
			"estado":            estado,
			"servicio_id":       servicioID,
			"nombre_invitado":   nombre,
			"cedula_invitado":   cedulaInv,
			"telefono_invitado": telefonoInv,
			"tipo":              "invitado", // <-- identificador de tipo
		}

		todasLasCitas = append(todasLasCitas, cita)
	}

	c.JSON(http.StatusOK, todasLasCitas)
}

func CancelarCitaConMotivo(c *gin.Context) {
	id := c.Param("id")

	// Leer el motivo del cuerpo del request
	var datos struct {
		Motivo string `json:"motivo"`
	}
	if err := c.ShouldBindJSON(&datos); err != nil || datos.Motivo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Motivo de cancelación es requerido"})
		return
	}

	// Verificar que la cita exista y esté pendiente o confirmada
	var estado string
	err := dto.DB.QueryRow("SELECT estado FROM citas WHERE id = ?", id).Scan(&estado)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cita no encontrada"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar cita"})
		return
	}
	if estado != "pendiente" && estado != "confirmada" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solo se pueden cancelar citas pendientes o confirmadas"})
		return
	}

	// Cancelar cita y registrar el motivo
	_, err = dto.DB.Exec(
		"UPDATE citas SET estado = ?, cancelacion_motivo = ?, actualizado_en = NOW() WHERE id = ?",
		"cancelada", datos.Motivo, id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cancelar la cita"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mensaje": "Cita cancelada correctamente",
		"motivo":  datos.Motivo,
	})
}

func ObtenerUltimaCitaInvitado(c *gin.Context) {
	cedula := c.Param("cedula")

	var cita dto.Cita
	query := `
        SELECT id, servicio_id, fecha_hora, estado, nombre_invitado, telefono_invitado
        FROM citas
        WHERE cedula_invitado = ?
        ORDER BY fecha_hora DESC
        LIMIT 1`

	err := dto.DB.QueryRow(query, cedula).Scan(&cita.ID, &cita.ServicioID, &cita.FechaHora, &cita.Estado, &cita.NombreInvitado, &cita.TelefonoInvitado)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró ninguna cita para esta cédula"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la base de datos"})
		return
	}

	c.JSON(http.StatusOK, cita)
}

func UltimaCitaCliente(c *gin.Context) {
	usuarioID, _ := c.Get("usuarioID")

	query := `
        SELECT c.id, c.servicio_id, c.fecha_hora, c.estado, s.nombre, s.precio
        FROM citas c
        JOIN servicios s ON c.servicio_id = s.id
        WHERE c.usuario_id = ?
        ORDER BY c.fecha_hora DESC
        LIMIT 1`

	var (
		citaID, servicioID int
		fechaHora, estado  string
		nombreServicio     string
		precio             float64
	)

	err := dto.DB.QueryRow(query, usuarioID).Scan(&citaID, &servicioID, &fechaHora, &estado, &nombreServicio, &precio)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No tiene citas registradas"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar cita"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         citaID,
		"fecha_hora": fechaHora,
		"estado":     estado,
		"servicio": gin.H{
			"id":     servicioID,
			"nombre": nombreServicio,
			"precio": precio,
		},
	})

}

func ObtenerCitasPorCedulaInvitado(c *gin.Context) {
	cedula := c.Param("cedula")

	query := `
		SELECT c.id, c.fecha_hora, c.estado, c.servicio_id, 
		       c.nombre_invitado, c.cedula_invitado, c.telefono_invitado
		FROM citas c
		WHERE c.cedula_invitado = ?
		ORDER BY c.fecha_hora DESC
	`

	rows, err := dto.DB.Query(query, cedula)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar citas"})
		return
	}
	defer rows.Close()

	var citas []map[string]interface{}

	for rows.Next() {
		var (
			id, servicioID    int
			fechaHora, estado string
			nombre, cedulaInv string
			telefonoInv       string
		)

		err := rows.Scan(&id, &fechaHora, &estado, &servicioID, &nombre, &cedulaInv, &telefonoInv)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer citas"})
			return
		}

		citas = append(citas, map[string]interface{}{
			"id":                id,
			"fecha_hora":        fechaHora,
			"estado":            estado,
			"servicio_id":       servicioID,
			"nombre_invitado":   nombre,
			"cedula_invitado":   cedulaInv,
			"telefono_invitado": telefonoInv,
		})
	}

	c.JSON(http.StatusOK, citas)
}
