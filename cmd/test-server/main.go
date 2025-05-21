package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

const (
	apiVersion = "2023-02-28"
	// Resource types.
	resourceUsers         = "users"
	resourceOrganizations = "organizations"
	resourceRoles         = "roles"
)

// Config holds all configuration for the server.
type config struct {
	port            string
	readTimeout     time.Duration
	writeTimeout    time.Duration
	idleTimeout     time.Duration
	shutdownTimeout time.Duration
	rateLimit       int
	rateWindow      time.Duration
}

// defaultConfig returns a config with sensible defaults.
func defaultConfig() *config {
	return &config{
		port:            "8080",
		readTimeout:     10 * time.Second,
		writeTimeout:    10 * time.Second,
		idleTimeout:     30 * time.Second,
		shutdownTimeout: 10 * time.Second,
		rateLimit:       1000, // requests per window
		rateWindow:      8 * time.Hour,
	}
}

// This is a test server that simulates a REST API for user management, authentication, and organization data
// It provides endpoints for:
// 1. Authentication (/oauth/token) - Issues Bearer tokens using client credentials
// 2. Users management (/api/sites/{site}/users)
// 3. Organizations (/api/sites/{site}/organizations)
// 4. Roles (/api/sites/{site}/roles)
// 5. Role assignments by user (/api/sites/{site}/users/{userId}/roles)
// 6. Role assign to user (/api/sites/{site}/roles/{roleCode}/users/add)
// 7. Role removal from user (/api/sites/{site}/roles/{roleCode}/users/remove)

// Key features:
// - OAuth2 token-based authentication
// - Rate limiting
// - Pagination support
// - Request/Response logging
// - Graceful shutdown
// - Health check endpoint

// The server maintains in-memory data structures for.
var (
	// Cache for active authentication tokens.
	tokenCache   = make(map[string]tokenInfo)
	tokenCacheMu sync.RWMutex
)

type tokenInfo struct {
	Token     string    `json:"access_token"`
	Type      string    `json:"token_type"`
	ExpiresIn int       `json:"expires_in"`
	IssuedAt  time.Time `json:"-"`
}

// Server struct holds all the test data and configuration.
type server struct {
	config    *config                         // Server configuration (ports, timeouts, etc)
	baseURL   string                          // Base URL for the server
	users     map[string]userResponse         // Test user data
	orgs      map[string]organizationResponse // Test organization data
	roles     map[string]roleResponse         // Test role definitions
	roleUsers map[string][]string             // Maps roles to user IDs
}

// The server follows these main workflows:
// 1. Client requests a token with client credentials
// 2. Server validates credentials and issues a Bearer token
// 3. Client uses token in Authorization header for subsequent requests
// 4. Server validates token and rate limits before processing requests
// 5. Responses include pagination and rate limit headers

// Test data is populated in addTestData() method
// The server can be configured via environment variables
// Default configuration provides sensible defaults for timeouts and rate limits

type userResponse struct {
	UserID              string `json:"userId"`
	UserName            string `json:"userName"`
	FirstName           string `json:"firstName"`
	MiddleName          string `json:"middleName,omitempty"`
	LastName            string `json:"lastName"`
	LocalizedFirstName  string `json:"localizedFirstName,omitempty"`
	LocalizedMiddleName string `json:"localizedMiddleName,omitempty"`
	LocalizedLastName   string `json:"localizedLastName,omitempty"`
	SecondaryLastName   string `json:"secondaryLastName,omitempty"`
	Email               string `json:"email,omitempty"`
	Title               string `json:"title,omitempty"`
	ManagerUserName     string `json:"managerUserName,omitempty"`
	OrganizationCode    string `json:"organizationCode"`
	OuCode              string `json:"ouCode"`
	CostCenterNumber    string `json:"costCenterNumber"`
	LocationCode        string `json:"locationCode"`
	LanguageCode        string `json:"languageCode"`
}

type organizationResponse struct {
	OrganizationCode string `json:"organizationCode"`
	DisplayName      string `json:"displayName"`
	Description      string `json:"description,omitempty"`
}

