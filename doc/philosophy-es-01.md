# Queryfy: Componer en Tiempo de Compilación, Validar en Tiempo de Ejecución

## El Principio Fundamental

Queryfy encarna una filosofía de diseño que procura aliviar la tensión eterna entre flexibilidad y predictibilidad en los sistemas de software. En su esencia, esta filosofía reconoce que la **composición** y la **configuración** no constituyen fuerzas opuestas, sino más bien dos caras de la misma moneda que se manifiestan en diferentes fases del ciclo de vida de la aplicación.

## El Modelo de Dos Fases

### Fase 1: Composición en Tiempo de Compilación (Máxima Flexibilidad)

Durante la fase de construcción, Queryfy proporciona flexibilidad ilimitada a través de constructores componibles. Es aquí donde reinan la creatividad y la adaptabilidad:

```go
// Se compone la lógica de validación con libertad absoluta
schema := builders.Object().
    Field("email", builders.String().Email().Required()).
    Field("age", builders.Number().Min(18).Max(120)).
    Field("preferences", builders.Object().
        Field("notifications", builders.Bool()).
        Field("theme", builders.String().Enum("light", "dark")))

// Se componen comportamientos mediante transformación
emailSchema := builders.Transform(
    builders.String().Email()
).Add(transformers.Lowercase()).Add(transformers.Trim())

// Se compone lógica compleja mediante combinadores
contactSchema := builders.Or(
    builders.String().Email(),
    builders.String().Pattern(`^\+?[1-9]\d{9,14}$`)
)
```

### Fase 2: Validación en Tiempo de Ejecución (Determinismo Completo)

Una vez compuestos, los esquemas se convierten en validadores inmutables con comportamiento predecible y determinista:

```go
// En tiempo de ejecución, el comportamiento es fijo y predecible
err := qf.Validate(userData, schema)  // Misma entrada → Misma salida, siempre

// Sin mutaciones, sin sorpresas
// El esquema no puede cambiar durante la validación
// El proceso de validación es puro y libre de efectos secundarios
```

## Por Qué Esto Importa

### 1. **Claridad del Modelo Mental**

Los desarrolladores pueden pensar en dos modos distintos:
- **Modo diseño**: "¿Cómo se compone el comportamiento necesario?"
- **Modo ejecución**: "¿Qué sucederá cuando esto valide?"

Esta separación reduce la carga cognitiva y facilita el razonamiento sobre los sistemas.

### 2. **Flexibilidad con Seguridad de Tipos**

La composición en tiempo de compilación permite que el compilador de Go detecte errores:

```go
// Esto no compilará - Email() no está disponible en Number
schema := builders.Number().Email()  // ❌ Error de compilación

// El compilador guía hacia composiciones válidas
schema := builders.String().Email()  // ✅ Seguro en tipos
```

### 3. **Rendimiento Mediante Invariancia**

Dado que los esquemas son invariantes en tiempo de ejecución, Queryfy puede optimizar agresivamente:
- Patrones regex compilados una sola vez
- Rutas de validación predeterminadas
- Sin árboles de decisión en tiempo de ejecución
- Uso de memoria predecible

## Los Patrones de Composición

### Patrón 1: Composición por Capas

```go
// Capa base
baseUser := builders.Object().
    Field("id", builders.String().UUID()).
    Field("createdAt", builders.DateTime())

// Capa de mejora
activeUser := baseUser.
    Field("email", builders.String().Email().Required()).
    Field("lastLogin", builders.DateTime().Required())

// Capa de especialización
adminUser := activeUser.
    Field("permissions", builders.Array().Required()).
    Field("auditLog", builders.Array())
```

### Patrón 2: Composición Comportamental

```go
// Se componen comportamientos, no solo estructura
validatedString := builders.String().
    Transform(transformers.Trim()).
    Transform(transformers.Lowercase()).
    MinLength(3).
    MaxLength(50)

// Cada método agrega comportamiento, creando un pipeline
// El esquema final representa la composición de todos los comportamientos
```

### Patrón 3: Composición Condicional

```go
// Incluso la lógica condicional se compone en tiempo de compilación
schema := builders.Object().WithDependencies().
    Field("userType", builders.String().Enum("person", "company")).
    DependentField("firstName",
        builders.Dependent("firstName").
            When(builders.WhenEquals("userType", "person")).
            Then(builders.String().Required())).
    DependentField("companyName",
        builders.Dependent("companyName").
            When(builders.WhenEquals("userType", "company")).
            Then(builders.String().Required()))

// Las condiciones se evalúan en tiempo de ejecución,
// pero la estructura está fija en tiempo de compilación
```

## Implicaciones Arquitectónicas

### Las Cuatro Capas

