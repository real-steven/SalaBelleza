// Steven: Manejador de usuarios (registro, login, gestión de perfiles, crear usuarios siendo admin).

package api

import (
	"fmt"
	"net/http"
	"restapi/dto"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

var secretKey = []byte("clave_secreta_super_segura")

// Registro de usuarios (cliente por defecto)
func RegistrarUsuario(c *gin.Context) {
	var usuario struct {
		Nombre     string `json:"nombre"`
		Correo     string `json:"correo"`
		Cedula     string `json:"cedula"`
		Contrasena string `json:"contrasena"`
	}

	if err := c.ShouldBindJSON(&usuario); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(usuario.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al encriptar contraseña"})
		return
	}

	_, err = dto.DB.Exec("INSERT INTO usuarios(nombre, correo, cedula, contrasena, rol) VALUES(?, ?, ?, ?, ?)",
		usuario.Nombre, usuario.Correo, usuario.Cedula, string(hashedPassword), "cliente")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al registrar usuario"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mensaje": "Usuario registrado correctamente"})
}

// Login de usuario (todos los roles)
func LoginUsuario(c *gin.Context) {
	var input struct {
		Correo     string `json:"correo"`
		Contrasena string `json:"contrasena"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var usuario dto.Usuario
	err := dto.DB.QueryRow("SELECT id, nombre, contrasena, rol FROM usuarios WHERE correo=?", input.Correo).
		Scan(&usuario.ID, &usuario.Nombre, &usuario.Contrasena, &usuario.Rol)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Correo o contraseña incorrectos"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(usuario.Contrasena), []byte(input.Contrasena))
	if err != nil {
		fmt.Println("🔴 Error al comparar bcrypt:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Correo o contraseña incorrectos"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":     usuario.ID,
		"nombre": usuario.Nombre,
		"rol":    usuario.Rol,
		"exp":    time.Now().Add(99999 * time.Minute).Unix(), //Para modificar el tiempo (para efectos de prueba se pondra mucho)
	})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"usuario": gin.H{
			"id":     usuario.ID,
			"nombre": usuario.Nombre,
			"rol":    usuario.Rol,
		},
	})

}

// Registro manual de usuarios (clientes o empleados) por parte de un administrador
func RegistrarUsuarioComoAdmin(c *gin.Context) {
	rol, existe := c.Get("rol")
	if !existe || rol != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo administradores pueden registrar usuarios manualmente"})
		return
	}

	var input struct {
		Nombre     string `json:"nombre"`
		Correo     string `json:"correo"`
		Cedula     string `json:"cedula"`
		Contrasena string `json:"contrasena"`
		Rol        string `json:"rol"` // cliente, empleado, admin
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	if input.Rol != "cliente" && input.Rol != "empleado" && input.Rol != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rol inválido"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al encriptar la contraseña"})
		return
	}

	_, err = dto.DB.Exec(`
		INSERT INTO usuarios (nombre, correo, cedula, contrasena, rol)
		VALUES (?, ?, ?, ?, ?)`,
		input.Nombre, input.Correo, input.Cedula, string(hashedPassword), input.Rol)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el usuario"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mensaje": "Usuario creado correctamente"})
}

// Ver el perfil del usuario autenticado
func VerMiPerfil(c *gin.Context) {
	usuarioID, _ := c.Get("usuarioID")

	var usuario struct {
		ID     int32  `json:"id"`
		Nombre string `json:"nombre"`
		Correo string `json:"correo"`
		Cedula string `json:"cedula"`
		Rol    string `json:"rol"`
	}

	err := dto.DB.QueryRow("SELECT id, nombre, correo, cedula, rol FROM usuarios WHERE id = ?", usuarioID).
		Scan(&usuario.ID, &usuario.Nombre, &usuario.Correo, &usuario.Cedula, &usuario.Rol)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo obtener el perfil"})
		return
	}

	c.JSON(http.StatusOK, usuario)
}
