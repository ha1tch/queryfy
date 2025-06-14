# Filosofía Queryfy: Ramificaciones y Patrones Prácticos

**La filosofía central de Queryfy—"Componer en Tiempo de Compilación, Validar en Tiempo de Ejecución"—conlleva profundas implicaciones para la arquitectura, despliegue y mantenimiento de sistemas de validación. Este documento explora las ramificaciones prácticas de esta filosofía en diferentes escalas y contextos, desde aplicaciones individuales hasta sistemas distribuidos.**
## Ramificaciones Arquitectónicas

### Arquitectura de Microservicios

La garantía de inmutabilidad transforma fundamentalmente cómo funciona la validación en sistemas distribuidos:

```go
// shared-schemas/v1/order.go
package schemas

var OrderSchemaV1 = builders.Object().
    Field("orderId", builders.String().UUID().Required()).
    Field("items", builders.Array().MinItems(1).Required()).
    Field("total", builders.Number().Min(0).Required())

// Servicio A: Creación de Órdenes
func (s *OrderService) CreateOrder(data []byte) error {
    var orderData map[string]interface{}
    json.Unmarshal(data, &orderData)
    
    if err := qf.Validate(orderData, schemas.OrderSchemaV1); err != nil {
        return err
    }
    // Procesar orden...
}

// Servicio B: Procesamiento de Órdenes
func (s *ProcessingService) ProcessOrder(data []byte) error {
    var orderData map[string]interface{}
    json.Unmarshal(data, &orderData)
    
    // Mismo esquema, validación garantizada idéntica
    if err := qf.Validate(orderData, schemas.OrderSchemaV1); err != nil {
        return err
    }
    // Procesar orden...
}
```

**Ramificación**: Los servicios pueden compartir lógica de validación sin acoplamiento estrecho. El paquete de esquemas se convierte en una definición de contrato.

### Patrón API Gateway

La filosofía habilita estrategias sofisticadas de validación en el borde:

```go
// api-gateway/validator.go
type ValidationStrategy struct {
    routes map[string]RouteValidation
}

type RouteValidation struct {
    versions map[string]queryfy.Schema
    rollout  RolloutStrategy
}

func (v *ValidationStrategy) ValidateRequest(r *http.Request) error {
    route := extractRoute(r)
    version := extractVersion(r)
    userID := extractUserID(r)
    
    routeValidation := v.routes[route]
    
    // Despliegue progresivo de nueva validación
    schema := routeValidation.rollout.SelectSchema(
        userID,
        version,
        routeValidation.versions,
    )
    
    body, _ := ioutil.ReadAll(r.Body)
    var data map[string]interface{}
    json.Unmarshal(body, &data)
    
    return qf.Validate(data, schema)
}

// Inicialización - todos los esquemas pre-compuestos
func initGateway() *ValidationStrategy {
    return &ValidationStrategy{
        routes: map[string]RouteValidation{
            "/api/orders": {
                versions: map[string]queryfy.Schema{
                    "v1": orderSchemaV1,
                    "v2": orderSchemaV2,
                    "v2-strict": orderSchemaV2Strict,
                },
                rollout: &PercentageRollout{
                    "v1": 60,
                    "v2": 35,
                    "v2-strict": 5,
                },
            },
        },
    }
}
```

**Ramificación**: El API gateway se convierte en un punto de aplicación de políticas sin convertirse en un cuello de botella.

### Arquitectura Dirigida por Eventos

En sistemas dirigidos por eventos, la inmutabilidad de esquemas habilita el versionado de eventos:

