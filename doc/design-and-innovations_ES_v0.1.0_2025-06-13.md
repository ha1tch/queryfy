# Queryfy: Filosofía de Diseño e Innovaciones

## Resumen Ejecutivo

Queryfy representa un replanteo fundamental sobre cómo las aplicaciones Go manejan datos JSON dinámicos. Si bien el sistema de tipos de Go sobresale en seguridad en tiempo de compilación, las aplicaciones del mundo real frecuentemente encuentran datos dinámicos provenientes de APIs, bases de datos y archivos de configuración. Queryfy cierra esta brecha con una API elegante y componible que hace que trabajar con `map[string]interface{}` sea tan placentero como trabajar con estructuras.

## El Espacio del Problema

### El Dilema de los Datos Dinámicos

Los desarrolladores de Go enfrentan una tensión filosófica:
- **Filosofía de Go**: Tipado fuerte, seguridad en tiempo de compilación, lo explícito es mejor que lo implícito
- **Realidad**: 30-40% de las aplicaciones web manejan cantidades significativas de JSON dinámico
- **Soluciones Actuales**: Aserciones de tipo verbosas, validación manual propensa a errores, herramientas fragmentadas

### Puntos de Dolor Comunes

1. **Infierno de Aserciones de Tipo**
```go
// Este patrón aparece miles de veces en bases de código en producción
if data, ok := response["data"].(map[string]interface{}); ok {
    if user, ok := data["user"].(map[string]interface{}); ok {
        if email, ok := user["email"].(string); ok {
            // Finalmente tenemos el email, pero ¿a qué costo?
        }
    }
}
```

2. **Complejidad de Validación**
- El código de validación manual es repetitivo y propenso a errores
- Las etiquetas de struct solo funcionan con tipos conocidos
- No existe un enfoque unificado para esquemas dinámicos

3. **Fragmentación de Herramientas**
- Una biblioteca para validación (validator)
- Otra para consultas JSON (gjson)
- Otra para conversión de estructuras (mapstructure)
- Ninguna solución cohesiva

## Filosofía de Diseño de Queryfy

### 1. **Composabilidad sobre Configuración**

En lugar de archivos de configuración o etiquetas de struct, Queryfy utiliza constructores componibles:

```go
// No esto
type User struct {
    Email string `validate:"required,email" mapstructure:"email"`
    Age   int    `validate:"min=18,max=120" mapstructure:"age"`
}

// Sino esto
schema := builders.Object().
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Range(18, 120))
```

**Por qué**: Los constructores son type-safe, descubribles mediante autocompletado del IDE, y pueden componerse dinámicamente.

### 2. **Mejora Progresiva**

Comenzar simple, agregar complejidad solo cuando sea necesario:

```go
// Nivel 1: Validación básica
err := qf.Validate(data, schema)

// Nivel 2: Agregar consultas
email, _ := qf.Query(data, "user.email")

// Nivel 3: Agregar iteración (v0.2.0)
qf.Each(data, "items[*]", processItem)

// Nivel 4: Agregar transformación
cleaned := qf.Transform(data, schema)

// Nivel 5: Convertir a estructuras (v0.3.0)
user, _ := qf.ToStructT[User](data)
```

### 3. **Mensajes de Error como Ciudadanos de Primera Clase**

Queryfy trata los mensajes de error como una parte crítica de la API:

```go
// No solo "validación fallida"
// Sino: "items[2].price: debe ser mayor que 0, se obtuvo -10"
```

Cada error incluye la ruta completa al campo problemático, haciendo la depuración directa.

### 4. **Rendimiento sin Complejidad**

- Optimización type-switch para tipos JSON comunes
- Compilación y caché de consultas
- Cero asignaciones para validaciones simples
- Sin reflexión para el 90% de los casos de uso

### 5. **Patrones Nativos de Go**

Queryfy sigue patrones establecidos de Go:

```go
// Como json.Unmarshal
err := qf.ToStruct(data, &user)

// Como filepath.Walk  
err := qf.Each(data, "items[*]", func(path string, item interface{}) error {
    return nil
})

// Como sql.Scanner
err := qf.ValidateToStruct(data, &result, schema)
```

### 6. **Seguridad en Tiempo de Compilación para Datos Dinámicos**

