package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// User represents our user model
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Phone     string    `json:"phone,omitempty"`
	BirthDate string    `json:"birthDate"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UserService handles business logic
type UserService struct {
	mu    sync.RWMutex
	users map[string]User
}

// NewUserService creates a new user service
func NewUserService() *UserService {
	return &UserService{
		users: make(map[string]User),
	}
}

// Schemas defines all validation schemas for the service
type Schemas struct {
	CreateUser qf.Schema
	UpdateUser qf.Schema
	QueryUser  qf.Schema
}

// Custom transformers
var (
	// normalizeUsername ensures usernames are lowercase with no spaces
	normalizeUsername = func(value interface{}) (interface{}, error) {
		username := value.(string)
		username = strings.ToLower(strings.TrimSpace(username))
		username = strings.ReplaceAll(username, " ", "_")
		return username, nil
	}

	// capitalizeNames properly capitalizes first/last names
	capitalizeNames = func(value interface{}) (interface{}, error) {
		name := value.(string)
		if name == "" {
			return name, nil
		}
		// Simple capitalization
		return strings.Title(strings.ToLower(name)), nil
	}

	// normalizeEmail custom function for email normalization
	normalizeEmail = func(value interface{}) (interface{}, error) {
		email := strings.TrimSpace(strings.ToLower(value.(string)))
		// Additional email normalization could go here
		return email, nil
	}
)

// InitSchemas creates all validation schemas
func InitSchemas() *Schemas {
	// Create user schema with comprehensive validation and transformation
	createUserSchema := builders.Object().
		Field("email",
			builders.Transform(
				builders.String().
					Email().
					Required(),
			).Add(transformers.Trim()).
			Add(transformers.Lowercase()).
			Add(normalizeEmail)).
		Field("username",
			builders.Transform(
				builders.String().
					MinLength(3).
					MaxLength(30).
					Pattern(`^[a-z0-9_]+$`).
					Required(),
			).Add(transformers.Trim()).
			Add(normalizeUsername)).
		Field("password",
			builders.String().
				MinLength(8).
				MaxLength(72).
				Pattern(`[A-Z]`).     // At least one uppercase
				Pattern(`[a-z]`).     // At least one lowercase
				Pattern(`[0-9]`).     // At least one digit
				Pattern(`[!@#$%^&*]`).// At least one special char
				Required()).
		Field("firstName",
			builders.Transform(
				builders.String().
					MinLength(1).
					MaxLength(50).
					Required(),
			).Add(transformers.Trim()).
			Add(capitalizeNames)).
		Field("lastName",
			builders.Transform(
				builders.String().
					MinLength(1).
					MaxLength(50).
					Required(),
			).Add(transformers.Trim()).
			Add(capitalizeNames)).
		Field("phone",
			builders.Transform(
				builders.String().
					Pattern(`^\+\d{10,15}$`).
					Optional(),
			).Add(transformers.Trim()).
			Add(transformers.NormalizePhone("US"))).
		Field("birthDate",
			builders.DateTime().
				DateOnly().
				Past().
				Age(18, 120).
				Required()).
		Field("role",
			builders.String().
				Enum("user", "admin", "moderator").
				Optional()).
		Custom(func(value interface{}) error {
			// Custom validation: ensure unique email/username
			// In a real app, check against database
			return nil
		})

	// Update user schema - all fields optional except ID
	updateUserSchema := builders.Object().
		Field("email",
			builders.Transform(
				builders.String().
					Email().
					Optional(),
			).Add(transformers.Trim()).
			Add(transformers.Lowercase()).
			Add(normalizeEmail)).
		Field("username",
			builders.Transform(
				builders.String().
					MinLength(3).
					MaxLength(30).
					Pattern(`^[a-z0-9_]+$`).
					Optional(),
			).Add(transformers.Trim()).
			Add(normalizeUsername)).
		Field("firstName",
			builders.Transform(
				builders.String().
					MinLength(1).
					MaxLength(50).
					Optional(),
			).Add(transformers.Trim()).
			Add(capitalizeNames)).
		Field("lastName",
			builders.Transform(
				builders.String().
					MinLength(1).
					MaxLength(50).
					Optional(),
			).Add(transformers.Trim()).
			Add(capitalizeNames)).
		Field("phone",
			builders.Transform(
				builders.String().
					Pattern(`^\+\d{10,15}$`).
					Optional(),
			).Add(transformers.Trim()).
			Add(transformers.NormalizePhone("US"))).
		Field("birthDate",
			builders.DateTime().
				DateOnly().
				Past().
				Age(18, 120).
				Optional()).
		Field("role",
			builders.String().
				Enum("user", "admin", "moderator").
				Optional()).
		Field("status",
			builders.String().
				Enum("active", "suspended", "deleted").
				Optional())

	// Query validation schema
	queryUserSchema := builders.Object().
		Field("email", builders.String().Email().Optional()).
		Field("username", builders.String().Optional()).
		Field("role", builders.String().Enum("user", "admin", "moderator").Optional()).
		Field("status", builders.String().Enum("active", "suspended", "deleted").Optional()).
		Field("limit", builders.Number().Min(1).Max(100).Optional()).
		Field("offset", builders.Number().Min(0).Optional())

	return &Schemas{
		CreateUser: createUserSchema,
		UpdateUser: updateUserSchema,
		QueryUser:  queryUserSchema,
	}
}