```
┌─────────────────────────────┐
│   Capa de Declaración      │ ← Qué validar
├─────────────────────────────┤
│   Capa de Composición      │ ← Cómo construirlo
├─────────────────────────────┤
│   Capa de Comportamiento   │ ← Fijo en tiempo de compilación
├─────────────────────────────┤
│   Capa de Ejecución        │ ← Tiempo de ejecución determinista
└─────────────────────────────┘
```

Cada capa posee una responsabilidad específica:
- **Declaración**: Definir la estructura de los datos
- **Composición**: Combinar comportamientos y restricciones
- **Comportamiento**: El resultado inmutable de la composición
- **Ejecución**: Validación pura y predecible

### Flexibilidad Mediante Inmutabilidad

Paradójicamente, la inmutabilidad en tiempo de ejecución habilita mayor flexibilidad en tiempo de compilación:

```go
// Dado que los esquemas son inmutables, pueden compartirse con seguridad
var (
    emailSchema = builders.String().Email().Required()
    phoneSchema = builders.String().Pattern(`^\+?[1-9]\d{9,14}$`)
)

// Reutilización sin temor a mutaciones
userSchema := builders.Object().
    Field("primaryEmail", emailSchema).
    Field("secondaryEmail", emailSchema).  // Reutilización segura
    Field("phone", phoneSchema)
```

## El Continuo Configuración-Composición

### No Es Una Disyuntiva

El verdadero poder emerge cuando se reconoce que configuración y composición trabajan en conjunto, no en oposición. La configuración selecciona entre comportamientos pre-compuestos:

```go
// Esquemas pre-compuestos para diferentes versiones de API
var (
    // API v1.0 - Validación básica
    schemaV1 = builders.Object().
        Field("userId", builders.String().Required()).
        Field("action", builders.String().Required())
    
    // API v1.1 - Validación de timestamp agregada
    schemaV1_1 = builders.Object().
        Field("userId", builders.String().Required()).
        Field("action", builders.String().Required()).
        Field("timestamp", builders.DateTime().ISO8601().Required())
    
    // API v2.0 - Mejorada con campos de auditoría
    schemaV2 = builders.Object().
        Field("userId", builders.String().UUID().Required()).
        Field("action", builders.String().Enum("create", "update", "delete").Required()).
        Field("timestamp", builders.DateTime().ISO8601().Required()).
        Field("metadata", builders.Object().Optional())
)

// La configuración selecciona qué composición utilizar
func getSchemaForRequest(r *http.Request) Schema {
    apiVersion := os.Getenv("API_VERSION")
    
    // Despliegue progresivo basado en configuración
    if featureFlag.IsEnabled("api_v2", r.Header.Get("X-User-ID")) {
        return schemaV2
    }
    
    switch apiVersion {
    case "1.1":
        return schemaV1_1
    case "2.0":
        return schemaV2
    default:
        return schemaV1
    }
}
```

### Configuración como Selección de Composición

La configuración no crea comportamiento—selecciona entre comportamientos pre-compuestos:

```go
// Se componen todos los comportamientos posibles en tiempo de compilación
type ValidationStrategy struct {
    Strict    Schema
    Lenient   Schema  
    Migration Schema  // Acepta tanto formatos antiguos como nuevos
}

var userValidation = ValidationStrategy{
    Strict: builders.Object().
        Field("email", builders.String().Email().Required()).
        Field("phone", builders.String().Pattern(`^\+\d{11,14}$`).Required()),
    
    Lenient: builders.Object().
        Field("email", builders.String().Email().Required()).
        Field("phone", builders.String().Optional()),
    
    Migration: builders.Object().
        Field("email", builders.String().Email().Required()).
        Field("phone", builders.Or(
            builders.String().Pattern(`^\+\d{11,14}$`),    // Formato nuevo
            builders.String().Pattern(`^\d{10}$`),          // Formato heredado
        )),
}

// La configuración determina qué composición utilizar
func validateUser(data map[string]interface{}) error {
    mode := config.GetString("validation.mode")
    
    var schema Schema
    switch mode {
    case "strict":
        schema = userValidation.Strict
    case "migration":
        schema = userValidation.Migration
    default:
        schema = userValidation.Lenient
    }
    
    return qf.Validate(data, schema)
}
```

### Adopción Progresiva de Funcionalidades

La configuración habilita el despliegue gradual de nuevas reglas de validación:

