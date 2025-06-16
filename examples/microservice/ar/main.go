package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	qf "github.com/ha1tch/queryfy"
	"github.com/ha1tch/queryfy/builders"
	"github.com/ha1tch/queryfy/builders/transformers"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Usuario representa el modelo de usuario argentino
type Usuario struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DNI         string    `json:"dni"`
	CUIT        string    `json:"cuit,omitempty"`
	Nombre      string    `json:"nombre"`
	Apellido    string    `json:"apellido"`
	Telefono    string    `json:"telefono,omitempty"`
	Celular     string    `json:"celular"`
	FechaNac    string    `json:"fechaNacimiento"`
	Provincia   string    `json:"provincia"`
	Localidad   string    `json:"localidad"`
	CodigoPost  string    `json:"codigoPostal"`
	Direccion   string    `json:"direccion"`
	CBU         string    `json:"cbu,omitempty"`
	Rol         string    `json:"rol"`
	Estado      string    `json:"estado"`
	CreadoEn    time.Time `json:"creadoEn"`
	Actualizado time.Time `json:"actualizado"`
}

// Provincias argentinas válidas
var provinciasArgentinas = []string{
	"Buenos Aires", "CABA", "Catamarca", "Chaco", "Chubut", "Córdoba",
	"Corrientes", "Entre Ríos", "Formosa", "Jujuy", "La Pampa", "La Rioja",
	"Mendoza", "Misiones", "Neuquén", "Río Negro", "Salta", "San Juan",
	"San Luis", "Santa Cruz", "Santa Fe", "Santiago del Estero",
	"Tierra del Fuego", "Tucumán",
}

// ServicioUsuarios maneja la lógica de negocio
type ServicioUsuarios struct {
	mu       sync.RWMutex
	usuarios map[string]Usuario
}

// NuevoServicioUsuarios crea un nuevo servicio
func NuevoServicioUsuarios() *ServicioUsuarios {
	return &ServicioUsuarios{
		usuarios: make(map[string]Usuario),
	}
}

// Esquemas define todos los esquemas de validación
type Esquemas struct {
	CrearUsuario      qf.Schema
	ActualizarUsuario qf.Schema
	ConsultarUsuarios qf.Schema
}

// Transformadores personalizados para Argentina

// normalizarDNI quita puntos y espacios del DNI
var normalizarDNI = func(value interface{}) (interface{}, error) {
	dni := value.(string)
	// Quitar puntos, espacios y guiones
	dni = strings.ReplaceAll(dni, ".", "")
	dni = strings.ReplaceAll(dni, " ", "")
	dni = strings.ReplaceAll(dni, "-", "")
	return dni, nil
}

// normalizarCUIT formatea CUIT/CUIL correctamente
var normalizarCUIT = func(value interface{}) (interface{}, error) {
	cuit := value.(string)
	// Quitar todo excepto números
	re := regexp.MustCompile(`[^0-9]`)
	cuit = re.ReplaceAllString(cuit, "")
	
	if len(cuit) == 11 {
		// Formatear como XX-XXXXXXXX-X
		return fmt.Sprintf("%s-%s-%s", cuit[:2], cuit[2:10], cuit[10:]), nil
	}
	return cuit, nil
}

// normalizarTelefonoAR normaliza teléfonos argentinos
var normalizarTelefonoAR = func(value interface{}) (interface{}, error) {
	tel := value.(string)
	// Quitar todo excepto números
	re := regexp.MustCompile(`[^0-9]`)
	tel = re.ReplaceAllString(tel, "")
	
	// Quitar 0 inicial y 15 para celulares viejos
	if strings.HasPrefix(tel, "0") {
		tel = tel[1:]
	}
	if strings.HasPrefix(tel, "15") && len(tel) > 10 {
		tel = tel[2:]
	}
	
	// Si no tiene código de país, agregar +54
	if !strings.HasPrefix(tel, "54") {
		tel = "54" + tel
	}
	
	return "+" + tel, nil
}

// normalizarCBU formatea CBU correctamente
var normalizarCBU = func(value interface{}) (interface{}, error) {
	cbu := value.(string)
	// Quitar espacios y guiones
	cbu = strings.ReplaceAll(cbu, " ", "")
	cbu = strings.ReplaceAll(cbu, "-", "")
	return cbu, nil
}

