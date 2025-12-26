package service

import (
	"context"
	"testing"
	"time"

	"url-shortener/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ==================== MOCKS ====================

// MockURLRepository is a mock implementation of URLRepository
type MockURLRepository struct {
	mock.Mock
}

func (m *MockURLRepository) Create(ctx context.Context, url *domain.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockURLRepository) GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLRepository) GetByCustomAlias(ctx context.Context, alias string) (*domain.URL, error) {
	args := m.Called(ctx, alias)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLRepository) ExistsShortCode(ctx context.Context, shortCode string) (bool, error) {
	args := m.Called(ctx, shortCode)
	return args.Bool(0), args.Error(1)
}

func (m *MockURLRepository) ExistsCustomAlias(ctx context.Context, alias string) (bool, error) {
	args := m.Called(ctx, alias)
	return args.Bool(0), args.Error(1)
}

func (m *MockURLRepository) IncrementClicks(ctx context.Context, shortCode string) error {
	args := m.Called(ctx, shortCode)
	return args.Error(0)
}

func (m *MockURLRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockURLRepository) GetByID(ctx context.Context, id string) (*domain.URL, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockURLRepository) Update(ctx context.Context, url *domain.URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

// MockClickRepository is a mock implementation of ClickRepository
type MockClickRepository struct {
	mock.Mock
}

func (m *MockClickRepository) Create(ctx context.Context, click *domain.URLClick) error {
	args := m.Called(ctx, click)
	return args.Error(0)
}

func (m *MockClickRepository) GetByURLID(ctx context.Context, urlID string, limit, offset int) ([]*domain.URLClick, error) {
	args := m.Called(ctx, urlID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.URLClick), args.Error(1)
}

func (m *MockClickRepository) GetClickCount(ctx context.Context, urlID string) (int64, error) {
	args := m.Called(ctx, urlID)
	return args.Get(0).(int64), args.Error(1)
}

// MockCache is a mock implementation of Cache
type MockCache struct {
	mock.Mock
}

func (m *MockCache) GetURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockCache) SetURL(ctx context.Context, shortCode string, url *domain.URL) error {
	args := m.Called(ctx, shortCode, url)
	return args.Error(0)
}

func (m *MockCache) DeleteURL(ctx context.Context, shortCode string) error {
	args := m.Called(ctx, shortCode)
	return args.Error(0)
}

// ==================== TESTS ====================

func TestCreateShortURL_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockURLRepo := new(MockURLRepository)
	mockClickRepo := new(MockClickRepository)
	mockCache := new(MockCache)

	service := NewURLService(mockURLRepo, mockClickRepo, mockCache)

	// Mock expectations
	mockURLRepo.On("ExistsCustomAlias", ctx, "mylink").Return(false, nil)
	mockURLRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).Return(nil)
	mockCache.On("SetURL", ctx, "mylink", mock.AnythingOfType("*domain.URL")).Return(nil)

	// Act
	url, err := service.CreateShortURL(ctx, "https://example.com", "mylink", "user1", 0)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, "mylink", url.ShortCode)
	assert.Equal(t, "https://example.com", url.OriginalURL)
	assert.Equal(t, "user1", url.CreatedBy)
	mockURLRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestCreateShortURL_CustomAliasAlreadyExists(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockURLRepo := new(MockURLRepository)
	mockClickRepo := new(MockClickRepository)
	mockCache := new(MockCache)

	service := NewURLService(mockURLRepo, mockClickRepo, mockCache)

	// Mock: custom alias already exists
	mockURLRepo.On("ExistsCustomAlias", ctx, "taken").Return(true, nil)

	// Act
	url, err := service.CreateShortURL(ctx, "https://example.com", "taken", "user1", 0)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, url)
	assert.Contains(t, err.Error(), "custom alias already exists")
	mockURLRepo.AssertExpectations(t)
}

func TestCreateShortURL_WithExpiration(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockURLRepo := new(MockURLRepository)
	mockClickRepo := new(MockClickRepository)
	mockCache := new(MockCache)

	service := NewURLService(mockURLRepo, mockClickRepo, mockCache)

	mockURLRepo.On("ExistsShortCode", ctx, mock.Anything).Return(false, nil)
	mockURLRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).Return(nil)
	mockCache.On("SetURL", ctx, mock.Anything, mock.AnythingOfType("*domain.URL")).Return(nil)

	// Act
	url, err := service.CreateShortURL(ctx, "https://example.com", "", "user1", 24*time.Hour)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, url)
	assert.NotNil(t, url.ExpiresAt)
	assert.True(t, url.ExpiresAt.After(time.Now()))
}