```go
// Esquemas de eventos con compatibilidad garantizada
type EventSchemas struct {
    UserCreated map[string]queryfy.Schema
    OrderPlaced map[string]queryfy.Schema
}

var eventSchemas = EventSchemas{
    UserCreated: map[string]queryfy.Schema{
        "v1": builders.Object().
            Field("userId", builders.String().Required()).
            Field("email", builders.String().Email().Required()),
        
        "v2": builders.Object().
            Field("userId", builders.String().UUID().Required()).
            Field("email", builders.String().Email().Required()).
            Field("source", builders.String().Required()),
    },
}

// Procesador de eventos con detección de versión
func processEvent(event Event) error {
    eventType := event.Type
    version := event.Version
    
    schemas, exists := eventSchemas[eventType]
    if !exists {
        return fmt.Errorf("tipo de evento desconocido: %s", eventType)
    }
    
    schema, exists := schemas[version]
    if !exists {
        // Retroceder a la última versión compatible
        schema = findCompatibleSchema(schemas, version)
    }
    
    return qf.Validate(event.Data, schema)
}
```

**Ramificación**: La evolución de eventos se vuelve manejable con límites claros de versión.

## Ramificaciones Operacionales

### Estrategias de Despliegue

La filosofía habilita varios patrones de despliegue:

#### Despliegues Blue-Green con Cambios de Validación

```go
// Configuración de despliegue
type DeploymentConfig struct {
    Blue  SchemaSet
    Green SchemaSet
}

func (d *DeploymentConfig) GetActiveSchemas() SchemaSet {
    if isGreenActive() {
        return d.Green
    }
    return d.Blue
}

// Validación pre-despliegue
func validateDeployment(config DeploymentConfig) error {
    testData := loadTestDataSet()
    
    for _, data := range testData {
        blueResult := validate(data, config.Blue)
        greenResult := validate(data, config.Green)
        
        if !areCompatible(blueResult, greenResult) {
            return fmt.Errorf("validación incompatible entre blue/green")
        }
    }
    
    return nil
}
```

#### Despliegues Canary

```go
type CanaryValidator struct {
    stable queryfy.Schema
    canary queryfy.Schema
    meter  metrics.Meter
}

func (cv *CanaryValidator) Validate(data interface{}, userID string) error {
    // Ejecutar ambas validaciones
    stableErr := qf.Validate(data, cv.stable)
    canaryErr := qf.Validate(data, cv.canary)
    
    // Métricas para comparación
    if stableErr == nil && canaryErr != nil {
        cv.meter.Counter("validation.canary.stricter").Inc()
    } else if stableErr != nil && canaryErr == nil {
        cv.meter.Counter("validation.canary.looser").Inc()
    }
    
    // Usar estable para validación real
    if inCanaryGroup(userID) && canaryErr == nil {
        return canaryErr
    }
    
    return stableErr
}
```

**Ramificación**: Los cambios de validación pueden desplegarse con la misma confianza que los cambios de código.

### Monitoreo y Observabilidad

Los esquemas pre-compuestos habilitan observabilidad profunda:

```go
type ValidationMetrics struct {
    histogram *prometheus.HistogramVec
    counter   *prometheus.CounterVec
}

func instrumentedValidate(data interface{}, schema queryfy.Schema) error {
    start := time.Now()
    
    // Identificación del esquema
    schemaID := getSchemaID(schema)
    
    err := qf.Validate(data, schema)
    
    duration := time.Since(start)
    
    // Métricas enriquecidas
    labels := prometheus.Labels{
        "schema":  schemaID,
        "version": getSchemaVersion(schema),
        "success": strconv.FormatBool(err == nil),
    }
    
    metrics.histogram.With(labels).Observe(duration.Seconds())
    metrics.counter.With(labels).Inc()
    
    if err != nil {
        logValidationError(schemaID, err, data)
    }
    
    return err
}
```

**Ramificación**: La validación se convierte en un componente del sistema observable de primera clase.

## Ramificaciones del Flujo de Trabajo de Desarrollo

### Ciclo de Vida del Desarrollo de Esquemas

La filosofía sugiere un flujo de trabajo de desarrollo específico:

```go
// 1. Desarrollo: Componer nuevo esquema
func developFeature() {
    newSchema := builders.Object().
        Field("newField", builders.String().Required()).
        Merge(existingSchema)
    
    // 2. Testing: Validar contra muestra de datos de producción
    testWithProductionData(newSchema)
    
    // 3. Staging: Desplegar en entorno de staging
    stagingSchemas["feature-x"] = newSchema
    
    // 4. Producción: Despliegue progresivo
    productionRollout.AddVariant("feature-x", newSchema, 5) // 5% del tráfico
}

// Proceso de revisión de cambios de esquema
type SchemaChange struct {
    ID          string
    Description string
    Before      queryfy.Schema
    After       queryfy.Schema
    Impact      ImpactAnalysis
}

func (sc *SchemaChange) GenerateReport() ChangeReport {
    return ChangeReport{
        AddedFields:      sc.findAddedFields(),
        RemovedFields:    sc.findRemovedFields(),
        TightenedRules:   sc.findTightenedRules(),
        RelaxedRules:     sc.findRelaxedRules(),
        BackwardCompat:   sc.isBackwardCompatible(),
        MigrationNeeded:  sc.requiresMigration(),
    }
}
```

### Integración CI/CD

```yaml
# .github/workflows/schema-validation.yml
name: Schema Validation

on: [push, pull_request]

jobs:
  validate-schemas:
    steps:
      - name: Compile Schemas
        run: go test ./schemas/...
      
      - name: Backward Compatibility Check
        run: |
          go run ./tools/schema-compat-check \
            --baseline main \
            --proposed ${{ github.sha }}
      
      - name: Performance Regression Test
        run: |
          go test -bench=. ./schemas/... | \
          go run ./tools/bench-compare --threshold 10%
      
      - name: Schema Coverage Report
        run: |
          go run ./tools/schema-coverage \
            --data ./testdata \
            --schemas ./schemas
```

**Ramificación**: Los cambios de esquema adquieren el mismo rigor que los cambios de código.

## Ramificaciones de Rendimiento

### Optimización en Tiempo de Compilación

Los esquemas pre-compuestos habilitan optimización agresiva:

```go
// Oportunidades de optimización
type OptimizedSchema struct {
    original      queryfy.Schema
    compiled      *compiledSchema
    accessPaths   map[string][]int
    requiredMask  uint64
}

func compileSchema(schema queryfy.Schema) *OptimizedSchema {
    opt := &OptimizedSchema{
        original: schema,
    }
    
    // Pre-compilar patrones regex
    opt.compilePatterns()
    
    // Pre-calcular rutas de acceso a campos
    opt.calculateAccessPaths()
    
    // Máscara de bits para verificación de campos requeridos
    opt.buildRequiredMask()
    
    return opt
}

// Validación en tiempo de ejecución - ruta optimizada
func (opt *OptimizedSchema) FastValidate(data map[string]interface{}) error {
    // Verificación rápida de campos requeridos usando operaciones de bits
    if !opt.checkRequiredFields(data) {
        return opt.detailedRequiredCheck(data)
    }
    
    // Usar rutas pre-compiladas
    for field, path := range opt.accessPaths {
        value := opt.fastAccess(data, path)
        if err := opt.validateField(field, value); err != nil {
            return err
        }
    }
    
    return nil
}
```

### Eficiencia de Memoria

```go
// Pooling de esquemas para escenarios de alto rendimiento
var schemaPool = sync.Pool{
    New: func() interface{} {
        return &ValidationContext{
            path:   make([]string, 0, 10),
            errors: make([]FieldError, 0, 5),
        }
    },
}

func pooledValidate(data interface{}, schema queryfy.Schema) error {
    ctx := schemaPool.Get().(*ValidationContext)
    defer func() {
        ctx.Reset()
        schemaPool.Put(ctx)
    }()
    
    return schema.Validate(data, ctx)
}
```

**Ramificación**: La validación puede lograr casi cero asignaciones en rutas críticas.