// normalizarCodigoPostal normaliza códigos postales argentinos
var normalizarCodigoPostal = func(value interface{}) (interface{}, error) {
	cp := value.(string)
	cp = strings.ToUpper(strings.TrimSpace(cp))
	return cp, nil
}

// Validadores personalizados

// validarDNI verifica que el DNI sea válido
var validarDNI = func(value interface{}) error {
	dni := value.(string)
	// DNI debe ser numérico y tener entre 7 y 8 dígitos
	matched, _ := regexp.MatchString(`^\d{7,8}$`, dni)
	if !matched {
		return fmt.Errorf("DNI debe tener entre 7 y 8 dígitos")
	}
	
	// Verificar rango válido
	dniNum, _ := strconv.Atoi(dni)
	if dniNum < 1000000 || dniNum > 99999999 {
		return fmt.Errorf("DNI fuera de rango válido")
	}
	
	return nil
}

// validarCUIT verifica CUIT/CUIL con dígito verificador
var validarCUIT = func(value interface{}) error {
	cuit := value.(string)
	// Quitar formato
	re := regexp.MustCompile(`[^0-9]`)
	cuit = re.ReplaceAllString(cuit, "")
	
	if len(cuit) != 11 {
		return fmt.Errorf("CUIT debe tener 11 dígitos")
	}
	
	// Validar prefijos válidos
	prefijo := cuit[:2]
	prefijosValidos := []string{"20", "23", "24", "27", "30", "33", "34"}
	validPrefix := false
	for _, p := range prefijosValidos {
		if prefijo == p {
			validPrefix = true
			break
		}
	}
	if !validPrefix {
		return fmt.Errorf("prefijo de CUIT inválido")
	}
	
	// Validar dígito verificador
	if !validarDigitoVerificadorCUIT(cuit) {
		return fmt.Errorf("dígito verificador de CUIT inválido")
	}
	
	return nil
}

// validarDigitoVerificadorCUIT implementa el algoritmo de validación
func validarDigitoVerificadorCUIT(cuit string) bool {
	if len(cuit) != 11 {
		return false
	}
	
	multiplicadores := []int{5, 4, 3, 2, 7, 6, 5, 4, 3, 2}
	suma := 0
	
	for i := 0; i < 10; i++ {
		digit, _ := strconv.Atoi(string(cuit[i]))
		suma += digit * multiplicadores[i]
	}
	
	resto := suma % 11
	digitoCalculado := 11 - resto
	
	if digitoCalculado == 11 {
		digitoCalculado = 0
	} else if digitoCalculado == 10 {
		digitoCalculado = 9
	}
	
	digitoVerificador, _ := strconv.Atoi(string(cuit[10]))
	return digitoCalculado == digitoVerificador
}

// validarCBU verifica que el CBU sea válido
var validarCBU = func(value interface{}) error {
	cbu := value.(string)
	
	if len(cbu) != 22 {
		return fmt.Errorf("CBU debe tener 22 dígitos")
	}
	
	matched, _ := regexp.MatchString(`^\d{22}$`, cbu)
	if !matched {
		return fmt.Errorf("CBU debe contener solo números")
	}
	
	return nil
}

// validarCodigoPostal verifica formato de código postal argentino
var validarCodigoPostal = func(value interface{}) error {
	cp := value.(string)
	// Aceptar formato numérico (1425) o con letra inicial (C1425, B1900)
	if len(cp) < 4 || len(cp) > 8 {
		return fmt.Errorf("código postal debe tener entre 4 y 8 caracteres")
	}
	
	// Verificar que tenga el formato correcto
	matched, _ := regexp.MatchString(`^[A-Z]?\d{4}[A-Z]{0,3}$`, cp)
	if !matched {
		return fmt.Errorf("formato de código postal inválido")
	}
	
	return nil
}

