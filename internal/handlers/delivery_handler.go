package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"

	"github.com/gin-gonic/gin"
)

// ShipmentHandler handles shipment-related APIs
type ShipmentHandler struct {
	DelhiveryBaseURL string
	DelhiveryToken   string
}

// NewShipmentHandler creates new shipment handler
func NewShipmentHandler(baseURL, token string) *ShipmentHandler {
	return &ShipmentHandler{
		DelhiveryBaseURL: baseURL,
		DelhiveryToken:   token,
	}
}

// CreateShipment godoc
// @Summary Create shipment in Delhivery
// @Description Creates an order shipment and returns waybill + tracking URL
// @Tags Shipment
// @Accept json
// @Produce json
// @Param shipment body requests.Shipment true "Shipment Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /shipment [post]
func (h *ShipmentHandler) CreateShipment(c *gin.Context) {
	var shipmentReq requests.DelhiveryShipmentRequest
	if err := c.ShouldBindJSON(&shipmentReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shipment payload"})
		return
	}

	// Prepare payload: Delhivery expects format=json&data=<json> (x-www-form-urlencoded)
	payload := requests.NewCreateShipmentPayload(shipmentReq)
	dataJSON, _ := json.Marshal(payload.Data)

	form := url.Values{}
	form.Set("format", "json")
	form.Set("data", string(dataJSON))

	// Send request to Delhivery
	apiURL := fmt.Sprintf("%s/api/cmu/create.json", h.DelhiveryBaseURL)
	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Token "+h.DelhiveryToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to call Delhivery API"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Parse response
	shipmentResp, err := responses.ParseShipmentResponse(body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse Delhivery response"})
		return
	}

	// Validate success
	if !shipmentResp.IsSuccessful() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "shipment creation failed",
			"message": shipmentResp.GetErrorMessage(),
		})
		return
	}

	// Success → return waybill + tracking URL
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"waybill":      shipmentResp.GetWaybill(),
		"status":       shipmentResp.GetStatus(),
		"tracking_url": shipmentResp.GetTrackingURL(),
	})
}
