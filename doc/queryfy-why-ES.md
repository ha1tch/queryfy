# ¿Queryfy? ¿Por qué?

## El Problema Oculto en Todo Servicio Go

**Los desarrolladores de Go enfrentan una tensión fundamental. Mientras que el sistema de tipos de Go sobresale en seguridad en tiempo de compilación, las aplicaciones del mundo real deben manejar datos dinámicos provenientes de APIs, webhooks, bases de datos y archivos de configuración. Esto crea una división filosófica que atraviesa la mayoría de las bases de código Go, la tensión es real, se "siente" e introduce incontables momentos de fricción e indecisión durante el proceso de desarrollo de sistemas complejos. **

### La Base de Código con Doble Personalidad

Todo servicio web Go contiene dos enfoques distintos para el manejo de datos:

**El mundo de las estructuras** - Lo que mostramos en las revisiones de código:
```go
type User struct {
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18"`
}
```

**El mundo dinámico** - Lo que realmente maneja datos en producción:
```go
func HandleWebhook(payload map[string]interface{}) error {
    // Páginas de aserciones de tipo y validación manual
    if user, ok := payload["user"].(map[string]interface{}); ok {
        if email, ok := user["email"].(string); ok {
            // Más verificaciones anidadas...
        }
    }
}
```

El mundo dinámico es una realidad inevitable, es el modo natural de los datos que nos llegan en el cable, es el formato inesperado de los archivos de configuración: el mundo real no cuenta naturalmente con un sistema de tipos seguro.

Esto no se trata solo de estilo de código. Representa una falsa dicotomía que la comunidad Go ha internalizado: las estructuras son "Go apropiado" mientras que `map[string]interface{}` es un "mal necesario", como si fuera posible forzar la resolución de la tensión exclusivamente en un modo u otro, o peor aún, como si fuera posible ocultar esta tensión fundamental.

## Nadie Escribe Servicios Únicamente con Estructuras

He aquí la realidad: ningún servicio Go en producción es puramente basado en estructuras. Todo servicio maneja:
- Webhooks con estructura desconocida
- Parámetros de consulta API flexibles
- Variables GraphQL
- Configuraciones multi-tenant
- Respuestas de APIs externas

Sin embargo, nuestras herramientas asumen el mundo de sólo estructuras. Terminamos con:
```go
// go-playground/validator para estructuras
// + validación manual para partes dinámicas
// + gjson para consultar datos anidados
// + mapstructure para conversión de tipos
// + 200 líneas de código de pegamento
// + manejo inconsistente de errores
// = Tu "solución" actual
```

## La Percepción Fundamental de Queryfy

Queryfy reconoce que **la seguridad no está en la estructura de datos, está en cómo describimos lo que esperamos**. Un esquema es tan type-safe como una definición de estructura:

```go
// Esto es tan seguro como una definición de estructura
userSchema := builders.Object().
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Min(18).Max(120))

// Y funciona uniformemente con todos los tipos de datos
err := qf.Validate(structData, userSchema)    // ✓
err := qf.Validate(mapData, userSchema)       // ✓
err := qf.Validate(jsonBytes, userSchema)     // ✓
```

## Lo Inexplorado: Seguridad en Tiempo de Compilación para Datos Dinámicos

### Seguridad de Tipos en Tiempo de Construcción

Queryfy lleva las garantías de tiempo de compilación de Go a la validación dinámica mediante su API de constructores:

```go
// Estos son errores de tiempo de compilación
schema := builders.Number().Email()        // ❌ No compilará
schema := builders.String().Min(5).Email() // ❌ Email() no disponible después de Min()

// El IDE te guía hacia composiciones válidas
schema := builders.String().Email().Required() // ✓ Soporte completo de autocompletado
```

Esto no es solamente conveniencia—es un cambio fundamental:  es obtener la misma seguridad construyendo reglas de validación que la que obtienes definiendo estructuras.

### La Experiencia del IDE

A diferencia de las etiquetas de validación basadas en strings, cada método de Queryfy es:
- **Descubrible**: Escribe `builders.` y ve todas las opciones
- **Contextual**: Después de `.String()`, solo aparecen métodos de string
- **Documentado**: Documentación en línea para cada método
- **Refactorizable**: Renombra campos en toda tu base de código con seguridad

## Disolviendo la Falsa Dicotomía

Queryfy demuestra que la división entre estructuras y dinámico nunca fue necesaria. No necesitas dos modelos mentales, dos enfoques de validación, o dos conjuntos de manejo de errores.

### Un Modelo Mental Único para Todo

En lugar de alternar entre "modo estructura" y "modo dinámico":

```go
// Antes: Diferentes enfoques para diferentes datos
func validateStruct(u User) error { 
    return validator.Validate(u) 
}

func validateMap(data map[string]interface{}) error {
    // 50 líneas de validación manual
}

// Después: Un enfoque para todos los datos
func validate(data interface{}) error {
    return qf.Validate(data, userSchema)
}
```

### Refinamiento Progresivo de Datos

Queryfy proporciona un pipeline claro de lo desconocido a lo conocido:

```go
// 1. Recibir datos desconocidos
rawData := receiveWebhook()

// 2. Validar estructura
if err := qf.Validate(rawData, schema); err != nil {
    return err
}

// 3. Transformar y limpiar
cleaned, _ := qf.ValidateAndTransform(rawData, schema)

// 4. Consultar valores específicos (¡sin aserciones de tipo!)
email, _ := qf.Query(cleaned, "user.email")