func TestGetURL_CacheHit(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockURLRepo := new(MockURLRepository)
	mockClickRepo := new(MockClickRepository)
	mockCache := new(MockCache)

	service := NewURLService(mockURLRepo, mockClickRepo, mockCache)

	cachedURL := &domain.URL{
		ID:          "123",
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	// Mock: cache hit
	mockCache.On("GetURL", ctx, "abc123").Return(cachedURL, nil)

	// Act
	url, err := service.GetURL(ctx, "abc123")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, cachedURL, url)
	mockCache.AssertExpectations(t)
	// Database should NOT be called (cache hit)
	mockURLRepo.AssertNotCalled(t, "GetByShortCode")
}

func TestGetURL_CacheMiss_DatabaseHit(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockURLRepo := new(MockURLRepository)
	mockClickRepo := new(MockClickRepository)
	mockCache := new(MockCache)

	service := NewURLService(mockURLRepo, mockClickRepo, mockCache)

	dbURL := &domain.URL{
		ID:          "123",
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	// Mock: cache miss, database hit
	mockCache.On("GetURL", ctx, "abc123").Return(nil, nil)
	mockURLRepo.On("GetByShortCode", ctx, "abc123").Return(dbURL, nil)
	mockCache.On("SetURL", ctx, "abc123", dbURL).Return(nil)

	// Act
	url, err := service.GetURL(ctx, "abc123")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, dbURL, url)
	mockCache.AssertExpectations(t)
	mockURLRepo.AssertExpectations(t)
}

func TestGetURL_ExpiredURL(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockURLRepo := new(MockURLRepository)
	mockClickRepo := new(MockClickRepository)
	mockCache := new(MockCache)

	service := NewURLService(mockURLRepo, mockClickRepo, mockCache)

	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredURL := &domain.URL{
		ID:          "123",
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		IsActive:    true,
		ExpiresAt:   &expiredTime,
	}

	mockCache.On("GetURL", ctx, "abc123").Return(expiredURL, nil)

	// Act
	url, err := service.GetURL(ctx, "abc123")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, url)
	assert.Contains(t, err.Error(), "expired")
}

func TestRecordClick_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockURLRepo := new(MockURLRepository)
	mockClickRepo := new(MockClickRepository)
	mockCache := new(MockCache)

	service := NewURLService(mockURLRepo, mockClickRepo, mockCache)

	url := &domain.URL{
		ID:          "123",
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
	}

	mockURLRepo.On("GetByShortCode", ctx, "abc123").Return(url, nil)
	mockURLRepo.On("IncrementClicks", ctx, "abc123").Return(nil)
	mockClickRepo.On("Create", ctx, mock.AnythingOfType("*domain.URLClick")).Return(nil)

	// Act
	err := service.RecordClick(ctx, "abc123", "192.168.1.1", "Mozilla/5.0", "https://google.com")

	// Assert
	require.NoError(t, err)
	mockURLRepo.AssertExpectations(t)
	mockClickRepo.AssertExpectations(t)
}

// ==================== TABLE-DRIVEN TESTS ====================

func TestCreateShortURL_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		originalURL   string
		customAlias   string
		aliasExists   bool
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid URL without custom alias",
			originalURL: "https://example.com",
			customAlias: "",
			aliasExists: false,
			expectError: false,
		},
		{
			name:        "Valid URL with custom alias",
			originalURL: "https://example.com",
			customAlias: "mylink",
			aliasExists: false,
			expectError: false,
		},
		{
			name:          "Custom alias already taken",
			originalURL:   "https://example.com",
			customAlias:   "taken",
			aliasExists:   true,
			expectError:   true,
			errorContains: "already exists",
		},
		{
			name:          "Invalid URL",
			originalURL:   "not-a-valid-url",
			customAlias:   "",
			aliasExists:   false,
			expectError:   true,
			errorContains: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			ctx := context.Background()
			mockURLRepo := new(MockURLRepository)
			mockClickRepo := new(MockClickRepository)
			mockCache := new(MockCache)

			service := NewURLService(mockURLRepo, mockClickRepo, mockCache)

			if tt.customAlias != "" {
				mockURLRepo.On("ExistsCustomAlias", ctx, tt.customAlias).Return(tt.aliasExists, nil)
			} else {
				mockURLRepo.On("ExistsShortCode", ctx, mock.Anything).Return(false, nil)
			}

			if !tt.aliasExists && !tt.expectError {
				mockURLRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).Return(nil)
				mockCache.On("SetURL", ctx, mock.Anything, mock.AnythingOfType("*domain.URL")).Return(nil)
			}

			// Act
			url, err := service.CreateShortURL(ctx, tt.originalURL, tt.customAlias, "user1", 0)

			// Assert
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, url)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, url)
			}
		})
	}
}
