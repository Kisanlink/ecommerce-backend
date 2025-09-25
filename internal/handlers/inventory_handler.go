package handlers

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/gin-gonic/gin"
)

// InventoryHandler handles inventory-related HTTP requests
type InventoryHandler struct {
	inventoryService *services.InventoryService
}

// NewInventoryHandler creates a new inventory handler
func NewInventoryHandler(inventoryService *services.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
	}
}

// CreateInventory handles POST /api/v1/inventory
// @Summary Create a new inventory record
// @Description Create a new inventory record for a product (Admin only)
// @Tags Inventory
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.CreateInventoryRequest true "Inventory information"
// @Success 201 {object} responses.InventoryCreateResponse "Inventory created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory [post]
func (h *InventoryHandler) CreateInventory(c *gin.Context) {
	var req requests.CreateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set user ID from context
	req.SetUserID(getUserFromContext(c))

	response, err := h.inventoryService.CreateInventory(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusCreated, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// GetInventoryByProductID handles GET /api/v1/inventory/product/:product_id
// @Summary Get inventory by product ID
// @Description Get inventory information for a specific product
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Success 200 {object} responses.InventoryResponse "Inventory retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid product ID"
// @Failure 404 {object} map[string]interface{} "Inventory not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/product/{product_id} [get]
func (h *InventoryHandler) GetInventoryByProductID(c *gin.Context) {
	var req requests.GetInventoryRequest
	req.ProductID = c.Param("product_id")

	if req.ProductID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response, err := h.inventoryService.GetInventoryByProductID(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusNotFound, response)
	}
}

// UpdateInventory handles PUT /api/v1/inventory/product/:product_id
// @Summary Update inventory
// @Description Update inventory information for a product (Admin only)
// @Tags Inventory
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param product_id path string true "Product ID"
// @Param request body requests.UpdateInventoryRequest true "Inventory information"
// @Success 200 {object} responses.InventoryUpdateResponse "Inventory updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Admin access required"
// @Failure 404 {object} map[string]interface{} "Inventory not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/product/{product_id} [put]
func (h *InventoryHandler) UpdateInventory(c *gin.Context) {
	var req requests.UpdateInventoryRequest
	req.ProductID = c.Param("product_id")

	if req.ProductID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set user ID from context
	req.SetUserID(getUserFromContext(c))

	response, err := h.inventoryService.UpdateInventory(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// DeleteInventory handles DELETE /api/v1/inventory/product/:product_id
// @Summary Delete inventory
// @Description Delete inventory record for a product (Admin only)
// @Tags Inventory
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param product_id path string true "Product ID"
// @Success 200 {object} responses.InventoryDeleteResponse "Inventory deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid product ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Admin access required"
// @Failure 404 {object} map[string]interface{} "Inventory not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/product/{product_id} [delete]
func (h *InventoryHandler) DeleteInventory(c *gin.Context) {
	var req requests.DeleteInventoryRequest
	req.ProductID = c.Param("product_id")
	req.SetUserID(getUserFromContext(c))

	if req.ProductID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response, err := h.inventoryService.DeleteInventory(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListInventory handles GET /api/v1/inventory
// @Summary Get all inventory records
// @Description Get a paginated list of all inventory records with optional filtering
// @Tags Inventory
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param product_id query string false "Filter by product ID"
// @Param stock_status query string false "Filter by stock status"
// @Param low_stock query bool false "Filter low stock items"
// @Param out_of_stock query bool false "Filter out of stock items"
// @Param needs_reorder query bool false "Filter items needing reorder"
// @Success 200 {object} responses.InventoryListPaginatedResponse "Inventory retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid query parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory [get]
func (h *InventoryHandler) ListInventory(c *gin.Context) {
	var req requests.ListInventoryRequest

	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		response := base.NewErrorResponse("Invalid query parameters",
			base.NewValidationError("Parameter binding failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set default pagination if not provided
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PerPage == 0 {
		req.PerPage = 20
	}

	response, err := h.inventoryService.ListInventory(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateStock handles PUT /api/v1/inventory/stock/:product_id
// @Summary Update stock
// @Description Update stock quantity for a product (Admin only)
// @Tags Inventory
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param product_id path string true "Product ID"
// @Param request body requests.UpdateStockRequest true "Stock information"
// @Success 200 {object} responses.StockUpdateResponse "Stock updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Admin access required"
// @Failure 404 {object} map[string]interface{} "Inventory not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/stock/{product_id} [put]
func (h *InventoryHandler) UpdateStock(c *gin.Context) {
	var req requests.UpdateStockRequest
	req.ProductID = c.Param("product_id")

	if req.ProductID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set user ID from context
	req.SetUserID(getUserFromContext(c))

	response, err := h.inventoryService.UpdateStock(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// ReserveStock handles POST /api/v1/inventory/reserve/:product_id
// @Summary Reserve stock
// @Description Reserve stock for a product (typically for pending orders)
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param request body requests.ReserveStockRequest true "Reserve information"
// @Success 200 {object} responses.StockReserveResponse "Stock reserved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 404 {object} map[string]interface{} "Inventory not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/reserve/{product_id} [post]
func (h *InventoryHandler) ReserveStock(c *gin.Context) {
	var req requests.ReserveStockRequest
	req.ProductID = c.Param("product_id")

	if req.ProductID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response, err := h.inventoryService.ReserveStock(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// ReleaseStock handles POST /api/v1/inventory/release/:product_id
// @Summary Release stock
// @Description Release reserved stock for a product
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param request body requests.ReleaseStockRequest true "Release information"
// @Success 200 {object} responses.StockReleaseResponse "Stock released successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 404 {object} map[string]interface{} "Inventory not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/release/{product_id} [post]
func (h *InventoryHandler) ReleaseStock(c *gin.Context) {
	var req requests.ReleaseStockRequest
	req.ProductID = c.Param("product_id")

	if req.ProductID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response, err := h.inventoryService.ReleaseStock(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// SellStock handles POST /api/v1/inventory/sell/:product_id
// @Summary Sell stock
// @Description Process a sale and update stock
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param request body requests.SellStockRequest true "Sale information"
// @Success 200 {object} responses.StockSellResponse "Stock sold successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 404 {object} map[string]interface{} "Inventory not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/sell/{product_id} [post]
func (h *InventoryHandler) SellStock(c *gin.Context) {
	var req requests.SellStockRequest
	req.ProductID = c.Param("product_id")

	if req.ProductID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set user ID from context
	req.SetUserID(getUserFromContext(c))

	response, err := h.inventoryService.SellStock(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// Restock handles POST /api/v1/inventory/restock/:product_id
// @Summary Restock
// @Description Add stock to inventory
// @Tags Inventory
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param request body requests.RestockRequest true "Restock information"
// @Success 200 {object} responses.StockRestockResponse "Stock restocked successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 404 {object} map[string]interface{} "Inventory not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/restock/{product_id} [post]
func (h *InventoryHandler) Restock(c *gin.Context) {
	var req requests.RestockRequest
	req.ProductID = c.Param("product_id")

	if req.ProductID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set user ID from context
	req.SetUserID(getUserFromContext(c))

	response, err := h.inventoryService.Restock(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// GetLowStockItems handles GET /api/v1/inventory/low-stock
// @Summary Get low stock items
// @Description Get all items with low stock
// @Tags Inventory
// @Accept json
// @Produce json
// @Success 200 {object} responses.LowStockItemsResponse "Low stock items retrieved successfully"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/low-stock [get]
func (h *InventoryHandler) GetLowStockItems(c *gin.Context) {
	var req requests.GetLowStockItemsRequest

	response, err := h.inventoryService.GetLowStockItems(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetOutOfStockItems handles GET /api/v1/inventory/out-of-stock
// @Summary Get out of stock items
// @Description Get all out of stock items
// @Tags Inventory
// @Accept json
// @Produce json
// @Success 200 {object} responses.OutOfStockItemsResponse "Out of stock items retrieved successfully"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/out-of-stock [get]
func (h *InventoryHandler) GetOutOfStockItems(c *gin.Context) {
	var req requests.GetOutOfStockItemsRequest

	response, err := h.inventoryService.GetOutOfStockItems(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetItemsNeedingReorder handles GET /api/v1/inventory/needs-reorder
// @Summary Get items needing reorder
// @Description Get all items that need to be reordered
// @Tags Inventory
// @Accept json
// @Produce json
// @Success 200 {object} responses.ItemsNeedingReorderResponse "Items needing reorder retrieved successfully"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /inventory/needs-reorder [get]
func (h *InventoryHandler) GetItemsNeedingReorder(c *gin.Context) {
	var req requests.GetItemsNeedingReorderRequest

	response, err := h.inventoryService.GetItemsNeedingReorder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}