// InicializarEsquemas crea todos los esquemas de validación
func InicializarEsquemas() *Esquemas {
	// Esquema para crear usuario
	crearUsuarioSchema := builders.Object().
		Field("email",
			builders.Transform(
				builders.String().
					Email().
					Required(),
			).Add(transformers.Trim()).
			Add(transformers.Lowercase())).
		Field("dni",
			builders.Transform(
				builders.String().
					Required().
					Custom(validarDNI),
			).Add(transformers.Trim()).
			Add(normalizarDNI)).
		Field("cuit",
			builders.Transform(
				builders.String().
					Optional().
					Custom(validarCUIT),
			).Add(transformers.Trim()).
			Add(normalizarCUIT)).
		Field("nombre",
			builders.Transform(
				builders.String().
					MinLength(2).
					MaxLength(50).
					Required(),
			).Add(transformers.Trim()).
			Add(transformers.NormalizeWhitespace())).
		Field("apellido",
			builders.Transform(
				builders.String().
					MinLength(2).
					MaxLength(50).
					Required(),
			).Add(transformers.Trim()).
			Add(transformers.NormalizeWhitespace())).
		Field("telefono",
			builders.Transform(
				builders.String().
					Optional(),
			).Add(transformers.Trim()).
			Add(normalizarTelefonoAR)).
		Field("celular",
			builders.Transform(
				builders.String().
					Required(),
			).Add(transformers.Trim()).
			Add(normalizarTelefonoAR)).
		Field("fechaNacimiento",
			builders.DateTime().
				DateOnly().
				Past().
				Age(18, 120).
				Required()).
		Field("provincia",
			builders.String().
				Enum(provinciasArgentinas...).
				Required()).
		Field("localidad",
			builders.String().
				MinLength(2).
				MaxLength(100).
				Required()).
		Field("codigoPostal",
			builders.Transform(
				builders.String().
					MinLength(4).
					MaxLength(8).
					Required().
					Custom(validarCodigoPostal),
			).Add(normalizarCodigoPostal)).
		Field("direccion",
			builders.String().
				MinLength(5).
				MaxLength(200).
				Required()).
		Field("cbu",
			builders.Transform(
				builders.String().
					Optional().
					Custom(validarCBU),
			).Add(transformers.Trim()).
			Add(normalizarCBU)).
		Field("password",
			builders.String().
				MinLength(8).
				MaxLength(72).
				Pattern(`[A-Z]`).     // Al menos una mayúscula
				Pattern(`[a-z]`).     // Al menos una minúscula
				Pattern(`[0-9]`).     // Al menos un dígito
				Required()).
		Field("rol",
			builders.String().
				Enum("usuario", "admin", "moderador").
				Optional()).
		Custom(func(value interface{}) error {
			// Validación cruzada: si tiene CUIT, debe ser mayor de edad
			data := value.(map[string]interface{})
			if cuit, hasCUIT := data["cuit"].(string); hasCUIT && cuit != "" {
				// El validador de edad ya se encarga, pero podríamos agregar más lógica aquí
			}
			return nil
		})

	// Esquema para actualizar usuario (todos los campos opcionales excepto password)
	actualizarUsuarioSchema := builders.Object().
		Field("email",
			builders.Transform(
				builders.String().
					Email().
					Optional(),
			).Add(transformers.Trim()).
			Add(transformers.Lowercase())).
		Field("telefono",
			builders.Transform(
				builders.String().
					Optional(),
			).Add(transformers.Trim()).
			Add(normalizarTelefonoAR)).
		Field("celular",
			builders.Transform(
				builders.String().
					Optional(),
			).Add(transformers.Trim()).
			Add(normalizarTelefonoAR)).
		Field("provincia",
			builders.String().
				Enum(provinciasArgentinas...).
				Optional()).
		Field("localidad",
			builders.String().
				MinLength(2).
				MaxLength(100).
				Optional()).
		Field("codigoPostal",
			builders.Transform(
				builders.String().
					MinLength(4).
					MaxLength(8).
					Optional().
					Custom(validarCodigoPostal),
			).Add(normalizarCodigoPostal)).
		Field("direccion",
			builders.String().
				MinLength(5).
				MaxLength(200).
				Optional()).
		Field("cbu",
			builders.Transform(
				builders.String().
					Optional().
					Custom(validarCBU),
			).Add(transformers.Trim()).
			Add(normalizarCBU)).
		Field("rol",
			builders.String().
				Enum("usuario", "admin", "moderador").
				Optional()).
		Field("estado",
			builders.String().
				Enum("activo", "suspendido", "eliminado").
				Optional())

	// Esquema para consultas
	consultarUsuariosSchema := builders.Object().
		Field("email", builders.String().Email().Optional()).
		Field("dni", builders.String().Optional()).
		Field("cuit", builders.String().Optional()).
		Field("provincia", builders.String().Enum(provinciasArgentinas...).Optional()).
		Field("rol", builders.String().Enum("usuario", "admin", "moderador").Optional()).
		Field("estado", builders.String().Enum("activo", "suspendido", "eliminado").Optional()).
		Field("limite", builders.Number().Min(1).Max(100).Optional()).
		Field("offset", builders.Number().Min(0).Optional())

	return &Esquemas{
		CrearUsuario:      crearUsuarioSchema,
		ActualizarUsuario: actualizarUsuarioSchema,
		ConsultarUsuarios: consultarUsuariosSchema,
	}
}