type roleResponse struct {
	RoleCode    string `json:"roleCode"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
}

type paginationParams struct {
	skip int
	top  int
}

func parsePaginationParams(r *http.Request) (*paginationParams, error) {
	params := &paginationParams{
		top: 1000, // Default maximum as per API spec
	}

	if skipStr := r.URL.Query().Get("skip"); skipStr != "" {
		skip, err := strconv.Atoi(skipStr)
		if err != nil || skip < 0 {
			return nil, fmt.Errorf("invalid skip parameter")
		}
		params.skip = skip
	}

	if topStr := r.URL.Query().Get("top"); topStr != "" {
		top, err := strconv.Atoi(topStr)
		if err != nil || top < 0 || top > 1000 {
			return nil, fmt.Errorf("invalid top parameter")
		}
		params.top = top
	}

	return params, nil
}

// addRateLimitHeaders adds standard rate limit headers to the response.
func addRateLimitHeaders(w http.ResponseWriter, config *config) {
	w.Header().Set("X-Rate-Limit-Limit", strconv.Itoa(config.rateLimit))
	w.Header().Set("X-Rate-Limit-Remaining", strconv.Itoa(config.rateLimit))
	w.Header().Set("X-Rate-Limit-Reset", time.Now().Add(config.rateWindow).Format(time.RFC3339))
}

func newServer(config *config) *server {
	return &server{
		config:    config,
		baseURL:   fmt.Sprintf("http://localhost:%s", config.port),
		users:     make(map[string]userResponse),
		orgs:      make(map[string]organizationResponse),
		roles:     make(map[string]roleResponse),
		roleUsers: make(map[string][]string),
	}
}

// requestLoggerMiddleware logs request details.
func requestLoggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := newResponseWriter(w)

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		log.Printf(
			"method=%s path=%s status=%d duration=%s remote_addr=%s user_agent=%s",
			r.Method,
			r.URL.Path,
			ww.statusCode,
			duration,
			r.RemoteAddr,
			r.UserAgent(),
		)
	}
}

// responseWriter is a custom response writer that captures the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// extractPathSegments returns the path segments after a given prefix.
func extractPathSegments(path string, prefix string) ([]string, error) {
	if !strings.HasPrefix(path, prefix) {
		return nil, fmt.Errorf("path must start with %s", prefix)
	}
	rest := strings.TrimPrefix(path, prefix)
	segments := strings.Split(rest, "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("not enough path segments")
	}
	return segments, nil
}

// siteDispatcher handles /api/sites/{site}/...
func (s *server) siteDispatcher(w http.ResponseWriter, r *http.Request) {
	segments, err := extractPathSegments(r.URL.Path, "/api/sites/")
	if err != nil {
		sendErrorResponse(w, r, http.StatusNotFound, "invalid_path", err.Error(), nil, s.config)
		return
	}

	// siteID := segments[0]

	if len(segments) < 2 {
		sendErrorResponse(w, r, http.StatusNotFound, "invalid_path", "Missing resource segment", nil, s.config)
		return
	}

	resource := segments[1]

	switch {
	case resource == resourceUsers:
		s.handleUsers(w, r)

	case resource == resourceOrganizations:
		s.handleOrganizations(w, r)

	case resource == resourceRoles:
		// Extract roleCode if it exists
		roleCode := ""
		if len(segments) > 2 {
			roleCode = segments[2]
		}

		switch {
		// Handle /api/sites/{site}/roles/{roleCode}/users/add and /users/remove
		case len(segments) == 5 && roleCode != "" && segments[3] == resourceUsers:
			action := segments[4]
			switch action {
			case "add":
				s.handleAddRoleUsers(w, r, roleCode)
			case "remove":
				s.handleRemoveRoleUsers(w, r, roleCode)
			default:
				sendErrorResponse(w, r, http.StatusNotFound, "not_found", "Invalid action", nil, s.config)
			}
		// Handle /api/sites/{site}/roles/{roleCode}/users
		case len(segments) == 4 && roleCode != "" && segments[3] == resourceUsers:
			s.handleRoleUsersByCode(w, r, roleCode)
		// Handle /api/sites/{site}/roles
		default:
			s.handleRoles(w, r)
		}

	default:
		sendErrorResponse(w, r, http.StatusNotFound, "not_found", "Invalid resource", nil, s.config)
	}
}

// handleRoleUsersByCode handles requests for specific role users.
func (s *server) handleRoleUsersByCode(w http.ResponseWriter, r *http.Request, roleCode string) {
	// 1. Validate API version
	if apiVersionParam := r.URL.Query().Get("api-version"); apiVersionParam != apiVersion {
		sendErrorResponse(w, r, http.StatusBadRequest, "UNSUPPORTED_API_VERSION", "API version not supported", nil, s.config)
		return
	}

	if r.Method != http.MethodGet {
		sendErrorResponse(w, r, http.StatusMethodNotAllowed, "VALIDATION_ERROR", "Only GET method is allowed", nil, s.config)
		return
	}

	// Parse pagination parameters
	params, err := parsePaginationParams(r)
	if err != nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil, s.config)
		return
	}

	userIDs := s.roleUsers[roleCode]
	if userIDs == nil {
		userIDs = []string{}
	}

	// Sort userIds for consistent ordering
	sort.Strings(userIDs)

	// Calculate pagination
	startIndex := params.skip
	endIndex := startIndex + params.top
	if endIndex > len(userIDs) {
		endIndex = len(userIDs)
	}
	if startIndex >= len(userIDs) {
		startIndex = len(userIDs)
	}

	// Set rate limit headers
	addRateLimitHeaders(w, s.config)

	response := map[string]interface{}{
		"maxItems": len(userIDs),
		"value":    userIDs[startIndex:endIndex],
	}

	sendJSONResponse(w, r, response, s.config)
}

// Update the Start() method to use the new dispatcher.
func (s *server) Start() error {
	// Add test data
	s.addTestData()

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Auth endpoint
	mux.HandleFunc("/oauth/token", s.handleAuth)

	// API endpoints with middleware chain
	apiHandler := func(path string, handler http.HandlerFunc) {
		mux.HandleFunc(path, requestLoggerMiddleware(authMiddleware(s.config, handler)))
	}

	// Register the site dispatcher
	apiHandler("/api/sites/", s.siteDispatcher)

	server := &http.Server{
		Addr:              ":" + s.config.port,
		Handler:           mux,
		ReadTimeout:       s.config.readTimeout,
		WriteTimeout:      s.config.writeTimeout,
		IdleTimeout:       s.config.idleTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		log.Printf("Server listening on %s", s.baseURL)
		serverErrors <- server.ListenAndServe()
	}()

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Printf("Server is shutting down due to %v signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), s.config.shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

// handleHealth implements a health check endpoint.
func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed", nil, s.config)
		return
	}

	health := struct {
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp"`
		Version   string    `json:"version"`
	}{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   apiVersion,
	}

	sendJSONResponse(w, r, health, s.config)
}

