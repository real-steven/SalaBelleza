// Steven: Definición de estructuras de datos principales: Usuario, Cita, Servicio, etc.

package dto

import (
	"database/sql"
)

type Usuario struct {
	ID            int32        `json:"id"`
	Nombre        string       `json:"nombre"`
	Correo        string       `json:"correo"`
	Cedula        string       `json:"cedula"`
	Contrasena    string       `json:"contrasena"`
	Rol           string       `json:"rol"` // cliente, empleado o administrador
	CreadoEn      sql.NullTime `json:"creado_en"`
	ActualizadoEn sql.NullTime `json:"actualizado_en"`
}

type Cita struct {
	ID                int32          `json:"id"`
	UsuarioID         int32          `json:"usuario_id"`
	EmpleadoID        int32          `json:"empleado_id"`
	ServicioID        int32          `json:"servicio_id"`
	FechaHora         string         `json:"fecha_hora"`
	NombreInvitado    string         `json:"nombre_invitado"`
	TelefonoInvitado  string         `json:"telefono_invitado"`
	Estado            string         `json:"estado"` // pendiente, confirmada, rechazada, cancelada
	CreadoEn          sql.NullTime   `json:"creado_en"`
	ActualizadoEn     sql.NullTime   `json:"actualizado_en"`
	CancelacionMotivo sql.NullString `json:"cancelacion_motivo"` // motivo de cancelación si aplica
}

type Servicio struct {
	ID            int32          `json:"id"`
	Nombre        string         `json:"nombre"`
	Descripcion   sql.NullString `json:"descripcion"`
	Precio        float64        `json:"precio"`
	CreadoEn      sql.NullTime   `json:"creado_en"`
	ActualizadoEn sql.NullTime   `json:"actualizado_en"`
}

type Producto struct {
	ID                 int32        `json:"id"`
	Nombre             string       `json:"nombre"`
	Descripcion        string       `json:"descripcion"`
	Precio             float64      `json:"precio"`
	Imagen             string       `json:"imagen"`
	CantidadDisponible int32        `json:"cantidad_disponible"`
	CreadoEn           sql.NullTime `json:"creado_en"`
	ActualizadoEn      sql.NullTime `json:"actualizado_en"`
}