// Función auxiliar para extraer datos transformados
func extraerDatosTransformados(data map[string]interface{}, ctx *qf.ValidationContext) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = v
	}

	// Aplicar transformaciones basadas en el contexto
	for _, transform := range ctx.Transformations() {
		if transform.Path != "" {
			result[transform.Path] = transform.Result
		}
	}

	return result
}

// Operaciones CRUD

// Crear agrega un nuevo usuario
func (s *ServicioUsuarios) Crear(data map[string]interface{}) (*Usuario, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generar ID y timestamps
	usuario := Usuario{
		ID:          uuid.New().String(),
		Email:       data["email"].(string),
		DNI:         data["dni"].(string),
		Nombre:      data["nombre"].(string),
		Apellido:    data["apellido"].(string),
		Celular:     data["celular"].(string),
		FechaNac:    data["fechaNacimiento"].(string),
		Provincia:   data["provincia"].(string),
		Localidad:   data["localidad"].(string),
		CodigoPost:  data["codigoPostal"].(string),
		Direccion:   data["direccion"].(string),
		Rol:         "usuario", // Rol por defecto
		Estado:      "activo",
		CreadoEn:    time.Now(),
		Actualizado: time.Now(),
	}

	// Campos opcionales
	if cuit, ok := data["cuit"].(string); ok && cuit != "" {
		usuario.CUIT = cuit
	}
	if telefono, ok := data["telefono"].(string); ok && telefono != "" {
		usuario.Telefono = telefono
	}
	if cbu, ok := data["cbu"].(string); ok && cbu != "" {
		usuario.CBU = cbu
	}
	if rol, ok := data["rol"].(string); ok {
		usuario.Rol = rol
	}

	// Verificar duplicados
	for _, existente := range s.usuarios {
		if existente.Email == usuario.Email {
			return nil, fmt.Errorf("el email ya está registrado")
		}
		if existente.DNI == usuario.DNI {
			return nil, fmt.Errorf("el DNI ya está registrado")
		}
		if usuario.CUIT != "" && existente.CUIT == usuario.CUIT {
			return nil, fmt.Errorf("el CUIT ya está registrado")
		}
	}

	s.usuarios[usuario.ID] = usuario
	return &usuario, nil
}

// Obtener recupera un usuario por ID
func (s *ServicioUsuarios) Obtener(id string) (*Usuario, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	usuario, existe := s.usuarios[id]
	if !existe {
		return nil, fmt.Errorf("usuario no encontrado")
	}
	return &usuario, nil
}

// Actualizar modifica un usuario existente
func (s *ServicioUsuarios) Actualizar(id string, data map[string]interface{}) (*Usuario, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	usuario, existe := s.usuarios[id]
	if !existe {
		return nil, fmt.Errorf("usuario no encontrado")
	}

	// Actualizar campos si se proporcionan
	if email, ok := data["email"].(string); ok {
		// Verificar duplicado
		for uid, u := range s.usuarios {
			if uid != id && u.Email == email {
				return nil, fmt.Errorf("el email ya está registrado")
			}
		}
		usuario.Email = email
	}

	// Actualizar otros campos opcionales
	if telefono, ok := data["telefono"].(string); ok {
		usuario.Telefono = telefono
	}
	if celular, ok := data["celular"].(string); ok {
		usuario.Celular = celular
	}
	if provincia, ok := data["provincia"].(string); ok {
		usuario.Provincia = provincia
	}
	if localidad, ok := data["localidad"].(string); ok {
		usuario.Localidad = localidad
	}
	if codigoPostal, ok := data["codigoPostal"].(string); ok {
		usuario.CodigoPost = codigoPostal
	}
	if direccion, ok := data["direccion"].(string); ok {
		usuario.Direccion = direccion
	}
	if cbu, ok := data["cbu"].(string); ok {
		usuario.CBU = cbu
	}
	if rol, ok := data["rol"].(string); ok {
		usuario.Rol = rol
	}
	if estado, ok := data["estado"].(string); ok {
		usuario.Estado = estado
	}

	usuario.Actualizado = time.Now()
	s.usuarios[id] = usuario
	return &usuario, nil
}