// Helper function to extract transformed data
func extractTransformedData(data map[string]interface{}, ctx *qf.ValidationContext) map[string]interface{} {
	// Create a copy of the data
	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = v
	}

	// Apply transformations based on the context
	for _, transform := range ctx.Transformations() {
		// Parse path and apply transformation
		path := transform.Path
		if path != "" {
			// Simple implementation - in production, use proper path parsing
			result[path] = transform.Result
		}
	}

	return result
}

// CRUD Operations

// Create adds a new user
func (s *UserService) Create(data map[string]interface{}) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID and timestamps
	user := User{
		ID:        uuid.New().String(),
		Email:     data["email"].(string),
		Username:  data["username"].(string),
		FirstName: data["firstName"].(string),
		LastName:  data["lastName"].(string),
		BirthDate: data["birthDate"].(string),
		Role:      "user", // Default role
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set optional fields
	if phone, ok := data["phone"].(string); ok {
		user.Phone = phone
	}
	if role, ok := data["role"].(string); ok {
		user.Role = role
	}

	// Check for duplicate email/username
	for _, existing := range s.users {
		if existing.Email == user.Email {
			return nil, fmt.Errorf("email already exists")
		}
		if existing.Username == user.Username {
			return nil, fmt.Errorf("username already exists")
		}
	}

	s.users[user.ID] = user
	return &user, nil
}

// Get retrieves a user by ID
func (s *UserService) Get(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return &user, nil
}

// Update modifies an existing user
func (s *UserService) Update(id string, data map[string]interface{}) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	// Update fields if provided
	if email, ok := data["email"].(string); ok {
		// Check for duplicate email
		for uid, u := range s.users {
			if uid != id && u.Email == email {
				return nil, fmt.Errorf("email already exists")
			}
		}
		user.Email = email
	}
	if username, ok := data["username"].(string); ok {
		// Check for duplicate username
		for uid, u := range s.users {
			if uid != id && u.Username == username {
				return nil, fmt.Errorf("username already exists")
			}
		}
		user.Username = username
	}
	if firstName, ok := data["firstName"].(string); ok {
		user.FirstName = firstName
	}
	if lastName, ok := data["lastName"].(string); ok {
		user.LastName = lastName
	}
	if phone, ok := data["phone"].(string); ok {
		user.Phone = phone
	}
	if birthDate, ok := data["birthDate"].(string); ok {
		user.BirthDate = birthDate
	}
	if role, ok := data["role"].(string); ok {
		user.Role = role
	}
	if status, ok := data["status"].(string); ok {
		user.Status = status
	}

	user.UpdatedAt = time.Now()
	s.users[id] = user
	return &user, nil
}

// Delete removes a user (soft delete by changing status)
func (s *UserService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	user.Status = "deleted"
	user.UpdatedAt = time.Now()
	s.users[id] = user
	return nil
}

// Query searches for users based on criteria
func (s *UserService) Query(criteria map[string]interface{}) ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []User
	limit := 10
	offset := 0

	// Extract pagination
	if l, ok := criteria["limit"].(float64); ok {
		limit = int(l)
	}
	if o, ok := criteria["offset"].(float64); ok {
		offset = int(o)
	}

	// Filter users
	count := 0
	for _, user := range s.users {
		// Apply filters
		if email, ok := criteria["email"].(string); ok && user.Email != email {
			continue
		}
		if username, ok := criteria["username"].(string); ok && user.Username != username {
			continue
		}
		if role, ok := criteria["role"].(string); ok && user.Role != role {
			continue
		}
		if status, ok := criteria["status"].(string); ok && user.Status != status {
			continue
		}

		// Apply pagination
		if count >= offset && len(results) < limit {
			results = append(results, user)
		}
		count++
	}

	return results, nil
}