func authMiddleware(config *config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendAuthError(w, "No authorization header provided", config)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			sendAuthError(w, "Invalid authorization header format", config)
			return
		}

		tokenCacheMu.RLock()
		token, exists := tokenCache[parts[1]]
		tokenCacheMu.RUnlock()

		if !exists {
			sendAuthError(w, "Invalid token", config)
			return
		}

		if time.Since(token.IssuedAt) > time.Duration(token.ExpiresIn)*time.Second {
			sendAuthError(w, "Token expired", config)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (s *server) handleAuth(w http.ResponseWriter, r *http.Request) {
	logRequest("/oauth/token", r)

	if r.Method != http.MethodPost {
		sendErrorResponse(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed", nil, s.config)
		return
	}

	err := r.ParseForm()
	if err != nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "invalid_request", "Could not parse form data", nil, s.config)
		return
	}

	grantType := r.FormValue("grant_type")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if grantType != "client_credentials" {
		sendErrorResponse(w, r, http.StatusBadRequest, "invalid_grant", "Only client_credentials grant type is supported", nil, s.config)
		return
	}

	if clientID == "" || clientSecret == "" {
		sendErrorResponse(w, r, http.StatusUnauthorized, "invalid_client", "Invalid client credentials", nil, s.config)
		return
	}

	// Generate new token
	token := tokenInfo{
		Token:     uuid.New().String(),
		Type:      "Bearer",
		ExpiresIn: 3600,
		IssuedAt:  time.Now(),
	}

	// Cache the token
	tokenCacheMu.Lock()
	tokenCache[token.Token] = token
	tokenCacheMu.Unlock()

	sendJSONResponse(w, r, token, s.config)
}