// Eliminar realiza un borrado lógico
func (s *ServicioUsuarios) Eliminar(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	usuario, existe := s.usuarios[id]
	if !existe {
		return fmt.Errorf("usuario no encontrado")
	}

	usuario.Estado = "eliminado"
	usuario.Actualizado = time.Now()
	s.usuarios[id] = usuario
	return nil
}

// Consultar busca usuarios según criterios
func (s *ServicioUsuarios) Consultar(criterios map[string]interface{}) ([]Usuario, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var resultados []Usuario
	limite := 10
	offset := 0

	// Extraer paginación
	if l, ok := criterios["limite"].(float64); ok {
		limite = int(l)
	}
	if o, ok := criterios["offset"].(float64); ok {
		offset = int(o)
	}

	// Filtrar usuarios
	contador := 0
	for _, usuario := range s.usuarios {
		// Aplicar filtros
		if email, ok := criterios["email"].(string); ok && usuario.Email != email {
			continue
		}
		if dni, ok := criterios["dni"].(string); ok && usuario.DNI != dni {
			continue
		}
		if cuit, ok := criterios["cuit"].(string); ok && usuario.CUIT != cuit {
			continue
		}
		if provincia, ok := criterios["provincia"].(string); ok && usuario.Provincia != provincia {
			continue
		}
		if rol, ok := criterios["rol"].(string); ok && usuario.Rol != rol {
			continue
		}
		if estado, ok := criterios["estado"].(string); ok && usuario.Estado != estado {
			continue
		}

		// Aplicar paginación
		if contador >= offset && len(resultados) < limite {
			resultados = append(resultados, usuario)
		}
		contador++
	}

	return resultados, nil
}

// Manejadores HTTP

type Manejador struct {
	servicio *ServicioUsuarios
	esquemas *Esquemas
}

func NuevoManejador(servicio *ServicioUsuarios, esquemas *Esquemas) *Manejador {
	return &Manejador{
		servicio: servicio,
		esquemas: esquemas,
	}
}

// CrearUsuario maneja POST /usuarios
func (m *Manejador) CrearUsuario(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		responderError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	// Validar usando Queryfy
	ctx := qf.NewValidationContext(qf.Strict)
	if err := m.esquemas.CrearUsuario.Validate(data, ctx); err != nil || ctx.HasErrors() {
		responderErroresValidacion(w, ctx)
		return
	}

	// Extraer datos transformados
	datosTransformados := extraerDatosTransformados(data, ctx)

	// Crear usuario
	usuario, err := m.servicio.Crear(datosTransformados)
	if err != nil {
		responderError(w, http.StatusConflict, err.Error())
		return
	}

	responderJSON(w, http.StatusCreated, usuario)
}

// ObtenerUsuario maneja GET /usuarios/{id}
func (m *Manejador) ObtenerUsuario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	usuario, err := m.servicio.Obtener(id)
	if err != nil {
		responderError(w, http.StatusNotFound, "Usuario no encontrado")
		return
	}

	responderJSON(w, http.StatusOK, usuario)
}

// ActualizarUsuario maneja PUT /usuarios/{id}
func (m *Manejador) ActualizarUsuario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		responderError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	// Validar
	ctx := qf.NewValidationContext(qf.Strict)
	if err := m.esquemas.ActualizarUsuario.Validate(data, ctx); err != nil || ctx.HasErrors() {
		responderErroresValidacion(w, ctx)
		return
	}

	// Extraer datos transformados
	datosTransformados := extraerDatosTransformados(data, ctx)

	// Actualizar usuario
	usuario, err := m.servicio.Actualizar(id, datosTransformados)
	if err != nil {
		if err.Error() == "usuario no encontrado" {
			responderError(w, http.StatusNotFound, err.Error())
		} else {
			responderError(w, http.StatusConflict, err.Error())
		}
		return
	}

	responderJSON(w, http.StatusOK, usuario)
}

