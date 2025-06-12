package api

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"restapi/dto"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GET /productos
func ListarProductos(c *gin.Context) {
	busqueda := strings.ToLower(c.Query("buscar"))

	query := "SELECT id, nombre, descripcion, precio, imagen, cantidad_disponible FROM productos"
	var args []interface{}

	if busqueda != "" {
		query += " WHERE LOWER(nombre) LIKE ?"
		args = append(args, "%"+busqueda+"%")
	}

	rows, err := dto.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener productos"})
		return
	}
	defer rows.Close()

	var productos []dto.Producto
	for rows.Next() {
		var p dto.Producto
		var imagen sql.NullString
		err := rows.Scan(&p.ID, &p.Nombre, &p.Descripcion, &p.Precio, &imagen, &p.CantidadDisponible)
		if err == nil {
			if imagen.Valid {
				p.Imagen = imagen.String
			} else {
				p.Imagen = ""
			}
			productos = append(productos, p)
		}
	}
	c.JSON(http.StatusOK, productos)
}

// GET /productos/:id
func ObtenerProducto(c *gin.Context) {
	id := c.Param("id")

	var p dto.Producto
	var imagen sql.NullString
	err := dto.DB.QueryRow("SELECT id, nombre, descripcion, precio, imagen, cantidad_disponible FROM productos WHERE id = ?", id).
		Scan(&p.ID, &p.Nombre, &p.Descripcion, &p.Precio, &imagen, &p.CantidadDisponible)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Producto no encontrado"})
		return
	}

	if imagen.Valid {
		p.Imagen = imagen.String
	} else {
		p.Imagen = ""
	}

	c.JSON(http.StatusOK, p)
}

// POST /productos
func CrearProducto(c *gin.Context) {
	nombre := c.PostForm("nombre")
	descripcion := c.PostForm("descripcion")
	precioStr := c.PostForm("precio")
	cantidadStr := c.PostForm("cantidad")

	precio, err := strconv.ParseFloat(precioStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Precio inválido"})
		return
	}

	cantidad, err := strconv.Atoi(cantidadStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cantidad inválida"})
		return
	}

	file, err := c.FormFile("imagen")
	var rutaImagen string
	if err == nil {
		filename := strconv.FormatInt(time.Now().UnixNano(), 10) + filepath.Ext(file.Filename)
		rutaImagen = "recursos/" + filename
		err = c.SaveUploadedFile(file, rutaImagen)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar la imagen"})
			return
		}
	}

	query := "INSERT INTO productos (nombre, descripcion, precio, imagen, cantidad_disponible) VALUES (?, ?, ?, ?, ?)"
	result, err := dto.DB.Exec(query, nombre, descripcion, precio, rutaImagen, cantidad)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el producto"})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id, "nombre": nombre, "descripcion": descripcion, "precio": precio, "imagen": rutaImagen, "cantidad_disponible": cantidad})
}

// PUT /productos/:id
func ActualizarProducto(c *gin.Context) {
	id := c.Param("id")
	nombre := c.PostForm("nombre")
	descripcion := c.PostForm("descripcion")
	precioStr := c.PostForm("precio")
	cantidadStr := c.PostForm("cantidad")

	precio, err := strconv.ParseFloat(precioStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Precio inválido"})
		return
	}

	cantidad, err := strconv.Atoi(cantidadStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cantidad inválida"})
		return
	}

	file, err := c.FormFile("imagen")
	rutaImagen := ""
	if err == nil {
		filename := strconv.FormatInt(time.Now().UnixNano(), 10) + filepath.Ext(file.Filename)
		rutaImagen = "recursos/" + filename
		err = c.SaveUploadedFile(file, rutaImagen)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar la imagen"})
			return
		}
	}

	var query string
	var args []interface{}
	if rutaImagen != "" {
		query = `UPDATE productos SET nombre = ?, descripcion = ?, precio = ?, imagen = ?, cantidad_disponible = ? WHERE id = ?`
		args = []interface{}{nombre, descripcion, precio, rutaImagen, cantidad, id}
	} else {
		query = `UPDATE productos SET nombre = ?, descripcion = ?, precio = ?, cantidad_disponible = ? WHERE id = ?`
		args = []interface{}{nombre, descripcion, precio, cantidad, id}
	}

	_, err = dto.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar producto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Producto actualizado correctamente"})
}

// DELETE /productos/:id
func EliminarProducto(c *gin.Context) {
	id := c.Param("id")

	var imagen sql.NullString
	err := dto.DB.QueryRow("SELECT imagen FROM productos WHERE id = ?", id).Scan(&imagen)
	if err == nil && imagen.Valid {
		_ = os.Remove(imagen.String) // eliminar archivo si existe
	}

	_, err = dto.DB.Exec("DELETE FROM productos WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el producto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Producto eliminado exitosamente"})
}