```go
// Los feature flags controlan qué validaciones están activas
type FeatureSchemas struct {
    schemas map[string]Schema
    mu      sync.RWMutex
}

func (fs *FeatureSchemas) Register(feature string, schema Schema) {
    fs.mu.Lock()
    defer fs.mu.Unlock()
    fs.schemas[feature] = schema
}

func (fs *FeatureSchemas) GetActive() Schema {
    fs.mu.RLock()
    defer fs.mu.RUnlock()
    
    // Se componen funcionalidades activas basándose en configuración
    active := builders.Object()
    
    for feature, schema := range fs.schemas {
        if featureFlag.IsEnabled(feature) {
            // Fusionar esquemas para funcionalidades habilitadas
            active = mergeSchemas(active, schema)
        }
    }
    
    return active
}

// Uso
features := &FeatureSchemas{}

// Registrar esquemas de funcionalidades
features.Register("enhanced_validation", 
    builders.Object().Field("score", builders.Number().Min(0).Max(100)))

features.Register("ml_predictions",
    builders.Object().Field("confidence", builders.Number().Min(0).Max(1)))

// La configuración determina las funcionalidades activas
activeSchema := features.GetActive()
```

## Patrones Avanzados

### Selección Dinámica de Esquemas

La configuración puede seleccionar esquemas dinámicamente basándose en el contexto de ejecución:

```go
// Registro de esquemas con gestión de versiones
type SchemaRegistry struct {
    versions map[string]map[string]Schema  // dominio -> versión -> esquema
}

func (sr *SchemaRegistry) GetSchema(domain, clientVersion string) Schema {
    // Versión por defecto desde configuración
    defaultVersion := config.GetString(fmt.Sprintf("%s.default_version", domain))
    
    // Override de versión del cliente
    if clientVersion != "" && sr.supportsVersion(domain, clientVersion) {
        return sr.versions[domain][clientVersion]
    }
    
    // Despliegue gradual basado en porcentaje
    rolloutPercent := config.GetInt(fmt.Sprintf("%s.v2_rollout_percent", domain))
    if shouldRollout(rolloutPercent) {
        if schema, ok := sr.versions[domain]["v2"]; ok {
            return schema
        }
    }
    
    return sr.versions[domain][defaultVersion]
}

// Esquemas pre-compuestos registrados al inicio
func initSchemas(registry *SchemaRegistry) {
    // Esquemas del dominio usuario
    registry.Register("user", "v1", builders.Object().
        Field("name", builders.String().Required()).
        Field("email", builders.String().Email()))
    
    registry.Register("user", "v2", builders.Object().
        Field("name", builders.String().Required()).
        Field("email", builders.String().Email().Required()).
        Field("emailVerified", builders.Bool()).
        Field("profile", builders.Object()))
    
    // Esquemas del dominio orden
    registry.Register("order", "v1", legacyOrderSchema)
    registry.Register("order", "v2", modernOrderSchema)
}
```

### Composición Sensible al Entorno

Diferentes entornos pueden requerir estrategias de validación distintas:

```go
// Se componen comportamientos específicos del entorno
type EnvironmentSchemas struct {
    Development Schema
    Staging     Schema
    Production  Schema
}

func BuildEnvironmentSchemas() EnvironmentSchemas {
    base := builders.Object().
        Field("id", builders.String().Required()).
        Field("timestamp", builders.DateTime())
    
    return EnvironmentSchemas{
        // Desarrollo: Más permisivo
        Development: base.
            Field("debug", builders.Object().Optional()).
            Field("testData", builders.Bool().Optional()),
        
        // Staging: Cercano a producción
        Staging: base.
            Field("version", builders.String().Required()),
        
        // Producción: Validación estricta
        Production: base.
            Field("version", builders.String().Required()).
            Field("checksum", builders.String().Required()).
            Custom(validateIntegrity),
    }
}

// La configuración selecciona el entorno
func GetSchemaForEnvironment() Schema {
    env := os.Getenv("APP_ENV")
    schemas := BuildEnvironmentSchemas()
    
    switch env {
    case "development":
        return schemas.Development
    case "staging":
        return schemas.Staging
    default:
        return schemas.Production
    }
}
```

### Pruebas A/B de Reglas de Validación

La configuración habilita probar diferentes estrategias de validación:

```go
// Se componen estrategias de validación alternativas
type ABTestSchemas struct {
    Control    Schema  // Validación actual
    Variant    Schema  // Nueva validación a probar
    Metrics    *ValidationMetrics
}

func (ab *ABTestSchemas) ValidateWithABTest(data interface{}, userID string) error {
    // La configuración determina la participación en la prueba
    inVariant := abtest.IsUserInVariant("strict_validation_test", userID)
    
    var schema Schema
    var group string
    
    if inVariant {
        schema = ab.Variant
        group = "variant"
    } else {
        schema = ab.Control
        group = "control"
    }
    
    start := time.Now()
    err := qf.Validate(data, schema)
    duration := time.Since(start)
    
    // Registrar métricas para análisis
    ab.Metrics.Record(group, err == nil, duration)
    
    return err
}

// Configurar prueba A/B
abTest := &ABTestSchemas{
    Control: builders.Object().
        Field("email", builders.String()),  // Email opcional
        
    Variant: builders.Object().
        Field("email", builders.String().Email().Required()),  // Email válido requerido
        
    Metrics: NewValidationMetrics(),
}
```

