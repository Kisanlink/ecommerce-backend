package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CORSMiddlewareWithConfig allows custom CORS configuration
func CORSMiddlewareWithConfig(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Initialize logger if debug is enabled
		var logger *zap.Logger
		if config.Debug {
			logger, _ = zap.NewDevelopment()
			defer logger.Sync()
		}

		// Check if origin is allowed
		originAllowed := false
		var allowedOrigin string

		for _, allowed := range config.AllowedOrigins {
			if allowed == "*" && config.AllowWildcard {
				originAllowed = true
				allowedOrigin = "*"
				break
			} else if origin == allowed {
				originAllowed = true
				allowedOrigin = origin
				break
			}
		}

		// Set CORS headers
		if originAllowed {
			// If wildcard is allowed but credentials are enabled, return the specific origin instead of '*'
			if allowedOrigin == "*" && config.AllowCredentials && origin != "" {
				allowedOrigin = origin
				c.Header("Vary", "Origin")
			}
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
		} else if origin != "" && config.Debug && logger != nil {
			logger.Warn("CORS: Origin not allowed",
				zap.String("origin", origin),
				zap.Strings("allowed_origins", config.AllowedOrigins),
				zap.String("path", c.Request.URL.Path),
			)
		}

		// Set credentials header
		if config.AllowCredentials && originAllowed {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Set allowed headers
		if len(config.AllowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
		}

		// Set allowed methods
		if len(config.AllowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
		}

		// Set exposed headers
		if len(config.ExposedHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
		}

		// Set max age for preflight requests
		if config.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if config.Debug && logger != nil {
				logger.Info("CORS: Handling preflight request",
					zap.String("origin", origin),
					zap.String("method", c.Request.Header.Get("Access-Control-Request-Method")),
					zap.String("headers", c.Request.Header.Get("Access-Control-Request-Headers")),
				)
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
	AllowWildcard    bool
	Debug            bool
}

// GetCORSOriginsFromEnv reads allowed origins from environment variables
func GetCORSOriginsFromEnv() []string {
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origins == "" {
		// Default development origins
		return []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:5500",
			"http://localhost:5500",
			"https://amrti.com",
			"https://www.amrti.com",
		}
	}

	// Split comma-separated origins
	return strings.Split(origins, ",")
}

// GetCORSMethodsFromEnv reads allowed methods from environment variables
func GetCORSMethodsFromEnv() []string {
	methods := os.Getenv("CORS_ALLOWED_METHODS")
	if methods == "" {
		return []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	}
	return strings.Split(methods, ",")
}

// GetCORSHeadersFromEnv reads allowed headers from environment variables
func GetCORSHeadersFromEnv() []string {
	headers := os.Getenv("CORS_ALLOWED_HEADERS")
	if headers == "" {
		return []string{
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"Accept",
			"Origin",
			"Cache-Control",
			"X-Requested-With",
			"X-API-Key",
		}
	}
	return strings.Split(headers, ",")
}

// GetCORSExposedHeadersFromEnv reads exposed headers from environment variables
func GetCORSExposedHeadersFromEnv() []string {
	headers := os.Getenv("CORS_EXPOSED_HEADERS")
	if headers == "" {
		return []string{
			"X-Total-Count",
			"X-Page-Count",
			"X-Current-Page",
		}
	}
	return strings.Split(headers, ",")
}

// GetCORSMaxAgeFromEnv reads max age from environment variables
func GetCORSMaxAgeFromEnv() int {
	maxAge := os.Getenv("CORS_MAX_AGE")
	if maxAge == "" {
		return 86400 // 24 hours
	}

	// Parse max age (basic implementation)
	if maxAge == "0" {
		return 0
	}
	return 86400 // Default to 24 hours
}

// GetCORSAllowCredentialsFromEnv reads allow credentials from environment variables
func GetCORSAllowCredentialsFromEnv() bool {
	allowCreds := os.Getenv("CORS_ALLOW_CREDENTIALS")
	return allowCreds == "true" || allowCreds == "1"
}

// GetCORSAllowWildcardFromEnv reads allow wildcard from environment variables
func GetCORSAllowWildcardFromEnv() bool {
	allowWildcard := os.Getenv("CORS_ALLOW_WILDCARD")
	return allowWildcard == "true" || allowWildcard == "1"
}

// GetCORSDebugFromEnv reads debug mode from environment variables
func GetCORSDebugFromEnv() bool {
	debug := os.Getenv("CORS_DEBUG")
	return debug == "true" || debug == "1"
}

// DefaultCORSConfig returns a default CORS configuration based on environment variables
func DefaultCORSConfig() CORSConfig {
	cfg := CORSConfig{
		AllowedOrigins:   GetCORSOriginsFromEnv(),
		AllowedMethods:   GetCORSMethodsFromEnv(),
		AllowedHeaders:   GetCORSHeadersFromEnv(),
		ExposedHeaders:   GetCORSExposedHeadersFromEnv(),
		AllowCredentials: GetCORSAllowCredentialsFromEnv(),
		MaxAge:           GetCORSMaxAgeFromEnv(),
		AllowWildcard:    GetCORSAllowWildcardFromEnv(),
		Debug:            GetCORSDebugFromEnv(),
	}

	// Ensure OPTIONS is always permitted for preflight requests
	hasOptions := false
	for _, m := range cfg.AllowedMethods {
		if strings.EqualFold(strings.TrimSpace(m), "OPTIONS") {
			hasOptions = true
			break
		}
	}
	if !hasOptions {
		cfg.AllowedMethods = append(cfg.AllowedMethods, "OPTIONS")
	}

	return cfg
}

// CartCORSConfig returns a CORS configuration specifically for cart routes
func CartCORSConfig() CORSConfig {
	config := DefaultCORSConfig()
	// Cart routes might need additional headers for session management
	config.AllowedHeaders = append(config.AllowedHeaders,
		"X-Session-ID",
		"X-Cart-ID",
		"X-User-ID",
	)
	config.ExposedHeaders = append(config.ExposedHeaders,
		"X-Cart-Total",
		"X-Cart-Items",
		"X-Coupon-Applied",
	)
	return config
}

// APICORSConfig returns a CORS configuration for general API routes
func APICORSConfig() CORSConfig {
	config := DefaultCORSConfig()
	// API routes might need additional headers
	config.AllowedHeaders = append(config.AllowedHeaders,
		"X-API-Version",
		"X-Request-ID",
		// Allow session header used by frontend to avoid failing preflights
		"X-Session-ID",
	)
	return config
}
