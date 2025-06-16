#!/bin/bash

# User Management Microservice Demo Script
# This script demonstrates the key features of the Queryfy-powered user service

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Base URL
BASE_URL="http://localhost:8080"

# Function to print section headers
print_header() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

# Function to print test descriptions
print_test() {
    echo -e "${GREEN}▶ $1${NC}"
}

# Function to print errors
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Start demo
clear
echo -e "${YELLOW}╔══════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║          Queryfy User Management Microservice Demo                       ║${NC}"
echo -e "${YELLOW}║                                                                          ║${NC}"
echo -e "${YELLOW}║  This demo showcases validation, transformation, and error handling      ║${NC}"
echo -e "${YELLOW}╚══════════════════════════════════════════════════════════════════════════╝${NC}"

# Check if service is running
print_header "1. Service Health Check"
print_test "Checking if service is running on port 8080..."
if curl -s -f ${BASE_URL}/users > /dev/null 2>&1; then
    print_success "Service is running!"
else
    print_error "Service is not running. Please start the service first with: go run main.go"
    exit 1
fi

# Test 1: Data Transformation Features
print_header "2. Data Transformation Features"
print_test "Creating user with messy input data to showcase transformations"

echo -e "\nInput data (notice the formatting issues):"
cat << 'EOF'
{
  "email": "  JOHN.DOE@EXAMPLE.COM  ",
  "username": "John Doe 123",
  "password": "SecurePass123!",
  "firstName": "  john  ",
  "lastName": "DOE",
  "phone": "(555) 123-4567",
  "birthDate": "1990-05-15"
}
EOF

echo -e "\n${GREEN}Sending request...${NC}"
RESPONSE=$(curl -s -X POST ${BASE_URL}/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "  JOHN.DOE@EXAMPLE.COM  ",
    "username": "John Doe 123",
    "password": "SecurePass123!",
    "firstName": "  john  ",
    "lastName": "DOE",
    "phone": "(555) 123-4567",
    "birthDate": "1990-05-15"
  }')

echo -e "\nResponse (notice the transformations):"
echo "$RESPONSE" | jq '.'

USER_ID=$(echo "$RESPONSE" | jq -r '.id')
print_success "User created with ID: $USER_ID"
echo -e "\n${YELLOW}Transformations applied:${NC}"
echo "• Email: trimmed, lowercased → john.doe@example.com"
echo "• Username: lowercased, spaces→underscores → john_doe_123"
echo "• Names: trimmed and capitalized → John Doe"
echo "• Phone: normalized to US format → +15551234567"

# Test 2: Validation Errors
print_header "3. Validation Error Handling"
print_test "Testing various validation rules"

echo -e "\n${BLUE}Test 3.1: Invalid email format${NC}"
curl -s -X POST ${BASE_URL}/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "not-an-email",
    "username": "testuser",
    "password": "Test123!",
    "firstName": "Test",
    "lastName": "User",
    "birthDate": "1995-01-01"
  }' | jq '.'

echo -e "\n${BLUE}Test 3.2: Password too weak (missing special character)${NC}"
curl -s -X POST ${BASE_URL}/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "username": "testuser",
    "password": "Test1234",
    "firstName": "Test",
    "lastName": "User",
    "birthDate": "1995-01-01"
  }' | jq '.'

echo -e "\n${BLUE}Test 3.3: Username with invalid characters${NC}"
curl -s -X POST ${BASE_URL}/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test2@example.com",
    "username": "test@user!",
    "password": "Test123!",
    "firstName": "Test",
    "lastName": "User",
    "birthDate": "1995-01-01"
  }' | jq '.'

echo -e "\n${BLUE}Test 3.4: User too young (under 18)${NC}"
curl -s -X POST ${BASE_URL}/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "young@example.com",
    "username": "younguser",
    "password": "Test123!",
    "firstName": "Young",
    "lastName": "User",
    "birthDate": "2010-01-01"
  }' | jq '.'

# Test 3: Phone Number Normalization
print_header "4. Phone Number Normalization (US Format)"
print_test "Testing various phone number formats"

PHONE_FORMATS=(
    "(555) 987-6543"
    "555-987-6543"
    "555.987.6543"
    "5559876543"
    "+1 555 987 6543"
    "1-555-987-6543"
)

for i in "${!PHONE_FORMATS[@]}"; do
    echo -e "\n${BLUE}Format $((i+1)): ${PHONE_FORMATS[$i]}${NC}"
    
    RESPONSE=$(curl -s -X POST ${BASE_URL}/users \
      -H "Content-Type: application/json" \
      -d "{
        \"email\": \"phone$i@example.com\",
        \"username\": \"phoneuser$i\",
        \"password\": \"Test123!\",
        \"firstName\": \"Phone\",
        \"lastName\": \"Test$i\",
        \"phone\": \"${PHONE_FORMATS[$i]}\",
        \"birthDate\": \"1990-01-01\"
      }")
    
    NORMALIZED_PHONE=$(echo "$RESPONSE" | jq -r '.phone')
    echo "Normalized to: $NORMALIZED_PHONE"