## El Panorama Completo

### Tres Niveles de Flexibilidad

1. **Composición en Tiempo de Compilación**: Crear todos los comportamientos posibles
2. **Selección por Configuración**: Elegir qué comportamientos activar
3. **Ejecución en Tiempo de Ejecución**: Validación determinista

```
┌─────────────────────────────┐
│   Capa de Configuración    │ ← Selecciona comportamientos
├─────────────────────────────┤
│   Capa de Composición      │ ← Define comportamientos
├─────────────────────────────┤
│   Capa de Validación       │ ← Ejecuta comportamientos
└─────────────────────────────┘
```

### Beneficios de Este Enfoque

1. **Todos los comportamientos son testeados**: Como los esquemas están pre-compuestos, pueden ser probados
2. **Sin sorpresas en tiempo de ejecución**: La configuración solo selecciona, nunca crea
3. **Adopción progresiva**: Se despliegan cambios gradualmente con confianza
4. **Flexibilidad ambiental**: Diferentes reglas para diferentes contextos
5. **Rendimiento**: Sin sobrecarga de construcción de esquemas en tiempo de ejecución

### Anti-Patrones a Evitar

```go
// ❌ No se deben construir esquemas desde configuración en tiempo de ejecución
func badPattern(config map[string]interface{}) Schema {
    schema := builders.Object()
    for field, rules := range config {
        // Esto crea esquemas no testeados e impredecibles
        schema.Field(field, buildFromRules(rules))
    }
    return schema
}

// ✅ Se debe seleccionar desde esquemas pre-construidos
func goodPattern(config map[string]interface{}) Schema {
    schemaName := config["schema"].(string)
    return schemaRegistry.Get(schemaName)
}
```

## La Filosofía en la Práctica

### Lo Que Esto Habilita

1. **Sistemas Predecibles**: Una vez desplegado, el comportamiento está garantizado
2. **Lógica Testeable**: Funciones puras sin efectos secundarios
3. **Bibliotecas Componibles**: Se construyen abstracciones de alto nivel con seguridad
4. **Límites Claros**: Responsabilidades de tiempo de compilación vs tiempo de ejecución

### Lo Que Esto Previene

1. **Sorpresas en Tiempo de Ejecución**: Sin reglas de validación cambiando durante la ejecución
2. **Errores de Configuración**: Detectados en tiempo de compilación, no en producción
3. **Bugs de Mutación**: Los esquemas inmutables no pueden modificarse accidentalmente
4. **Degradación del Rendimiento**: Sin sobrecarga de construcción de esquemas en tiempo de ejecución

## Conclusión

El principio de "Componer en Tiempo de Compilación, Validar en Tiempo de Ejecución" no implica abandonar la configuración—significa utilizar la configuración sabiamente. La configuración se convierte en un mecanismo de selección para elegir entre comportamientos pre-compuestos y testeados, en lugar de una forma de crear comportamientos dinámicamente.

Este enfoque proporciona:
- **Flexibilidad mediante composición**: Se construyen todas las variaciones necesarias
- **Seguridad mediante selección**: La configuración solo elige lo que ya existe
- **Evolución mediante versionado**: Se despliegan nuevos comportamientos gradualmente
- **Confianza mediante testing**: Todos los comportamientos posibles pueden ser probados

Al combinar el poder de la composición con la flexibilidad de la configuración, Queryfy habilita sistemas que son simultáneamente:
- **Flexibles**: Se adaptan a diferentes contextos y requerimientos
- **Predecibles**: Todos los comportamientos son conocidos y testeados
- **Evolucionables**: Los cambios pueden desplegarse progresivamente
- **Performantes**: Sin sobrecarga de construcción de esquemas en tiempo de ejecución

No se trata de elegir entre configuración y código—se trata de utilizar cada herramienta para lo que mejor hace. La configuración sobresale en la selección en tiempo de ejecución; la composición sobresale en la definición de comportamiento. Juntas, crean sistemas que son tanto poderosos como manejables.

El futuro de la validación no radica en lenguajes de configuración más complejos o comportamiento más dinámico en tiempo de ejecución—radica en mejores herramientas de composición que faciliten construir los comportamientos deterministas exactos que se necesitan, combinadas con configuración simple que seleccione el comportamiento correcto para el contexto apropiado.