func (s *server) handleOrganizations(w http.ResponseWriter, r *http.Request) {
	// 1. Validate API version
	if apiVersionParam := r.URL.Query().Get("api-version"); apiVersionParam != apiVersion {
		sendErrorResponse(w, r, http.StatusBadRequest, "UNSUPPORTED_API_VERSION", "API version not supported", nil, s.config)
		return
	}

	if r.Method != http.MethodGet {
		sendErrorResponse(w, r, http.StatusMethodNotAllowed, "VALIDATION_ERROR", "Only GET method is allowed", nil, s.config)
		return
	}

	// Parse pagination parameters
	params, err := parsePaginationParams(r)
	if err != nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil, s.config)
		return
	}

	// Create a sorted slice of organizations
	orgs := make([]organizationResponse, 0, len(s.orgs))
	for _, org := range s.orgs {
		orgs = append(orgs, org)
	}

	// Sort organizations by OrganizationCode for consistent ordering
	sort.Slice(orgs, func(i, j int) bool {
		return orgs[i].OrganizationCode < orgs[j].OrganizationCode
	})

	// Calculate pagination
	startIndex := params.skip
	endIndex := startIndex + params.top
	if endIndex > len(orgs) {
		endIndex = len(orgs)
	}
	if startIndex >= len(orgs) {
		startIndex = len(orgs)
	}

	// Set rate limit headers
	addRateLimitHeaders(w, s.config)

	response := map[string]interface{}{
		"maxItems": len(orgs),
		"value":    orgs[startIndex:endIndex],
	}

	sendJSONResponse(w, r, response, s.config)
}

func (s *server) handleUsers(w http.ResponseWriter, r *http.Request) {
	// 1. Validate API version
	if apiVersionParam := r.URL.Query().Get("api-version"); apiVersionParam != apiVersion {
		sendErrorResponse(w, r, http.StatusBadRequest, "UNSUPPORTED_API_VERSION", "API version not supported", nil, s.config)
		return
	}

	if r.Method != http.MethodGet {
		sendErrorResponse(w, r, http.StatusMethodNotAllowed, "VALIDATION_ERROR", "Only GET method is allowed", nil, s.config)
		return
	}

	// Parse pagination parameters
	params, err := parsePaginationParams(r)
	if err != nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil, s.config)
		return
	}

	// Create a sorted slice of users
	users := make([]userResponse, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	// Sort users by UserID for consistent ordering
	sort.Slice(users, func(i, j int) bool {
		return users[i].UserID < users[j].UserID
	})

	// Calculate pagination
	startIndex := params.skip
	endIndex := startIndex + params.top
	if endIndex > len(users) {
		endIndex = len(users)
	}
	if startIndex >= len(users) {
		startIndex = len(users)
	}

	// Set rate limit headers
	addRateLimitHeaders(w, s.config)

	response := map[string]interface{}{
		"maxItems": len(users),
		"value":    users[startIndex:endIndex],
	}

	sendJSONResponse(w, r, response, s.config)
}

