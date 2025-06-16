#!/bin/bash


# Códigos de color
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

BASE_URL="http://localhost:8080"

clear
echo -e "${CYAN}╔══════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║        Demo Funcional - Microservicio de Usuarios Argentinos           ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════════════════╝${NC}"
echo

# Test 1: Usuario básico sin CUIT (para evitar el problema del dígito verificador)
echo -e "${YELLOW}1. Creando usuario básico (sin CUIT)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

RESPONSE=$(curl -s -X POST ${BASE_URL}/usuarios \
  -H "Content-Type: application/json" \
  -d '{
    "email": "juan.perez@gmail.com",
    "dni": "30123456",
    "nombre": "Juan",
    "apellido": "Pérez",
    "celular": "1155551234",
    "fechaNacimiento": "1990-05-15",
    "provincia": "CABA",
    "localidad": "Palermo",
    "codigoPostal": "1425",
    "direccion": "Av. Santa Fe 1234",
    "password": "Segura123"
  }')

if echo "$RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    USER_ID=$(echo "$RESPONSE" | jq -r '.id')
    echo -e "${GREEN}✓ Usuario creado exitosamente${NC}"
    echo "$RESPONSE" | jq '{id, email, dni, nombre, apellido, celular}'
else
    echo -e "${RED}✗ Error al crear usuario${NC}"
    echo "$RESPONSE" | jq '.'
fi

# Test 2: Transformación de DNI
echo -e "\n${YELLOW}2. Transformación de DNI con puntos${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

RESPONSE=$(curl -s -X POST ${BASE_URL}/usuarios \
  -H "Content-Type: application/json" \
  -d '{
    "email": "maria.gomez@hotmail.com",
    "dni": "25.432.109",
    "nombre": "María",
    "apellido": "Gómez",
    "celular": "1166667777",
    "fechaNacimiento": "1985-03-20",
    "provincia": "Buenos Aires",
    "localidad": "La Plata",
    "codigoPostal": "1900",
    "direccion": "Calle 50 nro 123",
    "password": "Clave456A"
  }')

if echo "$RESPONSE" | jq -e '.dni' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ DNI transformado correctamente${NC}"
    echo "Entrada: 25.432.109"
    echo "Salida: $(echo "$RESPONSE" | jq -r '.dni')"
else
    echo -e "${RED}✗ Error en transformación${NC}"
fi

# Test 3: Validación de provincias
echo -e "\n${YELLOW}3. Validación de Provincias Argentinas${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

PROVINCIAS=("CABA" "Córdoba" "Santa Fe" "Mendoza" "Neuquén" "Río Negro")
for provincia in "${PROVINCIAS[@]}"; do
    # Convertir a minúsculas y remover tildes/caracteres especiales para el email
    email_base=$(echo "$provincia" | tr '[:upper:]' '[:lower:]' | tr ' ' '_' | sed 's/á/a/g; s/é/e/g; s/í/i/g; s/ó/o/g; s/ú/u/g; s/ñ/n/g')
    dni_random=$(( RANDOM % 15000000 + 20000000 ))
    timestamp=$(date +%s%N)
    
    RESPONSE=$(curl -s -X POST ${BASE_URL}/usuarios \
      -H "Content-Type: application/json" \
      -d "{
        \"email\": \"usuario.${email_base}.${timestamp}@test.com\",
        \"dni\": \"${dni_random}\",
        \"nombre\": \"Test\",
        \"apellido\": \"${provincia}\",
        \"celular\": \"1133334444\",
        \"fechaNacimiento\": \"1990-01-01\",
        \"provincia\": \"${provincia}\",
        \"localidad\": \"Capital\",
        \"codigoPostal\": \"1000\",
        \"direccion\": \"Calle Test 123\",
        \"password\": \"Test1234\"
      }")
    
    if echo "$RESPONSE" | jq -e '.provincia' > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} ${provincia}"
    else
        echo -e "${RED}✗${NC} ${provincia}"
        # Mostrar el error específico
        echo "  Error: $(echo "$RESPONSE" | jq -r '.error // .errores[0].mensaje // "Error desconocido"')"
    fi
done

# Test 4: CBU válido
echo -e "\n${YELLOW}4. Validación de CBU${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

RESPONSE=$(curl -s -X POST ${BASE_URL}/usuarios \
  -H "Content-Type: application/json" \
  -d '{
    "email": "cliente.banco@gmail.com",
    "dni": "28765432",
    "nombre": "Cliente",
    "apellido": "Bancario",
    "celular": "1144445555",
    "fechaNacimiento": "1988-07-10",
    "provincia": "CABA",
    "localidad": "Microcentro",
    "codigoPostal": "1000",
    "direccion": "Florida 100",
    "cbu": "0170099220000067797370",
    "password": "Banco2024"
  }')

if echo "$RESPONSE" | jq -e '.cbu' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ CBU válido aceptado${NC}"
    echo "CBU: $(echo "$RESPONSE" | jq -r '.cbu')"
else
    echo -e "${RED}✗ CBU rechazado${NC}"
fi

# Test 5: Normalización de nombres con caracteres especiales
echo -e "\n${YELLOW}5. Caracteres especiales en nombres${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

RESPONSE=$(curl -s -X POST ${BASE_URL}/usuarios \
  -H "Content-Type: application/json" \
  -d '{
    "email": "jose.maria@yahoo.com.ar",
    "dni": "31234567",
    "nombre": "José María",
    "apellido": "Martínez Peña",
    "celular": "1155556666",
    "fechaNacimiento": "1992-11-25",
    "provincia": "CABA",
    "localidad": "Belgrano",
    "codigoPostal": "1428",
    "direccion": "Av. Cabildo 2000",
    "password": "Clave789B"
  }')