Queryfy logra algo único en el ecosistema Go: llevar garantías de tiempo de compilación a la validación de datos en runtime. Mientras trabaja con `map[string]interface{}`, los desarrolladores aún obtienen:

- **Cadenas de métodos type-safe** - Las composiciones inválidas no compilarán
- **Inteligencia del IDE** - Autocompletado completo, refactorización y documentación en línea
- **Detección temprana de errores** - Errores capturados durante el desarrollo, no en producción

```go
// Estos no compilarán - errores capturados inmediatamente
schema := builders.Number().Email()     // ❌ Email() indefinido
schema := builders.String().Min(5).Email() // ❌ Email() no disponible después de Min()

// Solo las composiciones válidas compilan
schema := builders.String().Email().Required() // ✅ El IDE guía el camino
```

Esto crea **contratos en tiempo de compilación para datos dinámicos** - la misma seguridad que los desarrolladores Go esperan, extendida a la validación en runtime.

## Lo que Queryfy Hace Mejor

### 1. **API Unificada para Todo el Flujo de Trabajo**

Otras herramientas resuelven piezas del rompecabezas. Queryfy proporciona el flujo de trabajo completo:

| Necesidad | Enfoque Tradicional | Queryfy |
|-----------|---------------------|---------|
| Validar | etiquetas validator + código manual | `qf.Validate(data, schema)` |
| Consultar | gjson o navegación manual | `qf.Query(data, "user.email")` |
| Transformar | Conversión manual de tipos | `qf.Transform(data, schema)` |
| Iterar | Bucles manuales con aserciones de tipo | `qf.Each(data, "items[*]", fn)` |
| Convertir | mapstructure | `qf.ToStruct(data, &user)` |

### 2. **Esquema como Fuente Única de Verdad**

Una definición de esquema sirve múltiples propósitos:

```go
userSchema := builders.Object().
    Field("email", builders.String().Email().Transform(transformers.Lowercase())).
    Field("age", builders.Number().Min(18))

// Usar para validación
err := qf.Validate(userData, userSchema)

// Usar para transformación
cleaned := qf.Transform(userData, userSchema)

// Usar para conversión de struct con validación
var user User
err := qf.ValidateToStruct(userData, &user, userSchema)
```

### 3. **Reporte de Errores Contextual**

A diferencia de las listas planas de errores, Queryfy mantiene el contexto completo:

```go
// validator: "email debe ser un email válido"
// Queryfy: "addresses[2].contact.email: debe ser una dirección de email válida"
```

### 4. **Patrón Constructor Type-Safe**

El patrón constructor proporciona:
- Verificación de métodos en tiempo de compilación
- Autocompletado del IDE
- API similar al lenguaje natural
- Sin DSL basado en strings para aprender

### 5. **Rendimiento Listo para Producción**

- Compilación de consultas con caché
- Optimización type-switch evita reflexión
- Características de rendimiento predecibles
- Sin asignaciones ocultas

### 6. **DSL Type-Safe en lugar de Configuración por Strings**

Las bibliotecas de validación tradicionales usan DSLs basados en strings que fallan en runtime:

```go
// etiquetas struct - errores tipográficos y reglas inválidas solo capturadas en runtime
type User struct {
    Email string `validate:"required,emal"` // ¡Error tipográfico! Pánico en runtime
    Age   int    `validate:"email"`        // ¡Regla incorrecta! Error en runtime
}

// Queryfy - todos los errores capturados en tiempo de compilación
schema := builders.String().Emal()  // ❌ No compilará
schema := builders.Number().Email() // ❌ No compilará
```

El patrón constructor fluido actúa como un **DSL type-safe** donde el compilador y el IDE trabajan juntos para prevenir errores antes de que puedan ocurrir.

## Innovaciones Propuestas

### 1. **Métodos de Iteración (v0.2.0)**

**Problema**: No hay forma elegante de procesar múltiples elementos coincidentes

**Solución**: Tres métodos construidos específicamente que siguen patrones de Go:

```go
// Each - Procesar elementos
qf.Each(data, "items[*]", func(path string, item interface{}) error {
    fmt.Printf("Procesando %s\n", path)
    return nil
})

// Collect - Transformar y reunir
prices, _ := qf.Collect(data, "items[*].price", func(p interface{}) (interface{}, error) {
    return p.(float64) * 1.1, nil // Agregar impuesto
})

// ValidateEach - Validar múltiples elementos
err := qf.ValidateEach(data, "items[*]", itemSchema)
```