func (s *server) handleRoles(w http.ResponseWriter, r *http.Request) {
	// 1. Validate API version
	if apiVersionParam := r.URL.Query().Get("api-version"); apiVersionParam != apiVersion {
		sendErrorResponse(w, r, http.StatusBadRequest, "UNSUPPORTED_API_VERSION", "API version not supported", nil, s.config)
		return
	}

	if r.Method != http.MethodGet {
		sendErrorResponse(w, r, http.StatusMethodNotAllowed, "VALIDATION_ERROR", "Only GET method is allowed", nil, s.config)
		return
	}

	// Parse pagination parameters
	params, err := parsePaginationParams(r)
	if err != nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil, s.config)
		return
	}

	// Create a sorted slice of roles
	roles := make([]roleResponse, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}

	// Sort roles by RoleCode for consistent ordering
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].RoleCode < roles[j].RoleCode
	})

	// Calculate pagination
	startIndex := params.skip
	endIndex := startIndex + params.top
	if endIndex > len(roles) {
		endIndex = len(roles)
	}
	if startIndex >= len(roles) {
		startIndex = len(roles)
	}

	// Set rate limit headers
	addRateLimitHeaders(w, s.config)

	response := map[string]interface{}{
		"maxItems": len(roles),
		"value":    roles[startIndex:endIndex],
	}

	sendJSONResponse(w, r, response, s.config)
}

func (s *server) addTestData() {
	// Add test users
	s.users["b1a8f5e2-3c4d-4e6f-8a9b-1c2d3e4f5a6b"] = userResponse{
		UserID:              "b1a8f5e2-3c4d-4e6f-8a9b-1c2d3e4f5a6b",
		UserName:            "JDOE123",
		FirstName:           "John",
		MiddleName:          "Robert",
		LastName:            "Doe",
		LocalizedFirstName:  "John",
		LocalizedMiddleName: "Rob",
		LocalizedLastName:   "Doe",
		SecondaryLastName:   "",
		Email:               "jdoe@example.com",
		Title:               "Senior Accountant",
		ManagerUserName:     "MGR456",
		OrganizationCode:    "acc",
		OuCode:              "123",
		CostCenterNumber:    "123",
		LocationCode:        "Alameda",
		LanguageCode:        "en-GB",
	}
	s.users["c2b9f6e3-4d5e-5f7g-9b0c-2d3e4f5g6h7i"] = userResponse{
		UserID:              "c2b9f6e3-4d5e-5f7g-9b0c-2d3e4f5g6h7i",
		UserName:            "JSMITH456",
		FirstName:           "Jane",
		MiddleName:          "Elizabeth",
		LastName:            "Smith",
		LocalizedFirstName:  "Jane",
		LocalizedMiddleName: "Liz",
		LocalizedLastName:   "Smith",
		SecondaryLastName:   "",
		Email:               "jsmith@example.com",
		Title:               "HR Manager",
		ManagerUserName:     "MGR789",
		OrganizationCode:    "hr",
		OuCode:              "456",
		CostCenterNumber:    "456",
		LocationCode:        "San Francisco",
		LanguageCode:        "en-US",
	}
	s.users["a53cdeee-fa3a-44af-be6b-7f5b37816982"] = userResponse{
		UserID:              "a53cdeee-fa3a-44af-be6b-7f5b37816982",
		UserName:            "X987123",
		FirstName:           "Molly",
		MiddleName:          "Beverly",
		LastName:            "Howe",
		LocalizedFirstName:  "Polly",
		LocalizedMiddleName: "Bev",
		LocalizedLastName:   "Red",
		SecondaryLastName:   "van Zijl",
		Email:               "pred@domain.com",
		Title:               "VP of Procurement",
		ManagerUserName:     "32109121116",
		OrganizationCode:    "acc",
		OuCode:              "123",
		CostCenterNumber:    "9097",
		LocationCode:        "Alameda",
		LanguageCode:        "en-GB",
	}

	// Add test organizations
	s.orgs["acc"] = organizationResponse{
		OrganizationCode: "acc",
		DisplayName:      "Accounting Department",
		Description:      "Handles all financial transactions and reporting.",
	}
	s.orgs["hr"] = organizationResponse{
		OrganizationCode: "hr",
		DisplayName:      "Human Resources",
		Description:      "Manages employee relations, benefits, and recruitment.",
	}
	s.orgs["dev"] = organizationResponse{
		OrganizationCode: "dev",
		DisplayName:      "Development Team",
		Description:      "Responsible for software development and maintenance.",
	}
	s.orgs["qa"] = organizationResponse{
		OrganizationCode: "qa",
		DisplayName:      "Quality Assurance",
		Description:      "Ensures the quality and reliability of products through rigorous testing.",
	}
	s.orgs["sales"] = organizationResponse{
		OrganizationCode: "sales",
		DisplayName:      "Sales Department",
		Description:      "Focuses on customer acquisition and revenue generation.",
	}

	// Add test roles
	addRole(s, "admin", "Administrator")
	addRole(s, "user", "Regular User")
	addRole(s, "HM-1", "Client Hiring Manager")

	// Add test role assignments
	s.roleUsers["admin"] = []string{"b1a8f5e2-3c4d-4e6f-8a9b-1c2d3e4f5a6b"}
	s.roleUsers["user"] = []string{"c2b9f6e3-4d5e-5f7g-9b0c-2d3e4f5g6h7i"}
	s.roleUsers["HM-1"] = []string{"a53cdeee-fa3a-44af-be6b-7f5b37816982"}
}

