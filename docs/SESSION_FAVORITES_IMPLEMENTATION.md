# Session-Based Favorites Implementation

This document describes the implementation of session-based favorites functionality that allows users to add, remove, and view their favorites even when not logged in, similar to the cart system.

## Overview

The session-based favorites system provides:
- **Anonymous user support**: Users can manage favorites without authentication
- **Session persistence**: Favorites are stored in memory using session IDs
- **Seamless migration**: Session favorites can be migrated to user favorites after login
- **Merge functionality**: Session favorites can be merged with existing user favorites
- **Consistent API**: Same endpoints work for both authenticated and anonymous users

## Architecture

### Models

#### SessionFavorite
```go
type SessionFavorite struct {
    *base.BaseModel `gorm:"embedded"`
    SessionID       string   `json:"session_id" gorm:"not null;index"`
    ProductID       string   `json:"product_id" gorm:"not null;index"`
    Product         *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ID"`
    AddedAt         time.Time `json:"added_at"`
}
```

#### SessionFavoriteList
```go
type SessionFavoriteList struct {
    SessionID   string             `json:"session_id"`
    Favorites   []*SessionFavorite `json:"favorites"`
    TotalCount  int                `json:"total_count"`
    LastUpdated time.Time          `json:"last_updated"`
}
```

### Services

#### SessionFavoriteService
- **In-memory storage**: Uses `map[string]*SessionFavoriteList` for session storage
- **Session management**: Generates and validates session IDs
- **Product validation**: Ensures products exist before adding to favorites
- **Migration support**: Migrates session favorites to user favorites
- **Merge functionality**: Merges session favorites with existing user favorites

### Handlers

#### SessionFavoriteHandler
- **Dual mode operation**: Handles both authenticated and anonymous users
- **Session ID management**: Generates session IDs for anonymous users
- **Consistent responses**: Returns session ID in responses for anonymous users
- **Error handling**: Comprehensive error handling with proper HTTP status codes

## API Endpoints

### Public Endpoints (No Authentication Required)

All endpoints work for both authenticated and anonymous users:

#### GET /api/v1/favorites
Get user's favorites (authenticated) or session favorites (anonymous)

**Headers:**
- `X-Session-ID` (optional): Session ID for anonymous users
- `Authorization` (optional): Bearer token for authenticated users

**Query Parameters:**
- `session_id` (optional): Alternative to header for session ID

**Response:**
```json
{
  "success": true,
  "favorites": [...],
  "is_authenticated": true/false,
  "session_id": "session_abc123" // Only for anonymous users
}
```

#### POST /api/v1/favorites
Add a product to favorites

**Request Body:**
```json
{
  "product_id": "PROD_123"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Product added to favorites successfully",
  "is_authenticated": true/false,
  "session_id": "session_abc123" // Only for anonymous users
}
```

#### DELETE /api/v1/favorites
Remove a product from favorites

**Request Body:**
```json
{
  "product_id": "PROD_123"
}
```

#### GET /api/v1/favorites/check?product_id=PROD_123
Check if a product is in favorites

**Response:**
```json
{
  "success": true,
  "is_favorite": true/false,
  "product_id": "PROD_123",
  "is_authenticated": true/false,
  "session_id": "session_abc123" // Only for anonymous users
}
```

#### DELETE /api/v1/favorites/clear
Clear all favorites

### Authenticated Endpoints (Authentication Required)

#### POST /api/v1/favorites/migrate
Migrate session favorites to user favorites after login

**Headers:**
- `Authorization`: Bearer token
- `X-Session-ID`: Session ID to migrate

**Response:**
```json
{
  "success": true,
  "message": "Favorites migrated successfully",
  "user_id": "USER_123"
}
```

#### POST /api/v1/favorites/merge
Merge session favorites with user favorites after login

**Headers:**
- `Authorization`: Bearer token
- `X-Session-ID`: Session ID to merge

**Response:**
```json
{
  "success": true,
  "message": "Favorites merged successfully",
  "user_id": "USER_123"
}
```

## Usage Examples

### Anonymous User Flow

1. **Add to favorites** (first time - generates session ID):
```bash
curl -X POST http://localhost:8080/api/v1/favorites \
  -H "Content-Type: application/json" \
  -d '{"product_id": "PROD_123"}'