// EliminarUsuario maneja DELETE /usuarios/{id}
func (m *Manejador) EliminarUsuario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := m.servicio.Eliminar(id); err != nil {
		responderError(w, http.StatusNotFound, "Usuario no encontrado")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ConsultarUsuarios maneja GET /usuarios con parámetros de consulta
func (m *Manejador) ConsultarUsuarios(w http.ResponseWriter, r *http.Request) {
	// Parsear parámetros de consulta
	queryData := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			// Convertir parámetros numéricos
			if key == "limite" || key == "offset" {
				if num, err := strconv.Atoi(values[0]); err == nil {
					queryData[key] = float64(num)
				}
			} else {
				queryData[key] = values[0]
			}
		}
	}

	// Validar parámetros de consulta
	ctx := qf.NewValidationContext(qf.Strict)
	if err := m.esquemas.ConsultarUsuarios.Validate(queryData, ctx); err != nil || ctx.HasErrors() {
		responderErroresValidacion(w, ctx)
		return
	}

	// Consultar usuarios
	usuarios, err := m.servicio.Consultar(queryData)
	if err != nil {
		responderError(w, http.StatusInternalServerError, "Error al consultar usuarios")
		return
	}

	responderJSON(w, http.StatusOK, map[string]interface{}{
		"usuarios": usuarios,
		"total":    len(usuarios),
	})
}

// Funciones auxiliares

func responderJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func responderError(w http.ResponseWriter, status int, mensaje string) {
	responderJSON(w, status, map[string]interface{}{
		"error": mensaje,
	})
}

func responderErroresValidacion(w http.ResponseWriter, ctx *qf.ValidationContext) {
	errores := make([]map[string]string, 0)
	for _, err := range ctx.Errors() {
		errores = append(errores, map[string]string{
			"campo":   err.Path,
			"mensaje": err.Message,
		})
	}
	responderJSON(w, http.StatusBadRequest, map[string]interface{}{
		"errores": errores,
	})
}

// Función principal
func main() {
	// Inicializar servicio y esquemas
	servicio := NuevoServicioUsuarios()
	esquemas := InicializarEsquemas()
	manejador := NuevoManejador(servicio, esquemas)

	// Configurar rutas
	router := mux.NewRouter()
	router.HandleFunc("/usuarios", manejador.CrearUsuario).Methods("POST")
	router.HandleFunc("/usuarios", manejador.ConsultarUsuarios).Methods("GET")
	router.HandleFunc("/usuarios/{id}", manejador.ObtenerUsuario).Methods("GET")
	router.HandleFunc("/usuarios/{id}", manejador.ActualizarUsuario).Methods("PUT")
	router.HandleFunc("/usuarios/{id}", manejador.EliminarUsuario).Methods("DELETE")

	// Agregar algunos datos de ejemplo
	datosEjemplo := map[string]interface{}{
		"email":           "admin@ejemplo.com.ar",
		"dni":             "12345678",
		"cuit":            "20-12345678-9",
		"nombre":          "Sistema",
		"apellido":        "Administrador",
		"celular":         "11-6123-4567",
		"fechaNacimiento": "1980-01-01",
		"provincia":       "CABA",
		"localidad":       "Palermo",
		"codigoPostal":    "C1425",
		"direccion":       "Av. Santa Fe 1234",
		"password":        "Admin123!",
		"rol":             "admin",
	}
	
	ctx := qf.NewValidationContext(qf.Strict)
	if err := esquemas.CrearUsuario.Validate(datosEjemplo, ctx); err == nil && !ctx.HasErrors() {
		datosTransformados := extraerDatosTransformados(datosEjemplo, ctx)
		if _, err := servicio.Crear(datosTransformados); err == nil {
			log.Println("Usuario administrador de ejemplo creado")
		}
	}

	// Iniciar servidor
	puerto := ":8080"
	log.Printf("Iniciando servicio de usuarios argentinos en %s", puerto)
	log.Printf("Prueba: curl -X POST http://localhost%s/usuarios -H 'Content-Type: application/json' -d '{\"email\":\"juan.perez@gmail.com\",\"dni\":\"30123456\",\"nombre\":\"Juan\",\"apellido\":\"Pérez\",\"celular\":\"11-5555-1234\",\"fechaNacimiento\":\"1990-05-15\",\"provincia\":\"Buenos Aires\",\"localidad\":\"La Plata\",\"codigoPostal\":\"1900\",\"direccion\":\"Calle 7 nro 123\",\"password\":\"Segura123!\"}'", puerto)
	log.Fatal(http.ListenAndServe(puerto, router))
}