func addRole(s *server, roleCode, description string) {
	if _, exists := s.roles[roleCode]; exists {
		log.Printf("Role %s already exists, skipping addition.", roleCode)
		return
	}
	s.roles[roleCode] = roleResponse{
		RoleCode:    roleCode,
		DisplayName: roleCode,
		Description: description,
	}
	log.Printf("Added role: %s - %s", roleCode, description)
}

func sendAuthError(w http.ResponseWriter, message string, config *config) {
	sendErrorResponse(w, nil, http.StatusUnauthorized, "unauthorized", message, nil, config)
}

// Update the request structure to match Beeline API.
type roleUserRequest struct {
	Value []string `json:"value"` // This is correct, but we need to validate UUID format
}

// Helper function to validate users exist.
func (s *server) validateUsers(userIDs []string) ([]string, error) {
	var invalidUsers []string
	for _, userID := range userIDs {
		if _, exists := s.users[userID]; !exists {
			invalidUsers = append(invalidUsers, userID)
		}
	}
	if len(invalidUsers) > 0 {
		return invalidUsers, fmt.Errorf("users not found")
	}
	return nil, nil
}

// Helper function to check if a user already has a role.
func (s *server) hasRole(userId string, roleCode string) bool {
	existingUsers := s.roleUsers[roleCode]
	for _, existingId := range existingUsers {
		if existingId == userId {
			return true
		}
	}
	return false
}

