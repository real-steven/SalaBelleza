package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func Autenticar() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		// Extra: segunda búsqueda por si viene el header raro (por seguridad adicional)
		if tokenString == "" {
			tokenString = c.Request.Header.Get("Authorization")
		}

		// Normalizamos: eliminamos el "Bearer "
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token requerido"})
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			id, _ := claims["id"].(float64)
			rol, _ := claims["rol"].(string)

			c.Set("usuarioID", int(id))
			c.Set("rol", rol)
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			return
		}
	}
}
