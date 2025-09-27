package swagger

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	scalar "github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterSwaggerRoutes registers Scalar API documentation routes
func RegisterSwaggerRoutes(r *gin.Engine) {
	// Initialize logger for swagger routes
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Get Swagger configuration from environment
	swaggerConfig := config.GetSwaggerEnvConfig()

	// Serve OpenAPI and Scalar-powered docs UI (gated by env)
	if swaggerConfig.Enabled {
		r.StaticFile("/docs/swagger.json", "docs/swagger.json")
		r.StaticFile("/docs/swagger.yaml", "docs/swagger.yaml")
		r.GET("/docs", func(c *gin.Context) {
			scheme := "http"
			if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
				scheme = "https"
			}
			specURL := scheme + "://" + c.Request.Host + "/docs/swagger.json"

			htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
				SpecURL:  specURL,
				DarkMode: true,
				CustomOptions: scalar.CustomOptions{
					PageTitle: swaggerConfig.Title + " API Reference",
				},
			})
			if err != nil {
				c.String(http.StatusInternalServerError, "failed to render API docs: %v", err)
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
		})
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/docs")
		})

		// Add favicon route to avoid 401 errors
		r.GET("/favicon.ico", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
	} else {
		r.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, swaggerConfig.Title+" running")
		})
		logger.Info("Documentation endpoints disabled (ENABLE_DOCS=false)")
	}
}