func (s *server) handleAddRoleUsers(w http.ResponseWriter, r *http.Request, roleCode string) {
	// 1. Validate API version
	if apiVersionParam := r.URL.Query().Get("api-version"); apiVersionParam != apiVersion {
		sendErrorResponse(w, r, http.StatusBadRequest, "UNSUPPORTED_API_VERSION", "API version not supported", nil, s.config)
		return
	}

	if r.Method != http.MethodPost {
		sendErrorResponse(w, r, http.StatusMethodNotAllowed, "VALIDATION_ERROR", "Only POST method is allowed", nil, s.config)
		return
	}

	// Validate role exists
	if _, exists := s.roles[roleCode]; !exists {
		sendErrorResponse(w, r, http.StatusBadRequest, "NOT_FOUND", fmt.Sprintf("Role %s does not exist", roleCode), nil, s.config)
		return
	}

	var req roleUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil, s.config)
		return
	}

	// Validate request has value field
	if req.Value == nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Missing required field: value", nil, s.config)
		return
	}

	// Validate request has at least one user
	if len(req.Value) == 0 {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Must provide at least one userId", nil, s.config)
		return
	}

	// Validate users exist
	if invalidUsers, err := s.validateUsers(req.Value); err != nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "NOT_FOUND", "Some users do not exist", invalidUsers, s.config)
		return
	}

	// Initialize role users array if it doesn't exist
	if s.roleUsers[roleCode] == nil {
		s.roleUsers[roleCode] = make([]string, 0)
	}

	// Filter out users that already have the role
	newUsers := make([]string, 0)
	for _, userId := range req.Value {
		if !s.hasRole(userId, roleCode) {
			newUsers = append(newUsers, userId)
		}
	}

	// Only append users that don't already have the role
	if len(newUsers) > 0 {
		s.roleUsers[roleCode] = append(s.roleUsers[roleCode], newUsers...)
	}

	// Set rate limit headers
	addRateLimitHeaders(w, s.config)

	// Return 200 with no body as per API spec
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleRemoveRoleUsers(w http.ResponseWriter, r *http.Request, roleCode string) {
	// 1. Validate API version
	if apiVersionParam := r.URL.Query().Get("api-version"); apiVersionParam != apiVersion {
		sendErrorResponse(w, r, http.StatusBadRequest, "UNSUPPORTED_API_VERSION", "API version not supported", nil, s.config)
		return
	}

	if r.Method != http.MethodPost {
		sendErrorResponse(w, r, http.StatusMethodNotAllowed, "VALIDATION_ERROR", "Only POST method is allowed", nil, s.config)
		return
	}

	var req roleUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil, s.config)
		return
	}

	// Validate request has value field
	if req.Value == nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Missing required field: value", nil, s.config)
		return
	}

	// Validate request has at least one user
	if len(req.Value) == 0 {
		sendErrorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Must provide at least one userId", nil, s.config)
		return
	}

	// Validate users exist
	if invalidUsers, err := s.validateUsers(req.Value); err != nil {
		sendErrorResponse(w, r, http.StatusBadRequest, "NOT_FOUND", "Some users do not exist", invalidUsers, s.config)
		return
	}

	// Create a map for quick lookup of users to remove
	toRemove := make(map[string]bool)
	for _, id := range req.Value {
		toRemove[id] = true
	}

	// Filter out the users that should be removed
	existing := s.roleUsers[roleCode]
	filtered := make([]string, 0, len(existing))
	for _, id := range existing {
		if !toRemove[id] {
			filtered = append(filtered, id)
		}
	}
	s.roleUsers[roleCode] = filtered

	// Set rate limit headers
	addRateLimitHeaders(w, s.config)

	// Return 200 with no body as per API spec
	w.WriteHeader(http.StatusOK)
}

func sendErrorResponse(w http.ResponseWriter, _ *http.Request, statusCode int, code, message string, target interface{}, config *config) {
	response := struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Target  interface{} `json:"target,omitempty"`
	}{
		Code:    code,
		Message: message,
		Target:  target,
	}

	// Set rate limit headers even for error responses
	addRateLimitHeaders(w, config)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}

func sendJSONResponse(w http.ResponseWriter, r *http.Request, data interface{}, config *config) {
	responseJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		sendErrorResponse(w, r, http.StatusInternalServerError, "internal_error", "Internal server error", nil, config)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Correlation-Id", uuid.New().String())
	w.Header().Set("X-API-Version", apiVersion)

	logResponse(r.URL.Path, http.StatusOK, w.Header(), responseJSON)

	if _, err := w.Write(responseJSON); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func logRequest(endpoint string, r *http.Request) {
	log.Printf("=== INCOMING REQUEST ===")
	log.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))
	log.Printf("Endpoint: %s", endpoint)
	log.Printf("Method: %s", r.Method)
	log.Printf("URL: %s", r.URL.String())
	log.Printf("Query parameters: %s", r.URL.RawQuery)

	log.Println("Headers:")
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			log.Printf("Request body:\n%s", string(bodyBytes))
			// Restore the body for further processing
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		} else {
			log.Printf("Error reading request body: %v", err)
		}
	}
	log.Printf("=== END REQUEST ===\n")
}

func logResponse(endpoint string, statusCode int, headers http.Header, body []byte) {
	log.Printf("=== OUTGOING RESPONSE ===")
	log.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))
	log.Printf("Endpoint: %s", endpoint)
	log.Printf("Status Code: %d", statusCode)

	log.Println("Headers:")
	for name, values := range headers {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	log.Printf("Response body:\n%s", string(body))
	log.Printf("=== END RESPONSE ===\n")
}

func run() error {
	config := defaultConfig()

	// Allow port override via environment variable
	if port := os.Getenv("PORT"); port != "" {
		config.port = port
	}

	server := newServer(config)

	log.Printf("Starting test server with configuration: %+v", config)

	return server.Start()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
