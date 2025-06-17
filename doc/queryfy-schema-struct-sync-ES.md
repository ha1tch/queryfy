# Sincronización Esquema-Struct de Queryfy: Filosofía y Práctica (Mejorada)

## Índice

1. [El Desafío Central](#el-desafío-central)
2. [La Filosofía del Trasvasado](#la-filosofía-del-trasvasado)
3. [Múltiples Esquemas por Entidad](#múltiples-esquemas-por-entidad)
4. [La Estrategia de Sincronización](#la-estrategia-de-sincronización)
5. [Patrones de Implementación](#patrones-de-implementación)
6. [Verificación en Tiempo de Prueba](#verificación-en-tiempo-de-prueba)
7. [El Viaje Completo de los Datos](#el-viaje-completo-de-los-datos)
8. [Análisis de Rendimiento y Compromisos](#análisis-de-rendimiento-y-compromisos)
9. [Por Qué Este Enfoque Sobre Alternativas](#por-qué-este-enfoque-sobre-alternativas)
10. [Claridad Filosófica: Abrazando la Realidad](#claridad-filosófica-abrazando-la-realidad)
11. [Direcciones Futuras](#direcciones-futuras)
12. [Conclusión: El Arco Largo](#conclusión-el-arco-largo)

---

## El Desafío Central

En el desarrollo con Go, vivimos en dos mundos:

1. **El Mundo Dinámico**: Donde el JSON llega de fuentes externas - desordenado, sin tipos, e impredecible
2. **El Mundo Estático**: Donde los structs de Go proveen seguridad de tipos, soporte IDE, y garantías en tiempo de compilación

El desafío no es solo mover datos entre estos mundos - es hacerlo manteniendo corrección, rendimiento y cordura del desarrollador.

### Por Qué Esto Importa

Los enfoques tradicionales fuerzan decisiones incómodas:
- Unmarshal directo a structs: Panic ante datos inesperados
- Trabajar con `map[string]interface{}`: Se pierde toda seguridad de tipos
- Validación manual después del unmarshal: Redundante y propenso a errores

Queryfy introduce una tercera vía: **trasvasado controlado** - la transferencia cuidadosa de datos desde forma dinámica a estática a través de un pipeline de validación/transformación.

---

## La Filosofía del Trasvasado

"Trasvasar" - transferir líquido de un recipiente a otro - captura perfectamente lo que estamos haciendo con los datos. Como un químico transfiriendo cuidadosamente una solución a través de un filtro, estamos moviendo datos desde su forma cruda hacia su contenedor final.

### Principios Clave

1. **Los datos cambian de recipiente, no solo de estado de validación**
   ```go
   // No esto: mismos datos, diferente estado
   datosCrudos → [validar] → datosValidados (misma estructura)
   
   // Sino esto: transformados en nuevo recipiente
   datosCrudos → [parsear/transformar] → datosFormados → [trasvasar] → struct
   ```

2. **La validación es solo asegurar que el trasvasado sea posible**
   - No validamos por validar
   - Validamos porque necesitamos los datos en una forma específica
   - Si el trasvasado fallara, queremos saberlo temprano

3. **El esquema define la transformación, no solo las reglas**
   - Un esquema es una especificación para parsear
   - Describe cómo crear datos válidos, no solo verificarlos

---

## Múltiples Esquemas por Entidad

Las entidades del mundo real requieren diferentes representaciones para diferentes operaciones. Las operaciones de creación necesitan contraseñas mientras que las actualizaciones no; los esquemas de consulta usan sintaxis de filtro en lugar de estructura de entidad; los endpoints de administración exponen campos diferentes que las APIs públicas; las operaciones de importación necesitan validación permisiva mientras que las APIs demandan validación estricta. Por esto típicamente vemos:

```go
type EsquemasUsuario struct {
    Crear          Schema  // Todos los campos requeridos, incluye contraseña
    Actualizar     Schema  // Mayoría de campos opcionales, sin contraseña
    ActualizParcial Schema  // Semántica PATCH
    Consultar      Schema  // Parámetros de filtro, no campos de entidad
    VistaAdmin     Schema  // Datos completos incluyendo campos sensibles
    VistaPublica   Schema  // Subconjunto seguro para API pública
    Importar       Schema  // Validación permisiva para importaciones masivas
}
```

Este patrón reconoce que una sola entidad tiene múltiples representaciones válidas dependiendo del contexto.

---

## La Estrategia de Sincronización

### Organización y Declaración en Tiempo de Diseño

```
entidades/
├── usuario.go                // type Usuario struct {...}
├── usuario_esquemas.go       // Todos los esquemas relacionados con Usuario
└── esquema_test.go          // Pruebas de compatibilidad
```

Los esquemas viven junto a sus structs correspondientes, haciendo las relaciones explícitas:

```go
// usuario_esquemas.go
package entidades

func NuevosEsquemasUsuario() *EsquemasUsuario {
    // Constructores de campos compartidos para consistencia
    campoEmail := func(requerido bool) Schema {
        campo := builders.Transform(
            builders.String().Email()
        ).Add(transformers.Lowercase())
        
        if requerido {
            return campo.Required()
        }
        return campo.Optional()
    }
    
    return &EsquemasUsuario{
        Crear: builders.Object().
            ForStruct(Usuario{}).  // Vincula esquema a struct
            Field("email", campoEmail(true)).
            Field("contraseña", campoContraseña(true)),
            
        Actualizar: builders.Object().
            ForStruct(Usuario{}).
            Field("email", campoEmail(false)),
            // Sin campo contraseña en actualizaciones
            
        Consultar: builders.Object().
            // Sin ForStruct - no mapea a Usuario
            Field("email", builders.String().Optional()).
            Field("limite", builders.Number().Max(100)),
    }
}
```

### La Anotación ForStruct: Cómo Funciona

El método `ForStruct()` crea un vínculo crítico entre esquemas y structs:

```go
// Concepto de implementación interna
type ObjectSchema struct {
    fields       map[string]Schema
    targetStruct reflect.Type  // Establecido por ForStruct
}

func (s *ObjectSchema) ForStruct(v interface{}) *ObjectSchema {
    s.targetStruct = reflect.TypeOf(v)
    return s
}
```

#### Lo que ForStruct Habilita

1. **Almacenamiento de Información de Tipos**
   ```go
   // El esquema ahora conoce:
   // - Nombres y tipos de campos en el struct objetivo
   // - Tags JSON para mapeo de campos
   // - Si los campos son punteros (opcionales)
   ```

2. **Verificación de Compatibilidad**
   ```go
   func (s *ObjectSchema) VerifyStruct(target interface{}) error {
       targetType := reflect.TypeOf(target).Elem()
       
       // Para cada campo del esquema:
       for fieldName, fieldSchema := range s.fields {
           // Encontrar campo struct correspondiente
           structField, found := targetType.FieldByName(fieldName)
           if !found {
               // Verificar tag JSON
               structField, found = findByJSONTag(targetType, fieldName)
           }
           
           if !found {
               return fmt.Errorf("campo de esquema %q no tiene campo struct", fieldName)
           }
           
           // Verificar compatibilidad de tipos
           schemaType := fieldSchema.GetOutputType()
           if !isAssignable(schemaType, structField.Type) {
               return fmt.Errorf("campo %q: esquema produce %v, struct espera %v",
                   fieldName, schemaType, structField.Type)
           }
       }
       return nil
   }
   ```

3. **Optimización Futura de ToStruct**
   ```go
   // Con ForStruct, ToStruct puede optimizarse:
   func (qf *Queryfy) ToStruct(data interface{}, target interface{}) error {
       schema := getSchemaFor(target) // Recuperado via mapeo ForStruct
       
       // Mapeo directo de campos sin escaneo completo de reflexión
       return optimizedMapping(data, target, schema.fieldMappings)
   }
   ```

---

## Verificación en Tiempo de Prueba

En lugar de verificaciones en tiempo de ejecución que impactan el rendimiento, verificamos la compatibilidad en tiempo de prueba. Esto puede hacerse a través de pruebas generadas o patrones manuales:

```go
// esquema_test.go
func TestCompatibilidadEsquemaStruct(t *testing.T) {
    esquemas := NuevosEsquemasUsuario()
    
    // Verificar cada esquema que mapea a Usuario
    for nombre, esquema := range map[string]Schema{
        "Crear": esquemas.Crear,
        "Actualizar": esquemas.Actualizar,
        // Nota: Consultar no incluido - no mapea a Usuario
    } {
        t.Run(nombre, func(t *testing.T) {
            var usuario Usuario
            if err := esquema.VerifyStruct(&usuario); err != nil {
                t.Fatal(err)
                // Errores detallados como:
                // Campo "email": esquema produce string, struct espera int
                // Campo "telefono": campo de esquema no tiene campo struct correspondiente
            }
        })
    }
}
```

### Qué Se Verifica

1. **Compatibilidad de Tipos**: ¿Puede la salida del esquema asignarse al campo struct? ¿Son compatibles los tipos numéricos?
2. **Mapeo de Campos**: ¿Cada campo del esquema tiene un objetivo struct? ¿Los tags JSON están mapeados correctamente?
3. **Compatibilidad de Transformación**: ¿La salida transformada coincidirá con los tipos struct?

---

## El Viaje Completo de los Datos

```go
// Etapa 1 y 2: Recepción y Transformación
jsonCrudo := recibirDeAPI()
esquema := esquemas.Usuario.Crear
datosLimpios, err := esquema.ValidateAndTransform(jsonCrudo)
// datosLimpios ahora tiene estructura validada, valores transformados y tipos correctos

// Etapa 3 y 4: Inspección y Trasvasado
email := datosLimpios["email"].(string)  // Seguro - esquema garantiza string
datosLimpios["creadoEn"] = time.Now()    // Agregar campos calculados si es necesario

var usuario Usuario
err = qf.ToStruct(datosLimpios, &usuario)
// Éxito garantizado si esquema.VerifyStruct() pasó en las pruebas

// Etapa 5: Lógica de Negocio
usuario.ID = generarID()
err = db.Save(&usuario)
```

Cada etapa sirve un propósito específico en mover desde datos externos no confiables hacia estructuras internas confiables.

---

## Análisis de Rendimiento y Compromisos

### Cuando Mantenemos Cero Asignaciones

Basándonos en nuestras sesiones de benchmark, Queryfy + Superjsonic mantiene cero asignaciones durante:

1. **Fase de parseo de tokens**: Lectura y tokenización de JSON
2. **Validación estructural**: Verificar que la estructura JSON coincida con el esquema
3. **Validación de tipos simple**: Verificar strings, números, booleanos

### Cuando Asignamos Memoria

La asignación de memoria ocurre durante:

1. **Fase de transformación**: Crear nuevas estructuras de datos limpias
   ```go
   // Ejemplo: Transformar email
   "  JUAN@EJEMPLO.COM  " → "juan@ejemplo.com"  // Nueva string asignada
   ```

2. **Creación de map/slice para resultados**: Construir el mapa de datos limpios
   ```go
   datosLimpios := make(map[string]interface{})  // Asignación
   ```

3. **Fase ToStruct**: Convertir a forma struct final

### Características de Rendimiento por Tamaño de Carga

**Cargas pequeñas (<10KB)**: 
- Sobrecarga de memoria despreciable
- Costo de transformación mínimo
- Adecuado para APIs REST, microservicios

**Cargas medianas (10KB-1MB)**:
- Memoria se duplica temporalmente durante transformación
- Aún eficiente para la mayoría de casos de uso
- Considerar streaming para arrays

**Cargas grandes (>1MB)**:
- La presión de memoria se vuelve significativa
- Investigación actual en estrategias de pooling
- Patrones de procesamiento por lotes en desarrollo

### Optimizaciones Futuras: Investigación de Procesamiento por Lotes

Estamos investigando activamente estrategias para mantener casi cero asignaciones incluso durante la transformación de grandes conjuntos de datos. La idea clave es que el procesamiento por lotes tiene características diferentes al procesamiento de documentos individuales:

```go
// Concepto: ForStructBatch para procesamiento de alto volumen
esquema := builders.Object().
    ForStructBatch(Usuario{}, OpcionesBatch{
        TamañoPool: 1000,
        ReutilizarAsignaciones: true,
    })

// Esto habilitaría:
procesador := qf.NewProcesadorBatch(esquema)
for _, loteJSON := range conjuntoDatosEnorme {
    usuarios := procesador.ProcesarLote(loteJSON)
    // Reutiliza asignaciones de memoria entre lotes
}
```

#### La Estrategia de Procesamiento por Lotes

Al procesar millones de documentos JSON pequeños (piense en ingestión de logs, flujos de eventos, importaciones masivas), el patrón de asignación cambia:

1. **Pre-asignar buffers de transformación** basados en el tamaño esperado del documento
2. **Reutilizar estructuras map** - limpiar y rellenar en lugar de asignar nuevo
3. **Pool de buffers string** para transformaciones como minúsculas/trim
4. **Amortizar costo de asignación** entre miles de documentos

Esta investigación está en curso porque queremos asegurar:
- La API permanece simple para casos comunes
- Las optimizaciones por lotes no complican el uso de documentos individuales
- El pooling de memoria no introduce problemas de concurrencia
- Los beneficios justifican la complejidad adicional

Estamos aprendiendo de las exitosas estrategias de pooling de Superjsonic y explorando cómo aplicar principios similares a la fase de transformación.

---

## Por Qué Este Enfoque Sobre Alternativas

### ¿Por Qué No Generar Esquemas desde Structs?

```go
// El intento:
type Usuario struct {
    Email string `json:"email" validate:"email,required"`
}
```

**Los problemas**: Los tags de struct no pueden expresar transformaciones (¿cómo etiquetas "minúsculas y trim"?), no pueden manejar datos desordenados del mundo real (API envía "25" para un campo int), y no pueden representar diferentes operaciones (crear necesita email requerido, actualizar necesita opcional).

### ¿Por Qué No Arreglar Datos Después del Unmarshal?

```go
// El intento:
json.Unmarshal(data, &usuario)
usuario.Email = strings.ToLower(usuario.Email)
```

**Los problemas**: Unmarshal podría hacer panic con tipos incorrectos, la lógica de transformación se dispersa por el código base, y no puedes validar hasta después de potencialmente corromper tus structs.

### ¿Por Qué No Separar Validación y Transformación?

```go
// El intento:
validar(datos) → transformar(datos) → unmarshal(datos)
```

**Los problemas**: Múltiples pasadas dañan el rendimiento, validación y transformación están naturalmente entrelazadas (email válido incluye normalización), y los mensajes de error no pueden sugerir qué transformación arreglaría los problemas.

**La idea clave**: La validación nunca es el objetivo final. El objetivo es obtener datos en una forma que puedas usar. Queryfy reconoce esto tratando validación y transformación como una operación.

---

## Claridad Filosófica: Abrazando la Realidad

### La Falsa Dicotomía

Muchos sistemas intentan mantener separación rígida:
- "Los esquemas no deberían saber nada sobre structs"
- "La validación nunca debería modificar datos"
- "Tipado dinámico y estático no deberían mezclarse"

Esta pureza ideológica crea problemas prácticos.

### La Filosofía Pragmática de Queryfy

**Abrazamos la tensión natural** entre JSON dinámico y structs estáticos. Esta tensión no es un problema a resolver - es una realidad con la que trabajar.

1. **Los esquemas conocen los structs cuando es útil** (ForStruct)
   - Pero los structs no dependen de esquemas
   - El acoplamiento unidireccional es intencional y saludable

2. **Validación y transformación son una operación**
   - Porque eso es lo que realmente necesitas
   - Separarlas es artificial

3. **Múltiples representaciones son normales**
   - Un struct, muchos esquemas
   - Diferentes vistas para diferentes contextos

### Trabajando con la Realidad

El mundo real nos presenta datos desordenados e inconsistentes donde:
- Las APIs envían números como strings
- Las fechas vienen en 15 formatos
- Los números de teléfono son caos
- Los campos requeridos a veces faltan

La filosofía no se trata de afirmar tener todas las respuestas. Estamos explorando soluciones prácticas a problemas reales, aprendiendo de cada implementación, y ajustando nuestro enfoque basado en lo que descubrimos. Esta humildad está incorporada en el diseño de Queryfy - proveemos escapes, múltiples enfoques, y reconocemos que diferentes escenarios necesitan diferentes soluciones.

---

## Direcciones Futuras

### Constructores de Esquema Genéricos (Go 1.18+)
```go
esquema := builders.ObjectFor[Usuario]().
    Field("email", builders.String().Email())
// Error de compilación si "email" no existe en Usuario
```

### Generación Automática de Esquema con Mejora
```go
esquemaBase := qf.GenerateSchema(Usuario{})
// Luego mejorar lo que los tags no pueden expresar:
esquema := esquemaBase.
    FieldTransform("email", transformers.Lowercase()).
    FieldValidation("telefono", validators.NumeroTelefono("US"))
```

### Integración con IDE
- Plugin que muestra mapeos esquema-struct
- Advertencias para cambios incompatibles
- Auto-generar pruebas de compatibilidad

### Optimizaciones de Rendimiento
- Generación de código para ToStruct para evitar reflexión
- Trasvasado con streaming para grandes conjuntos de datos
- Procesamiento paralelo para campos array

---

## Conclusión: El Arco Largo

En el arco largo de la filosofía de procesamiento de datos de Queryfy, la sincronización esquema-struct representa el puente final entre dos mundos que han sido artificialmente separados en la mayoría de bibliotecas de validación.

### El Viaje que Habilitamos

1. **Recibir** datos desordenados del mundo real
2. **Parsear** a través de validación hacia forma limpia
3. **Transformar** durante el parseo para eficiencia
4. **Verificar** que el resultado coincida con expectativas
5. **Trasvasar** hacia estructuras con seguridad de tipos
6. **Procesar** con confianza en lógica de negocio

### La Innovación Clave

Al tratar la validación como parseo y la transformación como parte de ese parseo, hemos creado un sistema donde:
- **Los esquemas son participantes activos** en formar datos, no validadores pasivos
- **Las pruebas garantizan seguridad en producción** a través de verificación en tiempo de compilación y prueba
- **Múltiples esquemas por entidad** reconocen la complejidad del mundo real
- **El trasvasado es seguro** porque la compatibilidad se verifica aguas arriba

### La Filosofía Realizada

"Parsea, no valides" alcanza su expresión completa cuando el pipeline de parseo termina con datos con seguridad de tipos en structs Go apropiados. No solo hemos validado los datos - los hemos transformado, limpiado, y colocado de forma segura en su recipiente final.

Esto no es solo sobre hacer el manejo de JSON más seguro o rápido. Es sobre traer todo el poder del sistema de tipos de Go para procesar datos del mundo real mientras se reconoce que el mundo real es desordenado, inconsistente y cambiante.

### La Idea Final

En enfoques tradicionales, validas datos para probar que coinciden con tus structs. En el enfoque de Queryfy, parseas datos en formas que encajan en tus structs. Esta inversión - de verificar a crear - es lo que hace al sistema tanto poderoso como ergonómico.

El arco largo se inclina hacia la seguridad, pero llega allí a través de transformación, no restricción. Al abrazar la realidad de datos desordenados en lugar de desearla lejos, Queryfy provee un camino práctico desde el caos de APIs externas hacia la seguridad del sistema de tipos de Go.

Esa es la forma de Queryfy: encontrar los datos donde están, darles forma en lo que necesitas, y entregarlos de forma segura a su destino.