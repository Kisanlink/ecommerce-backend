# CORS Configuration Guide

This document explains how CORS (Cross-Origin Resource Sharing) is configured in the ecommerce backend.

## Overview

CORS is now handled entirely by the Go application middleware, providing better control and avoiding conflicts with nginx. The configuration is environment-based and supports different settings for different route groups.

## Environment Variables

Configure CORS using the following environment variables:

### Core CORS Settings

- `CORS_ALLOWED_ORIGINS`: Comma-separated list of allowed origins
  - Default: `http://localhost:3000,http://localhost:5173,http://127.0.0.1:5500,http://localhost:5500,https://amrti.com,https://www.amrti.com`
  - Example: `http://localhost:3000,https://amrti.com,https://www.amrti.com`

- `CORS_ALLOWED_METHODS`: Comma-separated list of allowed HTTP methods
  - Default: `GET,POST,PUT,DELETE,PATCH,OPTIONS`
  - Example: `GET,POST,PUT,DELETE,PATCH,OPTIONS`

- `CORS_ALLOWED_HEADERS`: Comma-separated list of allowed headers
  - Default: `Content-Type,Content-Length,Accept-Encoding,X-CSRF-Token,Authorization,Accept,Origin,Cache-Control,X-Requested-With,X-API-Key`
  - Example: `Content-Type,Authorization,X-API-Key`

- `CORS_EXPOSED_HEADERS`: Comma-separated list of exposed headers
  - Default: `X-Total-Count,X-Page-Count,X-Current-Page`
  - Example: `X-Total-Count,X-Page-Count`

### Advanced Settings

- `CORS_MAX_AGE`: Max age for preflight requests in seconds
  - Default: `86400` (24 hours)
  - Example: `3600` (1 hour)

- `CORS_ALLOW_CREDENTIALS`: Allow credentials (true/false)
  - Default: `true`
  - Example: `true`

- `CORS_ALLOW_WILDCARD`: Allow wildcard origin (true/false)
  - Default: `false`
  - **WARNING**: Only enable in development, never in production

- `CORS_DEBUG`: Enable CORS debug logging (true/false)
  - Default: `false`
  - Example: `true` (for debugging)

## Route-Specific CORS

### General API Routes
All API routes use the `APICORSConfig()` which includes:
- Standard API headers like `X-API-Version`, `X-Request-ID`
- All environment-configured settings

### Cart Routes
Cart routes use the `CartCORSConfig()` which includes additional headers:
- **Allowed Headers**: `X-Session-ID`, `X-Cart-ID`, `X-User-ID`
- **Exposed Headers**: `X-Cart-Total`, `X-Cart-Items`, `X-Coupon-Applied`

## Configuration Examples

### Development Environment
```bash
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
CORS_DEBUG=true
CORS_ALLOW_WILDCARD=false
```

### Production Environment
```bash
CORS_ALLOWED_ORIGINS=https://amrti.com,https://www.amrti.com
CORS_DEBUG=false
CORS_ALLOW_WILDCARD=false
CORS_MAX_AGE=3600
```

### Testing Environment
```bash
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173,https://staging.amrti.com
CORS_DEBUG=true
CORS_ALLOW_CREDENTIALS=true
```

## Debugging CORS Issues

1. Enable debug logging by setting `CORS_DEBUG=true`
2. Check the application logs for CORS-related messages
3. Verify that your frontend origin is included in `CORS_ALLOWED_ORIGINS`
4. Ensure required headers are included in `CORS_ALLOWED_HEADERS`

## Security Considerations

1. **Never use wildcard origins in production** (`CORS_ALLOW_WILDCARD=false`)
2. **Be specific with allowed origins** - only include domains you control
3. **Limit allowed methods** to only what your API actually uses
4. **Use HTTPS in production** for all origins
5. **Regularly review and update** allowed origins as your application grows

## Migration from Previous Configuration

The previous nginx-based CORS configuration has been removed to avoid conflicts. All CORS handling is now done by the Go middleware, providing:

- Better error handling and logging
- Environment-based configuration
- Route-specific CORS settings
- More granular control over headers and methods