**Por qué es mejor**: 
- Mantiene el contexto de ruta para depuración
- Soporta terminación temprana
- Componible con esquemas existentes
- Sin necesidad de bucles manuales de aserción de tipo

### 2. **Conversión a Estructuras (v0.3.0)**

**Problema**: Pasar de `map[string]interface{}` validado a estructuras requiere otra biblioteca

**Solución**: Conversión integrada de estructuras que aprovecha esquemas existentes:

```go
// Conversión simple
var user User
err := qf.ToStruct(userData, &user)

// Con validación
err := qf.ValidateToStruct(userData, &user, userSchema)

// Conveniencia genérica
user, err := qf.ToStructT[User](userData)
```

**Por qué es mejor**:
- Una biblioteca en lugar de dos
- Reutiliza definiciones de esquema
- Aplica transformaciones durante la conversión
- Mantiene el excelente reporte de errores de Queryfy

### 3. **Pipeline de Transformación** (Implementado en v0.1.0)

**Ya Entregado**: La transformación de datos está completamente integrada en la validación

**Implementación Actual**:
```go
// Pipeline de transformación componible
schema := builders.String().
    Transform(transformers.Trim()).
    Transform(transformers.Lowercase()).
    Transform(transformers.NormalizeEmail()).
    Email()

// Validar y transformar en un paso
transformed, err := schema.ValidateAndTransform(data, ctx)
```

**Transformadores Disponibles**:
- **String**: Trim, Lowercase, Uppercase, NormalizeWhitespace, Truncate
- **Números**: ToInt, ToFloat64, Round, Clamp, FromPercentage
- **Teléfonos**: NormalizePhone con detección de país
- **Valores por defecto**: Valores predeterminados para campos opcionales
- **Personalizado**: Cualquier transformación definida por el usuario

### 4. **Validación DateTime** (Implementado en v0.1.0)

**Ya Entregado**: Validación completa de fecha/hora

**Implementación Actual**:
```go
// Soporte de múltiples formatos
birthDateSchema := builders.DateTime().
    DateOnly().              // YYYY-MM-DD
    Past().                  // Debe estar en el pasado
    Age(18, 100).           // Validación de edad
    Required()

// Validación de horario laboral
appointmentSchema := builders.DateTime().
    Format("2006-01-02 15:04").
    Future().
    BusinessDay().           // Solo lunes-viernes
    Between(start, end).
    Required()
```

### 5. **Validación de Campos Dependientes** (Implementado en v0.1.0)

**Ya Entregado**: Validación condicional basada en otros campos

**Implementación Actual**:
```go
// Formulario de pago con campos condicionales
paymentSchema := builders.Object().WithDependencies().
    Field("paymentMethod", builders.String().
        Enum("credit_card", "paypal", "bank_transfer")).
    DependentField("cardNumber",
        builders.Dependent("cardNumber").
            When(builders.WhenEquals("paymentMethod", "credit_card")).
            Then(builders.String().Pattern(`^\d{16}$`).Required())).
    DependentField("paypalEmail",
        builders.Dependent("paypalEmail").
            When(builders.WhenEquals("paymentMethod", "paypal")).
            Then(builders.String().Email().Required()))
```

### 6. **Composición Dinámica de Esquemas** (v0.2.0-v0.3.0)

**Problema**: Las etiquetas de struct estáticas no pueden adaptarse a condiciones en tiempo de ejecución

**Solución**: Métodos para composición de esquemas en runtime:

```go
// Esquema base
baseUser := builders.Object().
    Field("id", builders.String().Required()).
    Field("email", builders.String().Email())

// Composición condicional
if isPremiumUser {
    baseUser.AddField("subscription", subscriptionSchema)
}

// Campos específicos del entorno
if config.Region == "EU" {
    baseUser.AddField("gdprConsent", builders.Bool().Required())
}

// Fusionar esquemas
finalSchema := baseSchema.Merge(regionSchema).Merge(featureSchema)
```

**Implementación**: Métodos como `AddField()`, `RemoveField()`, `Merge()` permiten flexibilidad poderosa en runtime mientras mantienen seguridad en tiempo de compilación en los métodos constructores.