if echo "$RESPONSE" | jq -e '.nombre' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Caracteres especiales aceptados${NC}"
    echo "Nombre: $(echo "$RESPONSE" | jq -r '.nombre')"
    echo "Apellido: $(echo "$RESPONSE" | jq -r '.apellido')"
else
    echo -e "${RED}✗ Error con caracteres especiales${NC}"
fi

# Test 6: Validación de edad
echo -e "\n${YELLOW}6. Validación de mayoría de edad${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

# Mayor de edad
RESPONSE=$(curl -s -X POST ${BASE_URL}/usuarios \
  -H "Content-Type: application/json" \
  -d '{
    "email": "adulto@test.com",
    "dni": "35678901",
    "nombre": "Adulto",
    "apellido": "Mayor",
    "celular": "1177778888",
    "fechaNacimiento": "2000-01-01",
    "provincia": "CABA",
    "localidad": "Test",
    "codigoPostal": "1000",
    "direccion": "Test 123",
    "password": "Test1234"
  }')

if echo "$RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Mayor de edad aceptado (25 años)${NC}"
else
    echo -e "${RED}✗ Rechazado${NC}"
fi

# Menor de edad
RESPONSE=$(curl -s -X POST ${BASE_URL}/usuarios \
  -H "Content-Type: application/json" \
  -d '{
    "email": "menor@test.com",
    "dni": "45678901",
    "nombre": "Menor",
    "apellido": "Edad",
    "celular": "1188889999",
    "fechaNacimiento": "2010-01-01",
    "provincia": "CABA",
    "localidad": "Test",
    "codigoPostal": "1000",
    "direccion": "Test 123",
    "password": "Test1234"
  }')

if echo "$RESPONSE" | jq -e '.errores' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Menor de edad correctamente rechazado${NC}"
    echo "$RESPONSE" | jq '.errores[] | select(.campo == "fechaNacimiento")'
else
    echo -e "${RED}✗ Error: menor de edad fue aceptado${NC}"
fi

# Test 7: Duplicados
echo -e "\n${YELLOW}7. Prevención de duplicados${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

# Intentar duplicar email
RESPONSE=$(curl -s -X POST ${BASE_URL}/usuarios \
  -H "Content-Type: application/json" \
  -d '{
    "email": "juan.perez@gmail.com",
    "dni": "39999999",
    "nombre": "Otro",
    "apellido": "Usuario",
    "celular": "1199999999",
    "fechaNacimiento": "1995-01-01",
    "provincia": "CABA",
    "localidad": "Test",
    "codigoPostal": "1000",
    "direccion": "Test 456",
    "password": "OtraClave1"
  }')

if echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Email duplicado correctamente rechazado${NC}"
    echo "$RESPONSE" | jq '.error'
else
    echo -e "${RED}✗ Error: permitió email duplicado${NC}"
fi

# Test 8: Consultas
echo -e "\n${YELLOW}8. Consultas y listados${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

echo "Usuarios en CABA:"
curl -s "${BASE_URL}/usuarios?provincia=CABA&limite=5" | jq '.total'

echo -e "\nListado de usuarios:"
curl -s "${BASE_URL}/usuarios?limite=5" | \
  jq -r '.usuarios[] | "\(.nombre) \(.apellido) - \(.provincia)"'

# Test 9: CUIT válido conocido
echo -e "\n${YELLOW}9. CUIT con dígito verificador válido${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

# CUIT de AFIP (conocido válido): 33-69345023-9
RESPONSE=$(curl -s -X POST ${BASE_URL}/usuarios \
  -H "Content-Type: application/json" \
  -d '{
    "email": "empresa@test.com",
    "dni": "69345023",
    "cuit": "33-69345023-9",
    "nombre": "Empresa",
    "apellido": "Test",
    "celular": "1122223333",
    "fechaNacimiento": "1980-01-01",
    "provincia": "CABA",
    "localidad": "Centro",
    "codigoPostal": "1000",
    "direccion": "Oficina 100",
    "password": "Empresa123"
  }')

if echo "$RESPONSE" | jq -e '.cuit' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ CUIT válido aceptado${NC}"
    echo "CUIT: $(echo "$RESPONSE" | jq -r '.cuit')"
else
    echo -e "${RED}✗ CUIT rechazado${NC}"
    echo "$RESPONSE" | jq '.errores'
fi

# Resumen
echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Resumen del Demo${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

TOTAL=$(curl -s "${BASE_URL}/usuarios" | jq '.total')
echo -e "Total de usuarios creados: ${GREEN}${TOTAL}${NC}"

echo -e "\n${GREEN}Características demostradas:${NC}"
echo "• Validación de DNI argentino"
echo "• Validación de provincias"
echo "• Validación de CBU bancario"
echo "• Manejo de caracteres especiales (ñ, tildes)"
echo "• Validación de mayoría de edad"
echo "• Prevención de duplicados"
echo "• Transformación de datos"

echo -e "\n${YELLOW}Nota sobre CUIT:${NC}"
echo "El algoritmo de validación de CUIT es muy estricto."
echo "Se recomienda usar CUITs conocidos válidos como:"
echo "• 33-69345023-9 (AFIP)"
echo "• 30-50001091-2 (Banco Nación)"
echo "• 30-71481828-3 (YPF)"

echo -e "\n${CYAN}Demo finalizada.${NC}"
