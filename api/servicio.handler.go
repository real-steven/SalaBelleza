package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"restapi/dto"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ServicioInput struct {
	Nombre      string  `json:"nombre"`
	Descripcion string  `json:"descripcion"`
	Precio      float64 `json:"precio"`
}

func CrearServicio(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo administradores pueden crear servicios"})
		return
	}

	var input ServicioInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	if input.Nombre == "" || input.Precio <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nombre y precio válidos son obligatorios"})
		return
	}

	_, err := dto.DB.Exec("INSERT INTO servicios (nombre, descripcion, precio) VALUES (?, ?, ?)",
		input.Nombre, input.Descripcion, input.Precio)

	if err != nil {
		fmt.Println("❌ Error al insertar servicio:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear servicio"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mensaje": "Servicio creado exitosamente"})
}

func ObtenerServicio(c *gin.Context) {
	id := c.Param("id")

	var servicio struct {
		ID          int     `json:"id"`
		Nombre      string  `json:"nombre"`
		Descripcion string  `json:"descripcion"`
		Precio      float64 `json:"precio"`
	}

	err := dto.DB.QueryRow("SELECT id, nombre, descripcion, precio FROM servicios WHERE id = ?", id).
		Scan(&servicio.ID, &servicio.Nombre, &servicio.Descripcion, &servicio.Precio)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Servicio no encontrado"})
			return
		}
		fmt.Println("❌ Error al obtener servicio:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener servicio"})
		return
	}

	c.JSON(http.StatusOK, servicio)
}

func ActualizarServicio(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo administradores pueden actualizar servicios"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	// Verificar si hay citas asociadas a este servicio
	var count int
	err = dto.DB.QueryRow("SELECT COUNT(*) FROM citas WHERE servicio_id = ?", id).Scan(&count)
	if err != nil {
		fmt.Println("❌ Error al verificar citas relacionadas:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al verificar dependencias"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "No se puede actualizar el servicio porque está vinculado a citas existentes"})
		return
	}

	var input ServicioInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	_, err = dto.DB.Exec("UPDATE servicios SET nombre=?, descripcion=?, precio=? WHERE id=?",
		input.Nombre, input.Descripcion, input.Precio, id)

	if err != nil {
		fmt.Println("❌ Error al actualizar servicio:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar servicio"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Servicio actualizado correctamente"})
}

func EliminarServicio(c *gin.Context) {
	rol, _ := c.Get("rol")
	if rol != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo administradores pueden eliminar servicios"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	fmt.Println("🗑️ Intentando eliminar servicio con ID:", id)

	_, err = dto.DB.Exec("DELETE FROM servicios WHERE id=?", id)
	if err != nil {
		fmt.Println("❌ Error al eliminar servicio:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar el servicio. Verifica si está en uso."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Servicio eliminado correctamente"})
}

func ListarServicios(c *gin.Context) {
	rows, err := dto.DB.Query("SELECT id, nombre, descripcion, precio FROM servicios")
	if err != nil {
		fmt.Println("❌ Error al obtener lista de servicios:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener servicios"})
		return
	}
	defer rows.Close()

	var servicios []map[string]interface{}
	for rows.Next() {
		var (
			id          int
			nombre      string
			descripcion string
			precio      float64
		)

		if err := rows.Scan(&id, &nombre, &descripcion, &precio); err == nil {
			servicio := map[string]interface{}{
				"id":          id,
				"nombre":      nombre,
				"descripcion": descripcion,
				"precio":      precio,
			}
			servicios = append(servicios, servicio)
		}
	}

	c.JSON(http.StatusOK, servicios)
}