### 7. **Clonación de Esquemas** (v0.2.0-v0.3.0)

**Problema**: Modificar esquemas afecta a todos los usuarios de ese esquema

**Solución**: Clonación profunda para composición segura:

```go
// Composición segura sin modificar el original
premiumSchema := baseSchema.Clone().(*builders.ObjectSchema).
    AddField("tier", builders.String().Enum("gold", "platinum"))

// baseSchema original permanece sin cambios
```

**Por qué es crítico**: Permite reutilización de esquemas en diferentes contextos sin efectos secundarios.

### 8. **Tipos de Estado del Constructor** (Mejora Futura)

**Problema**: Algunas combinaciones de métodos no tienen sentido pero solo fallan en runtime

**Solución**: Patrón type-state para garantías aún más fuertes en tiempo de compilación:

```go
// Diferentes tipos para diferentes estados del constructor
type StringSchemaBase struct { *StringSchema }
type StringSchemaWithEmail struct { *StringSchema }

// Email() retorna un tipo diferente que no tiene Min/Max
func (s *StringSchemaBase) Email() *StringSchemaWithEmail {
    // Min() y Max() no disponibles en StringSchemaWithEmail
}

// Esto no compilaría:
schema := builders.String().Email().Min(5) // ❌ Error de compilación
```

**Compromiso**: Implementación más compleja pero máxima seguridad en tiempo de compilación.

### 9. **Introspección de Esquemas** (v0.3.0)

**Problema**: Los esquemas son opacos - difíciles de depurar o documentar

**Solución**: Métodos para inspeccionar la configuración del esquema:

```go
schema := builders.String().Min(3).Max(20).Email()
fmt.Println(schema.Describe())
// Salida: "string,min:3,max:20,format:email"

// Para mejores mensajes de error:
// Esperado: string,min:3,max:20,format:email
// Recibido: "ab" (demasiado corto)
```

**Beneficios**: Esquemas auto-documentados, mejores mensajes de error, soporte de depuración.

### 10. **Patrón Visitor de Esquemas** (v0.4.0)

**Problema**: El análisis complejo de esquemas requiere type switches

**Solución**: Patrón visitor para procesamiento extensible de esquemas:

```go
type SchemaVisitor interface {
    VisitString(*StringSchema) error
    VisitNumber(*NumberSchema) error
    VisitObject(*ObjectSchema) error
    VisitArray(*ArraySchema) error
}

// Permite: generación de documentación, comparación de esquemas,
// extracción de reglas de validación, herramientas de migración
```

**Casos de uso**: Herramientas avanzadas, análisis de esquemas, generación de código.

## Experiencia del Desarrollador Primero

Queryfy prioriza la experiencia del desarrollador en todos los niveles:

### Integración con IDE
- **Autocompletar todo** - Sin memorizar DSLs de strings
- **Ir a definición** - Saltar a definiciones de esquema
- **Buscar usos** - Ver dónde se usan los esquemas
- **Refactorización segura** - Renombrar campos con confianza

### Ciclo de Retroalimentación en Tiempo de Compilación
```go
// Retroalimentación inmediata mientras escribes
schema := builders.
    String().     // IDE muestra: Email(), URL(), Pattern(), Min(), Max()...
    Email().      // IDE muestra: Required(), Optional(), Transform()...
    Min(5)        // ❌ Subrayado rojo - Min() no disponible después de Email()
```

### Código Auto-Documentado
El esquema ES la documentación:
```go
// Este esquema te dice todo
userSchema := builders.Object().
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Integer().Range(18, 120))

// vs. etiquetas struct crípticas
type User struct {
    Email string `validate:"required,email" json:"email" db:"email"`
    Age   int    `validate:"min=18,max=120" json:"age" db:"age"`
}
```

## La Visión Completa

Cuando se realiza completamente, Queryfy proporciona una solución cohesiva para datos dinámicos con **seguridad completa en tiempo de compilación** y **características completas en runtime**:

```go
// Cada línea a continuación tiene verificación en tiempo de compilación y soporte del IDE
orderSchema := builders.Object().
    Field("id", builders.String().Required()).                    // ✓ Type-safe
    Field("date", builders.DateTime().ISO8601().Future()).       // ✓ Validación de fecha
    Field("items", builders.Array().Of(itemSchema)).             // ✓ Esquemas anidados
    Field("total", builders.Number().Min(0).                     // ✓ Restricciones
        Transform(transformers.Round(2)))                         // ✓ Transformaciones

// Operaciones type-safe en datos dinámicos
validated, err := qf.Validate(data, orderSchema)                 // Validación
transformed, err := qf.ValidateAndTransform(data, orderSchema)   // Transformar
order, err := qf.ToStructT[Order](transformed)                   // Conversión de tipo

// Todo con garantías en tiempo de compilación típicamente perdidas con map[string]interface{}
```

El flujo de trabajo completo mantiene la seguridad en cada paso:

```go
// 1. Recibir datos dinámicos
data := receiveWebhookPayload()

// 2. Definir esquema una vez
orderSchema := builders.Object().
    Field("id", builders.String().Required()).
    Field("items", builders.Array().Of(itemSchema)).
    Field("total", builders.Number().Min(0))

// 3. Validar y transformar
cleaned, err := qf.ValidateAndTransform(data, orderSchema)

// 4. Consultar valores específicos
customerEmail, _ := qf.Query(cleaned, "customer.email")

// 5. Procesar colecciones
qf.Each(cleaned, "items[*]", calculateInventory)

// 6. Convertir a struct para lógica de negocio
order, _ := qf.ToStructT[Order](cleaned)

// Todo con excelentes mensajes de error si algo sale mal
```

## Ventajas Competitivas

### vs. etiquetas struct (validator)
- **Validación en tiempo de compilación** de definiciones de esquema
- **Autocompletado del IDE** para todas las restricciones
- **Cadenas de métodos type-safe** vs strings propensos a errores
- Esquemas dinámicos sin recompilación
- Funciona con `interface{}`

### vs. gjson
- Validación integrada
- Consultas type-safe con esquemas
- Soporte de transformación

### vs. mapstructure  
- Validación incluida
- Conversión dirigida por esquema
- Mejores mensajes de error

### vs. Aserciones de Tipo Manuales
- 10x menos código
- Manejo consistente de errores
- Mantenible y testeable

## Principios de Diseño

1. **Hacer las Cosas Simples Simples**: La validación básica debe ser una línea
2. **Hacer las Cosas Complejas Posibles**: Soportar casos de uso avanzados sin compromiso
3. **Los Errores Deben Guiar Soluciones**: Cada error debe decirte cómo arreglarlo
4. **Componer, No Configurar**: Construir comportamiento complejo a partir de partes simples
5. **Rendimiento por Defecto**: Camino rápido para casos comunes
6. **Seguir Patrones de Go**: Sentirse familiar para desarrolladores de Go
7. **Seguridad en Tiempo de Compilación Primero**: Capturar errores en tiempo de construcción, no en runtime

## Mensajes Clave

Queryfy lleva **seguridad en tiempo de compilación a datos dinámicos**. No es solo una biblioteca de validación—es un DSL type-safe para trabajar con `map[string]interface{}` que mantiene las garantías de seguridad de Go a lo largo de todo el pipeline de datos.

- **"DSL Type-Safe para Datos Dinámicos"** - Los constructores son un DSL con verificación del compilador
- **"Contratos en Tiempo de Compilación"** - Los esquemas son contratos, aplicados por el compilador
- **"Diseño IDE-First"** - Construido para productividad del desarrollador con soporte completo de herramientas
- **"Capturar Errores Antes del Runtime"** - La propuesta de valor definitiva

## Conclusión

Queryfy no es solo otra biblioteca de validación—es un replanteo completo de cómo las aplicaciones Go deberían manejar datos dinámicos. Al proporcionar una API unificada y componible que cubre validación, consultas, transformación y conversión de tipos, Queryfy replantea categorías enteras de código repetitivo mientras mantiene los principios de simplicidad y claridad de Go.

Las adiciones propuestas (métodos de iteración y conversión de estructuras) completan la visión, haciendo de Queryfy una propuesta completa para el manejo de JSON dinámico en Go. Estas no son solo características—son las piezas faltantes, aproximándonos a un futuro en el que trabajar con datos dinámicos en Go fluya tan naturalmente como trabajar con tipos estáticos.