## Ramificaciones de Testing

### Testing Basado en Propiedades

Los esquemas inmutables habilitan poderosas estrategias de testing:

```go
func TestSchemaProperties(t *testing.T) {
    quick.Check(func(data map[string]interface{}) bool {
        // Propiedad 1: La validación es determinista
        result1 := qf.Validate(data, schema)
        result2 := qf.Validate(data, schema)
        return reflect.DeepEqual(result1, result2)
    }, nil)
    
    quick.Check(func(data map[string]interface{}) bool {
        // Propiedad 2: Datos válidos permanecen válidos después de ida y vuelta
        if err := qf.Validate(data, schema); err == nil {
            marshaled, _ := json.Marshal(data)
            var unmarshaled map[string]interface{}
            json.Unmarshal(marshaled, &unmarshaled)
            return qf.Validate(unmarshaled, schema) == nil
        }
        return true
    }, nil)
}
```

### Fuzzing

```go
func FuzzValidation(f *testing.F) {
    // Sembrar con casos extremos conocidos
    f.Add([]byte(`{"age": -1}`))
    f.Add([]byte(`{"age": 999999999999}`))
    f.Add([]byte(`{"email": "not-an-email"}`))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        var parsed map[string]interface{}
        if err := json.Unmarshal(data, &parsed); err != nil {
            return // Omitir JSON inválido
        }
        
        // No debe entrar en pánico
        _ = qf.Validate(parsed, schema)
    })
}
```

**Ramificación**: La lógica de validación se vuelve altamente testeable y verificable.

## Ramificaciones de Equipo y Organizacionales

### Colaboración Entre Equipos

La filosofía habilita nuevos patrones de colaboración:

```go
// Repositorio central de esquemas
// schemas-repo/catalog/user/v1/schema.go
package user

var V1 = builders.Object().
    Field("id", builders.String().UUID()).
    Field("email", builders.String().Email()).
    Field("profile", ProfileV1)

// Equipo A: Usa para validación de API
// Equipo B: Usa para validación de base de datos
// Equipo C: Usa para validación de eventos

// Gobernanza de esquemas
type SchemaOwnership struct {
    Domain  string
    Team    string
    Schemas map[string]SchemaVersion
}

var ownership = []SchemaOwnership{
    {
        Domain: "user",
        Team:   "identity-team",
        Schemas: map[string]SchemaVersion{
            "user":    {Current: "v2", Deprecated: []string{"v1"}},
            "profile": {Current: "v1"},
        },
    },
}
```

### Generación de Documentación

Los esquemas pre-compuestos pueden generar documentación:

```go
func generateOpenAPISpec(schema queryfy.Schema) openapi.Schema {
    return schemaToOpenAPI(schema)
}

func generateMarkdownDocs(schemas map[string]queryfy.Schema) string {
    var docs strings.Builder
    
    for name, schema := range schemas {
        docs.WriteString(fmt.Sprintf("## %s\n\n", name))
        docs.WriteString(generateSchemaTable(schema))
        docs.WriteString(generateExamples(schema))
    }
    
    return docs.String()
}
```

**Ramificación**: Los esquemas se convierten en la fuente de verdad para la documentación de API.

## Patrones de Evolución y Migración

### Estrategia de Versionado de Esquemas