// HTTP Handlers

type Handler struct {
	service *UserService
	schemas *Schemas
}

func NewHandler(service *UserService, schemas *Schemas) *Handler {
	return &Handler{
		service: service,
		schemas: schemas,
	}
}

// CreateUser handles POST /users
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate using Queryfy
	ctx := qf.NewValidationContext(qf.Strict)
	if err := h.schemas.CreateUser.Validate(data, ctx); err != nil || ctx.HasErrors() {
		respondValidationErrors(w, ctx)
		return
	}

	// Extract transformed data from context
	transformedData := extractTransformedData(data, ctx)

	// Create user
	user, err := h.service.Create(transformedData)
	if err != nil {
		respondError(w, http.StatusConflict, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, user)
}

// GetUser handles GET /users/{id}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, err := h.service.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// UpdateUser handles PUT /users/{id}
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate
	ctx := qf.NewValidationContext(qf.Strict)
	if err := h.schemas.UpdateUser.Validate(data, ctx); err != nil || ctx.HasErrors() {
		respondValidationErrors(w, ctx)
		return
	}

	// Extract transformed data
	transformedData := extractTransformedData(data, ctx)

	// Update user
	user, err := h.service.Update(id, transformedData)
	if err != nil {
		if err.Error() == "user not found" {
			respondError(w, http.StatusNotFound, err.Error())
		} else {
			respondError(w, http.StatusConflict, err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// DeleteUser handles DELETE /users/{id}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.service.Delete(id); err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// QueryUsers handles GET /users with query parameters
func (h *Handler) QueryUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters into a map
	queryData := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			// Convert numeric parameters
			if key == "limit" || key == "offset" {
				if num, err := strconv.Atoi(values[0]); err == nil {
					queryData[key] = float64(num)
				}
			} else {
				queryData[key] = values[0]
			}
		}
	}

	// Validate query parameters
	ctx := qf.NewValidationContext(qf.Strict)
	if err := h.schemas.QueryUser.Validate(queryData, ctx); err != nil || ctx.HasErrors() {
		respondValidationErrors(w, ctx)
		return
	}

	// Query users
	users, err := h.service.Query(queryData)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to query users")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error": message,
	})
}

func respondValidationErrors(w http.ResponseWriter, ctx *qf.ValidationContext) {
	errors := make([]map[string]string, 0)
	for _, err := range ctx.Errors() {
		errors = append(errors, map[string]string{
			"field":   err.Path,
			"message": err.Message,
		})
	}
	respondJSON(w, http.StatusBadRequest, map[string]interface{}{
		"errors": errors,
	})
}

// Main function
func main() {
	// Initialize service and schemas
	service := NewUserService()
	schemas := InitSchemas()
	handler := NewHandler(service, schemas)

	// Setup routes
	router := mux.NewRouter()
	router.HandleFunc("/users", handler.CreateUser).Methods("POST")
	router.HandleFunc("/users", handler.QueryUsers).Methods("GET")
	router.HandleFunc("/users/{id}", handler.GetUser).Methods("GET")
	router.HandleFunc("/users/{id}", handler.UpdateUser).Methods("PUT")
	router.HandleFunc("/users/{id}", handler.DeleteUser).Methods("DELETE")

	// Add some sample data
	sampleData := map[string]interface{}{
		"email":     "admin@example.com",
		"username":  "admin",
		"password":  "Admin123!",
		"firstName": "System",
		"lastName":  "Administrator",
		"birthDate": "1990-01-01",
		"role":      "admin",
	}
	
	ctx := qf.NewValidationContext(qf.Strict)
	if err := schemas.CreateUser.Validate(sampleData, ctx); err == nil && !ctx.HasErrors() {
		transformedData := extractTransformedData(sampleData, ctx)
		if _, err := service.Create(transformedData); err == nil {
			log.Println("Created sample admin user")
		}
	}

	// Start server
	port := ":8080"
	log.Printf("Starting user management service on %s", port)
	log.Printf("Try: curl -X POST http://localhost%s/users -H 'Content-Type: application/json' -d '{\"email\":\"test@example.com\",\"username\":\"testuser\",\"password\":\"Test123!\",\"firstName\":\"Test\",\"lastName\":\"User\",\"birthDate\":\"1995-01-01\"}'", port)
	log.Fatal(http.ListenAndServe(port, router))
}