// 5. Convertir a estructura cuando sea necesario
user, _ := qf.ToStructT[User](cleaned)
```

Cada paso es explícito, type-safe y testeable.

## La Innovación de Transformación

Queryfy no solo valida—ayuda a corregir problemas comunes de datos:

```go
// Definir validación Y transformación juntas
emailSchema := builders.Transform(
    builders.String().Email()
).Add(transformers.Trim()).
  Add(transformers.Lowercase())

// "  John.Doe@EXAMPLE.COM  " → "john.doe@example.com"
```

Esto resuelve problemas reales:
- APIs que envían números como strings
- Números de teléfono en varios formatos
- Formatos de fecha inconsistentes
- Valores monetarios con símbolos

El pipeline de transformación es:
- **Auditable**: Cada transformación se registra
- **Componible**: Encadena operaciones simples
- **Testeable**: Funciones puras sin efectos secundarios

## Características Prácticas para el Desorden del Mundo Real

### Modos Strict vs. Loose

Queryfy reconoce que los datos no siempre son perfectos:

```go
// Modo strict: Para servicios internos
err := qf.Validate(data, schema)

// Modo loose: Para APIs externas
err := qf.ValidateWithMode(data, schema, qf.Loose)
// "42" valida como número 42
// Los campos extra se ignoran
```

### Validación de Campos Dependientes

Los formularios reales tienen relaciones complejas:

```go
paymentSchema := builders.Object().WithDependencies().
    Field("method", builders.String().Enum("card", "paypal")).
    DependentField("cardNumber",
        builders.Dependent("cardNumber").
            When(builders.WhenEquals("method", "card")).
            Then(builders.String().Required().Pattern(`^\d{16}$`)))
```

### Lógica de Negocio Personalizada

Es posible incrustar validación compleja directamente:

```go
.Custom(func(value interface{}) error {
    order := value.(map[string]interface{})
    items := order["items"].([]interface{})
    total := order["total"].(float64)
    
    calculatedTotal := calculateItemsTotal(items)
    if math.Abs(total - calculatedTotal) > 0.01 {
        return fmt.Errorf("el total no coincide con la suma de items")
    }
    return nil
})
```

## Dónde Queryfy Puede Destacar

### Gateways de API
Diferente validación por ruta, manejando múltiples formatos upstream:
```go
routeSchemas := map[string]queryfy.Schema{
    "/v1/users":  userSchemaV1,
    "/v2/users":  userSchemaV2,
    "/webhooks":  webhookSchema,
}
```

### B2B Multi-tenant
Diferentes reglas de validación por cliente:
```go
customerSchemas := map[string]queryfy.Schema{
    "cliente-empresa": strictSchema,
    "cliente-startup": lenientSchema,
}
```

### Servidores GraphQL
Las variables son inherentemente dinámicas:
```go
func validateVariables(query string, variables map[string]interface{}) error {
    schema := getSchemaForQuery(query)
    return qf.Validate(variables, schema)
}
```

##  Una Filosofía: Componer en Tiempo de Compilación, Validar en Tiempo de Ejecución

Este principio tiene implicaciones profundas:

1. **Todos los comportamientos de validación son conocidos en tiempo de compilación**
2. **El runtime solo selecciona entre comportamientos pre-compuestos**
3. **Sin generación dinámica de esquemas desde entrada del usuario**
4. **Cada validación posible puede ser testeada**

Esto habilita:
- Despliegues blue-green con cambios de validación
- Pruebas A/B de reglas de validación
- Despliegue progresivo de validación más estricta
- Rastros de auditoría completos

## Detalles de Rendimiento e Implementación

### Optimizaciones de Cero Asignación
- Slices de ruta pre-asignados para anidamiento típico
- Constructores de strings en lugar de concatenación
- Type switches para evitar reflexión donde sea posible
- Compilación y caché de rutas de consulta

### Mensajes de Error como Documentación
Cada error es accionable:
- `"debe ser una dirección de email válida"` no `"validación fallida"`
- `"la longitud debe ser al menos 8, se obtuvo 5"` no `"longitud inválida"`
- `"debe ser uno de: admin, user, guest"` muestra opciones válidas

## La Conclusión

Queryfy no es solo otra biblioteca de validación. Es una reconciliación entre dos partes del desarrollo Go que han estado innecesariamente enfrentadas. Demuestra que:

1. **El manejo de datos dinámicos puede ser tan seguro como el manejo de estructuras**
2. **No necesitas diferentes modelos mentales para diferentes tipos de datos**
3. **La validación puede ayudar a corregir datos, no solo rechazarlos**
4. **Las reglas de negocio complejas pueden expresarse declarativamente**

Al llevar la seguridad de tiempo de compilación a la validación en runtime, Queryfy hace que trabajar con `map[string]interface{}` sea tan natural y seguro como trabajar con estructuras. No se trata de elegir entre flexibilidad y seguridad—se trata de tener ambas.

## Ruta de Integración

Queryfy complementa el código existente:

1. Mantén usando etiquetas de struct para validación pura de estructuras
2. Usa Queryfy para manejo de datos dinámicos
3. Comparte esquemas entre servicios como paquetes Go
4. Migra gradualmente la lógica de validación según sea necesario

El resultado es código más limpio, menos bugs, manejo consistente de errores, y confianza al tratar con datos externos. Más importante aún, elimina la culpa y complejidad entorno al manejo de datos dinámicos, convirtiéndolo en un participante de primera clase en las aplicaciones Go.