```go
type SchemaEvolution struct {
    versions   []VersionedSchema
    migrations map[string]Migration
}

type VersionedSchema struct {
    Version string
    Schema  queryfy.Schema
    Since   time.Time
    Until   *time.Time
}

type Migration func(oldData interface{}) (newData interface{}, err error)

// Migración automática
func (se *SchemaEvolution) ValidateWithMigration(
    data interface{}, 
    targetVersion string,
) (interface{}, error) {
    // Intentar validación directa
    if err := se.versions[targetVersion].Schema.Validate(data); err == nil {
        return data, nil
    }
    
    // Encontrar versión fuente válida
    sourceVersion := se.findValidVersion(data)
    if sourceVersion == "" {
        return nil, fmt.Errorf("no se encontró versión de esquema válida")
    }
    
    // Aplicar migraciones
    migrated := data
    for _, migration := range se.getMigrationPath(sourceVersion, targetVersion) {
        var err error
        migrated, err = migration(migrated)
        if err != nil {
            return nil, fmt.Errorf("migración falló: %w", err)
        }
    }
    
    // Validar datos migrados
    if err := se.versions[targetVersion].Schema.Validate(migrated); err != nil {
        return nil, fmt.Errorf("migración produjo datos inválidos: %w", err)
    }
    
    return migrated, nil
}
```

### Patrones de Deprecación

```go
type DeprecationManager struct {
    deprecated map[string]DeprecationInfo
    metrics    *DeprecationMetrics
}

type DeprecationInfo struct {
    Field       string
    Since       time.Time
    Sunset      time.Time
    Alternative string
}

func (dm *DeprecationManager) ValidateWithWarnings(
    data interface{},
    schema queryfy.Schema,
) (*ValidationResult, []Warning) {
    result := &ValidationResult{}
    warnings := []Warning{}
    
    // Validación estándar
    result.Error = qf.Validate(data, schema)
    
    // Verificar uso de campos deprecados
    for field, info := range dm.deprecated {
        if hasField(data, field) {
            dm.metrics.RecordUsage(field)
            
            warning := Warning{
                Field:   field,
                Message: fmt.Sprintf(
                    "El campo '%s' está deprecado desde %s y será eliminado el %s. Use '%s' en su lugar.",
                    field, info.Since, info.Sunset, info.Alternative,
                ),
            }
            warnings = append(warnings, warning)
        }
    }
    
    return result, warnings
}
```

**Ramificación**: La evolución de esquemas se vuelve manejable y medible.

## Anti-Patrones y Trampas

### Anti-Patrón: Mutación de Esquemas en Tiempo de Ejecución

```go
// ❌ INCORRECTO: Modificar esquemas en tiempo de ejecución
func badHandler(w http.ResponseWriter, r *http.Request) {
    schema := getSchema()
    
    // ¡No hacer esto!
    if r.Header.Get("X-Strict-Mode") == "true" {
        schema.(*ObjectSchema).Field("extra", String().Required())
    }
    
    // Esto rompe la garantía fundamental
    validate(r.Body, schema)
}

// ✅ CORRECTO: Seleccionar desde esquemas pre-compuestos
func goodHandler(w http.ResponseWriter, r *http.Request) {
    var schema queryfy.Schema
    
    if r.Header.Get("X-Strict-Mode") == "true" {
        schema = strictSchema
    } else {
        schema = normalSchema
    }
    
    validate(r.Body, schema)
}
```

### Anti-Patrón: Generación de Esquemas desde Entrada de Usuario

```go
// ❌ INCORRECTO: Construir esquemas desde entrada no confiable
func badValidation(rules []byte) error {
    var schemaConfig map[string]interface{}
    json.Unmarshal(rules, &schemaConfig)
    
    // ¡Nunca construir esquemas desde entrada de usuario!
    schema := buildSchemaFromConfig(schemaConfig)
    return validate(data, schema)
}

// ✅ CORRECTO: Mapear entrada de usuario a esquemas pre-definidos
func goodValidation(schemaName string) error {
    schema, exists := approvedSchemas[schemaName]
    if !exists {
        return fmt.Errorf("esquema desconocido: %s", schemaName)
    }
    
    return validate(data, schema)
}
```

**Ramificación**: La seguridad y estabilidad requieren gestión disciplinada de esquemas.

## Posibilidades Futuras

### Esquema como Servicio (SaaS)

La garantía de inmutabilidad habilita gestión centralizada de esquemas:

```go
// API de servicio de esquemas
type SchemaService interface {
    GetSchema(ctx context.Context, domain, version string) (queryfy.Schema, error)
    ListVersions(ctx context.Context, domain string) ([]Version, error)
    ValidateRemote(ctx context.Context, data interface{}, domain, version string) error
}

// Caché del lado del cliente
type CachedSchemaClient struct {
    client SchemaService
    cache  *lru.Cache
    ttl    time.Duration
}

func (c *CachedSchemaClient) GetSchema(ctx context.Context, domain, version string) (queryfy.Schema, error) {
    key := fmt.Sprintf("%s:%s", domain, version)
    
    if cached, ok := c.cache.Get(key); ok {
        return cached.(queryfy.Schema), nil
    }
    
    schema, err := c.client.GetSchema(ctx, domain, version)
    if err != nil {
        return nil, err
    }
    
    c.cache.SetWithTTL(key, schema, c.ttl)
    return schema, nil
}
```

### Evolución de Esquemas Asistida por IA

```go
type SchemaAdvisor struct {
    analyzer *DataAnalyzer
    proposer *SchemaProposer
}

func (sa *SchemaAdvisor) SuggestEvolution(
    currentSchema queryfy.Schema,
    recentData []interface{},
) []Suggestion {
    // Analizar fallas recientes de validación
    patterns := sa.analyzer.FindPatterns(currentSchema, recentData)
    
    // Proponer modificaciones de esquema
    suggestions := sa.proposer.GenerateSuggestions(patterns)
    
    // Cada sugerencia sigue siendo un esquema pre-compuesto
    return suggestions
}

type Suggestion struct {
    Reason      string
    NewSchema   queryfy.Schema
    Impact      Impact
    Migration   Migration
}
```

### Compilador de Optimización de Esquemas

```go
// Futuro: Compilar esquemas a código de validación optimizado
func CompileToGo(schema queryfy.Schema, packageName string) string {
    compiler := &SchemaCompiler{
        Package: packageName,
        Imports: []string{"fmt", "regexp"},
    }
    
    return compiler.Compile(schema)
}

// Ejemplo de código generado
/*
func ValidateOrder(data map[string]interface{}) error {
    // Código de validación optimizado
    if id, ok := data["id"].(string); !ok || id == "" {
        return fmt.Errorf("id: campo requerido faltante")
    }
    
    // Regex pre-compilado
    if !orderIDRegex.MatchString(id) {
        return fmt.Errorf("id: formato inválido")
    }
    
    // ... resto de la validación
}
*/
```

**Ramificación**: La filosofía abre puertas a herramientas avanzadas y optimización.

## Conclusión

La filosofía de Queryfy de "Componer en Tiempo de Compilación, Validar en Tiempo de Ejecución" conlleva ramificaciones que se extienden mucho más allá de la biblioteca misma. Sugiere una nueva forma de pensar sobre la validación en sistemas distribuidos:

1. **Validación como Código**: Los esquemas son artefactos de código que merecen el mismo rigor que el código de aplicación
2. **Inmutabilidad como Característica**: La inmutabilidad en tiempo de ejecución habilita mejor testing, monitoreo y despliegue
3. **Composición sobre Configuración**: El comportamiento complejo emerge de piezas simples y componibles
4. **Evolución Progresiva**: Los sistemas pueden evolucionar con seguridad mediante versionado y despliegue controlado

Estas ramificaciones transforman la validación de un mal necesario en una herramienta poderosa para construir sistemas robustos y evolucionables. La filosofía proporciona una base para tratar la validación como una preocupación arquitectónica de primera clase, digna de la misma atención que otorgamos a otros componentes críticos del sistema.

El futuro de la validación no se trata solo de capturar datos incorrectos—se trata de construir sistemas que puedan adaptarse y evolucionar mientras mantienen garantías sobre su comportamiento. La filosofía de Queryfy señala el camino hacia ese futuro.