done

# Test 4: Duplicate Prevention
print_header "5. Duplicate Email/Username Prevention"
print_test "Attempting to create user with existing email"

curl -s -X POST ${BASE_URL}/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe@example.com",
    "username": "different_username",
    "password": "Test123!",
    "firstName": "Another",
    "lastName": "User",
    "birthDate": "1985-01-01"
  }' | jq '.'

print_test "Attempting to create user with existing username"

curl -s -X POST ${BASE_URL}/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "another@example.com",
    "username": "john_doe_123",
    "password": "Test123!",
    "firstName": "Another",
    "lastName": "User",
    "birthDate": "1985-01-01"
  }' | jq '.'

# Test 5: Optional Fields and Defaults
print_header "6. Optional Fields and Default Values"
print_test "Creating user with minimal required fields only"

RESPONSE=$(curl -s -X POST ${BASE_URL}/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "minimal@example.com",
    "username": "minimaluser",
    "password": "Test123!",
    "firstName": "Minimal",
    "lastName": "User",
    "birthDate": "1992-01-01"
  }')

echo "$RESPONSE" | jq '.'
print_success "Notice: role defaults to 'user' and status to 'active'"

# Test 6: Update Operations
print_header "7. Update Operations with Partial Data"
print_test "Updating user with transformed fields"

echo -e "\nOriginal user data:"
curl -s ${BASE_URL}/users/${USER_ID} | jq '.'

echo -e "\n${GREEN}Updating with messy data...${NC}"
UPDATE_RESPONSE=$(curl -s -X PUT ${BASE_URL}/users/${USER_ID} \
  -H "Content-Type: application/json" \
  -d '{
    "email": "  UPDATED.EMAIL@EXAMPLE.COM  ",
    "firstName": "  johnny  ",
    "role": "admin"
  }')

echo -e "\nUpdated user (notice transformations still applied):"
echo "$UPDATE_RESPONSE" | jq '.'

# Test 7: Query Features
print_header "8. Query Features with Validation"
print_test "Querying users with various filters"

echo -e "\n${BLUE}Query 1: All admin users${NC}"
curl -s "${BASE_URL}/users?role=admin" | jq '.'

echo -e "\n${BLUE}Query 2: With pagination (limit=2, offset=0)${NC}"
curl -s "${BASE_URL}/users?limit=2&offset=0" | jq '.'

echo -e "\n${BLUE}Query 3: Invalid query parameter${NC}"
curl -s "${BASE_URL}/users?limit=200" | jq '.'
print_success "Notice: limit is capped at 100 by validation"

# Test 8: Soft Delete
print_header "9. Soft Delete Operation"
print_test "Deleting a user (soft delete - changes status)"

echo -e "\nDeleting user ${USER_ID}..."
curl -s -X DELETE ${BASE_URL}/users/${USER_ID}
print_success "User deleted (status changed to 'deleted')"

echo -e "\nVerifying soft delete:"
curl -s ${BASE_URL}/users/${USER_ID} | jq '.status'

# Test 9: Complex Validation Scenario
print_header "10. Complex Validation Scenario"
print_test "Multiple validation errors in single request"

echo -e "\nSending request with multiple issues:"
cat << 'EOF'
{
  "email": "invalid-email",
  "username": "a",
  "password": "weak",
  "firstName": "",
  "lastName": "TooLongLastNameThatExceedsTheMaximumAllowedLengthForValidation",
  "phone": "123",
  "birthDate": "2025-01-01"
}
EOF

echo -e "\n${GREEN}Response showing all validation errors:${NC}"
curl -s -X POST ${BASE_URL}/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "invalid-email",
    "username": "a",
    "password": "weak",
    "firstName": "",
    "lastName": "TooLongLastNameThatExceedsTheMaximumAllowedLengthForValidation",
    "phone": "123",
    "birthDate": "2025-01-01"
  }' | jq '.'

# Summary
print_header "Demo Complete!"
echo -e "${GREEN}This demo showcased:${NC}"
echo "• Data transformation (trim, lowercase, capitalize, normalize)"
echo "• Comprehensive validation rules"
echo "• Phone number normalization"
echo "• Duplicate prevention"
echo "• Default values for optional fields"
echo "• Partial updates with transformation"
echo "• Query parameter validation"
echo "• Soft delete operations"
echo "• Multiple validation errors in single response"
echo ""
echo -e "${YELLOW}All powered by Queryfy's declarative validation and transformation pipeline!${NC}"