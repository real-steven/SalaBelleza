// Archivo principal para iniciar la conexión a la base de datos y el servidor.

package main

import (
	"restapi/api"
	"restapi/dto"
)

func main() {
	dto.ConectarBaseDatos()
	router := api.InicializarServidor()
	router.Run(":8080")
}
