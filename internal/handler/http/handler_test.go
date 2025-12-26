package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"url-shortener/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ==================== MOCKS ====================

// MockURLService is a mock implementation of URLService
type MockURLService struct {
	mock.Mock
}

func (m *MockURLService) CreateShortURL(ctx context.Context, originalURL, customAlias, createdBy string, expiresIn time.Duration) (*domain.URL, error) {
	args := m.Called(ctx, originalURL, customAlias, createdBy, expiresIn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLService) GetURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLService) RecordClick(ctx context.Context, shortCode, ipAddress, userAgent, referer string) error {
	args := m.Called(ctx, shortCode, ipAddress, userAgent, referer)
	return args.Error(0)
}

func (m *MockURLService) GetURLStats(ctx context.Context, shortCode string) (*domain.URL, []*domain.URLClick, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*domain.URL), args.Get(1).([]*domain.URLClick), args.Error(2)
}

func (m *MockURLService) DeleteURL(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ==================== HELPER FUNCTIONS ====================

func setupTestHandler() (*Handler, *MockURLService) {
	mockService := new(MockURLService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewHandler(mockService, logger, "http://localhost:8080")
	return handler, mockService
}

// ==================== CREATE URL TESTS ====================

func TestCreateURL_Success(t *testing.T) {
	// Arrange
	handler, mockService := setupTestHandler()

	expectedURL := &domain.URL{
		ID:          "123",
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		CreatedBy:   "anonymous",
		CreatedAt:   time.Now(),
		IsActive:    true,
	}

	mockService.On("CreateShortURL", mock.Anything, "https://example.com", "", "anonymous", time.Duration(0)).
		Return(expectedURL, nil)

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "abc123", data["short_code"])
	assert.Equal(t, "https://example.com", data["original_url"])
	assert.Contains(t, data["short_url"], "abc123")

	mockService.AssertExpectations(t)
}

func TestCreateURL_WithCustomAlias(t *testing.T) {
	// Arrange
	handler, mockService := setupTestHandler()

	expectedURL := &domain.URL{
		ID:          "123",
		ShortCode:   "mylink",
		OriginalURL: "https://example.com",
		CustomAlias: stringPtr("mylink"),
		CreatedBy:   "anonymous",
		CreatedAt:   time.Now(),
		IsActive:    true,
	}

	mockService.On("CreateShortURL", mock.Anything, "https://example1.com", "mylink", "anonymous", time.Duration(0)).
		Return(expectedURL, nil)

	body := `{"url": "https://example1.com", "custom_alias": "mylink"}`
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "mylink", data["short_code"])

	mockService.AssertExpectations(t)
}

func TestCreateURL_WithExpiration(t *testing.T) {
	// Arrange
	handler, mockService := setupTestHandler()

	expiresAt := time.Now().Add(24 * time.Hour)
	expectedURL := &domain.URL{
		ID:          "123",
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		CreatedBy:   "anonymous",
		CreatedAt:   time.Now(),
		ExpiresAt:   &expiresAt,
		IsActive:    true,
	}

	mockService.On("CreateShortURL", mock.Anything, "https://example.com", "", "anonymous", 24*time.Hour).
		Return(expectedURL, nil)

	body := `{"url": "https://example.com", "expires_in_hours": 24}`
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestCreateURL_InvalidJSON(t *testing.T) {
	// Arrange
	handler, _ := setupTestHandler()

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "Invalid request")
}

func TestCreateURL_MissingURL(t *testing.T) {
	// Arrange
	handler, _ := setupTestHandler()

	body := `{"custom_alias": "test"}`
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "URL is required")
}

// ==================== REDIRECT URL TESTS ====================

func TestRedirectURL_Success(t *testing.T) {
	// Arrange
	handler, mockService := setupTestHandler()

	url := &domain.URL{
		ID:          "123",
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	mockService.On("GetURL", mock.Anything, "abc123").Return(url, nil)
	mockService.On("RecordClick", mock.Anything, "abc123", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest("GET", "/abc123", nil)
	w := httptest.NewRecorder()

	// Act
	handler.RedirectURL(w, req)

	// Assert
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))

	mockService.AssertExpectations(t)
}

func TestRedirectURL_NotFound(t *testing.T) {
	// Arrange
	handler, mockService := setupTestHandler()

	mockService.On("GetURL", mock.Anything, "notfound").Return(nil, assert.AnError)

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	// Act
	handler.RedirectURL(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "not found")

	mockService.AssertExpectations(t)
}

// ==================== GET URL STATS TESTS ====================

func TestGetURLStats_Success(t *testing.T) {
	// Arrange
	handler, mockService := setupTestHandler()

	url := &domain.URL{
		ID:          "123",
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		Clicks:      42,
		IsActive:    true,
	}

	clicks := []*domain.URLClick{
		{
			ID:        1,
			URLID:     "123",
			IPAddress: "192.168.1.1",
			ClickedAt: time.Now(),
		},
	}

	mockService.On("GetURLStats", mock.Anything, "abc123").Return(url, clicks, nil)

	req := httptest.NewRequest("GET", "/api/v1/urls/abc123/stats", nil)
	w := httptest.NewRecorder()

	// Act
	handler.GetURLStats(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "abc123", data["short_code"])
	assert.Equal(t, float64(42), data["clicks"]) // JSON numbers are float64

	mockService.AssertExpectations(t)
}

// ==================== HEALTH CHECK TESTS ====================

func TestHealthCheck(t *testing.T) {
	// Arrange
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()

	// Act
	handler.HealthCheck(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "ok", response["status"])
	assert.NotEmpty(t, response["time"])
}

// ==================== TABLE-DRIVEN TESTS ====================

func TestCreateURL_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		mockSetup      func(*MockURLService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:        "Valid URL",
			requestBody: `{"url": "https://example.com"}`,
			mockSetup: func(m *MockURLService) {
				url := &domain.URL{ShortCode: "abc123", OriginalURL: "https://example.com"}
				m.On("CreateShortURL", mock.Anything, "https://example.com", "", "anonymous", time.Duration(0)).
					Return(url, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "abc123", data["short_code"])
			},
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{invalid}`,
			mockSetup:      func(m *MockURLService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "Invalid request")
			},
		},
		{
			name:           "Empty URL",
			requestBody:    `{"url": ""}`,
			mockSetup:      func(m *MockURLService) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp["error"], "required")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			handler, mockService := setupTestHandler()
			tt.mockSetup(mockService)

			req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Act
			handler.CreateURL(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

// ==================== HELPER FUNCTIONS ====================

func stringPtr(s string) *string {
	return &s
}
