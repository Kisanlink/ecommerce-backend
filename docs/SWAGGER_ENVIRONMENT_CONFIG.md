# Swagger Environment Configuration

This document describes how to configure Swagger documentation using environment variables.

## Environment Variables

### Core Documentation Control

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_DOCS` | `true` | Set to `false` in production to disable API documentation |

### Swagger API Information

| Variable | Default | Description |
|----------|---------|-------------|
| `SWAGGER_TITLE` | `Ecommerce Backend API` | API title displayed in documentation |
| `SWAGGER_DESCRIPTION` | `A comprehensive ecommerce backend API...` | API description |
| `SWAGGER_VERSION` | `1.0.0` | API version |
| `SWAGGER_HOST` | `localhost:8080` | API host |
| `SWAGGER_BASE_PATH` | `/api/v1` | API base path |

### Contact Information

| Variable | Default | Description |
|----------|---------|-------------|
| `SWAGGER_CONTACT_NAME` | `API Support` | Contact name |
| `SWAGGER_CONTACT_URL` | `http://www.kisanlink.com/support` | Contact URL |
| `SWAGGER_CONTACT_EMAIL` | `support@kisanlink.com` | Contact email |

### License Information

| Variable | Default | Description |
|----------|---------|-------------|
| `SWAGGER_LICENSE_NAME` | `MIT` | License name |
| `SWAGGER_LICENSE_URL` | `https://opensource.org/licenses/MIT` | License URL |

## Usage Examples

### Development Environment
```bash
# Enable documentation with custom title
ENABLE_DOCS=true
SWAGGER_TITLE="Ecommerce API - Development"
SWAGGER_HOST=localhost:8080
```

### Production Environment
```bash
# Disable documentation for security
ENABLE_DOCS=false
```

### Staging Environment
```bash
# Enable with staging-specific configuration
ENABLE_DOCS=true
SWAGGER_TITLE="Ecommerce API - Staging"
SWAGGER_HOST=staging-api.kisanlink.com
SWAGGER_CONTACT_EMAIL=staging-support@kisanlink.com
```

## Implementation

The configuration is implemented in:
- `internal/config/config.go` - Configuration struct and parsing
- `internal/swagger/swagger.go` - Swagger route registration

## Security Considerations

1. **Production**: Always set `ENABLE_DOCS=false` in production
2. **Sensitive Information**: Avoid exposing internal API details in descriptions
3. **Access Control**: Consider adding authentication to documentation endpoints if needed

## Generated Documentation

When `ENABLE_DOCS=true`, the following endpoints are available:
- `GET /docs` - Interactive API documentation (Scalar UI)
- `GET /docs/swagger.json` - OpenAPI JSON specification
- `GET /docs/swagger.yaml` - OpenAPI YAML specification

## Coupon API Documentation

The coupon endpoints are fully documented with Swagger annotations:

### Public Endpoints
- `GET /api/v1/coupons/{code}` - Get coupon by code
- `GET /api/v1/coupons/active` - Get active coupons
- `POST /api/v1/coupons/validate` - Validate coupon

### Authenticated Endpoints
- `POST /api/v1/coupons/apply` - Apply coupon to user
- `GET /api/v1/coupons/{code}/stats` - Get coupon usage statistics

### Admin Endpoints
- `POST /api/v1/coupons` - Create coupon
- `GET /api/v1/coupons` - Get all coupons
- `PUT /api/v1/coupons/{id}` - Update coupon
- `DELETE /api/v1/coupons/{id}` - Delete coupon

### Coupon Rules (Admin Only)
- `POST /api/v1/coupon-rules/{couponId}` - Create coupon rule
- `PUT /api/v1/coupon-rules/{couponId}/{ruleId}` - Update coupon rule
- `DELETE /api/v1/coupon-rules/{couponId}/{ruleId}` - Delete coupon rule