```

Response includes `session_id`:
```json
{
  "success": true,
  "message": "Product added to favorites successfully",
  "is_authenticated": false,
  "session_id": "session_abc123"
}
```

2. **Use session ID for subsequent requests**:
```bash
curl -X GET http://localhost:8080/api/v1/favorites \
  -H "X-Session-ID: session_abc123"
```

3. **After login, migrate favorites**:
```bash
curl -X POST http://localhost:8080/api/v1/favorites/migrate \
  -H "Authorization: Bearer <token>" \
  -H "X-Session-ID: session_abc123"
```

### Authenticated User Flow

1. **Add to favorites** (uses user ID):
```bash
curl -X POST http://localhost:8080/api/v1/favorites \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"product_id": "PROD_123"}'
```

2. **Get favorites**:
```bash
curl -X GET http://localhost:8080/api/v1/favorites \
  -H "Authorization: Bearer <token>"
```

## Implementation Details

### Session ID Generation
- Uses cryptographically secure random bytes
- Format: `session_<hex_encoded_bytes>`
- Fallback to timestamp-based ID if random generation fails

### Session Validation
- Checks session ID format and length
- Validates session ID before any operation

### In-Memory Storage
- Uses Go map for fast access
- Session data persists for the lifetime of the service
- No persistence to database (intentional for session-based approach)

### Product Validation
- Validates product existence before adding to favorites
- Handles product not found errors gracefully

### Migration vs Merge
- **Migration**: Moves all session favorites to user favorites, clears session
- **Merge**: Adds session favorites to user favorites (skips duplicates), clears session

## Error Handling

### Common Error Responses

#### Invalid Session ID
```json
{
  "success": false,
  "error": "Invalid session ID"
}
```

#### Product Not Found
```json
{
  "success": false,
  "error": "Product not found"
}
```

#### Already in Favorites
```json
{
  "success": false,
  "error": "Product is already in favorites"
}
```

#### Favorite Not Found
```json
{
  "success": false,
  "error": "Favorite not found"
}
```

## Integration with Existing System

### Service Manager
The `SessionFavoriteService` is automatically initialized in the `ServiceManager` and available throughout the application.

### Route Registration
Routes are registered in `internal/routes/favorites/favorite_routes.go` with:
- Optional authentication middleware
- Error recovery middleware
- Fallback handlers for service unavailability

### Backward Compatibility
- Original authenticated-only routes are preserved
- New session-based routes handle both authenticated and anonymous users
- No breaking changes to existing API consumers

## Testing

### Unit Tests
- Test session ID generation and validation
- Test favorite operations (add, remove, check, clear)
- Test migration and merge functionality
- Test error handling scenarios

### Integration Tests
- Test API endpoints with and without authentication
- Test session persistence across requests
- Test migration after login flow

## Security Considerations

### Session ID Security
- Session IDs are generated using cryptographically secure random bytes
- No sensitive data is stored in session IDs
- Session IDs are validated before use

### Data Isolation
- Each session has its own isolated favorites list
- No cross-session data access
- Session data is not persisted to database

### Authentication
- Optional authentication middleware allows both authenticated and anonymous access
- Authenticated users get their persistent favorites
- Anonymous users get session-based favorites

## Performance Considerations

### Memory Usage
- In-memory storage is fast but consumes RAM
- Consider implementing cleanup for old sessions
- Monitor memory usage in production

### Scalability
- Current implementation is single-instance
- For multi-instance deployment, consider Redis-based session storage
- Session data is not shared between instances

## Future Enhancements

### Potential Improvements
1. **Redis Integration**: Store session data in Redis for multi-instance support
2. **Session Cleanup**: Implement automatic cleanup of old sessions
3. **Session Persistence**: Optional database persistence for session data
4. **Rate Limiting**: Add rate limiting for session operations
5. **Analytics**: Track session favorite usage patterns

### Configuration Options
- Session timeout duration
- Maximum favorites per session
- Cleanup interval for old sessions
- Redis configuration for distributed sessions

## Conclusion

The session-based favorites implementation provides a seamless experience for both authenticated and anonymous users, similar to the cart system. It maintains backward compatibility while adding new functionality for anonymous users, with easy migration paths when users log in.
