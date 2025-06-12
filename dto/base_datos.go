// Configuración de conexión a la base de datos.

package dto

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConectarBaseDatos() {
	var err error
	DB, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/salonbelleza?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		log.Fatal("Error al conectar la base de datos:", err)
	}
	err = DB.Ping()
	if err != nil {
		log.Fatal("No se pudo hacer ping a la base de datos:", err)
	}
}
