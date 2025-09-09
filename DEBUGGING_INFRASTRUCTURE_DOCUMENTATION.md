# Debugging Infrastructure Documentation

## Problem Statement

**Issue**: Payment verification works locally but fails in AWS deployment. Specifically:
- Local environment: AWB number and tracking URL are generated correctly
- AWS environment: Payment shows `success: true` but AWB and tracking URL are not set
- Root cause: Unknown - requires detailed logging to identify failure point

## Solution Overview

Comprehensive debugging infrastructure has been implemented across the payment and shipment creation flow to capture detailed diagnostic information in the AWS environment.

## System Architecture

### Payment and Shipment Flow Architecture
```

                http://localhost:8080/api/v1/payments/verify
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client/App    │───▶│  Payment Verify │───▶│   Order Create  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │                        │
                              ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Razorpay API  │◀───│ Signature Check │    │   Database      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                                                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Delhivery API  │◀───│ Shipment Create │◀───│  Order Data     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                      │
        ▼                      ▼
┌─────────────────┐    ┌─────────────────┐
│ AWB Generation  │───▶│   Success/Fail  │
└─────────────────┘    └─────────────────┘
```


http://localhost:8080/api/v1/orders/order_id


### Component Interaction Flow
1. **Payment Handler** receives verification request
2. **Razorpay Service** validates payment signature
3. **Order Service** creates order in database
4. **Delivery Service** calls Delhivery API
5. **Response Processing** extracts AWB and tracking URL
6. **Database Update** stores shipping information

## Debugging Strategy

### 1. Flow Markers
- **Purpose**: Track execution flow through the payment verification and shipment creation process
- **Implementation**: Strategic log markers with `=== FLOW_STEP ===` format for easy identification
- **Benefit**: Quickly identify where the process stops or fails

### 2. Detailed API Logging
- **Purpose**: Capture complete request/response data for Delhivery API calls
- **Implementation**: Full request body, headers, and response logging
- **Benefit**: Identify API communication issues or response parsing problems

### 3. Configuration Validation
- **Purpose**: Ensure all required configurations are present and valid
- **Implementation**: Runtime validation with detailed error reporting
- **Benefit**: Catch configuration-related issues in different environments

## Enhanced Files and Checkpoints

### 1. Payment Handler (`internal/handlers/payment_handler.go`)

**Enhanced Method**: `VerifyPayment`

**Method Architecture**:
```go
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
    // Step 1: Request Parsing and Validation
    // Step 2: Razorpay Signature Verification  
    // Step 3: Order Creation in Database
    // Step 4: Delhivery Shipment Creation
    // Step 5: Response Processing and AWB Extraction
    // Step 6: Final Response with Tracking Details
}
```

**Detailed Flow Checkpoints**:
```
=== STARTING PAYMENT VERIFICATION ===
├── Request parsing and validation
├── Extracting payment_id, order_id, signature
└── Input validation complete

=== RAZORPAY SIGNATURE VERIFICATION ===
├── Constructing signature string
├── Computing HMAC SHA256 hash
├── Comparing with received signature
└── Payment authenticity confirmed

=== ORDER CREATION PROCESS ===
├── Fetching cart items from database
├── Calculating total amount
├── Creating order record
├── Updating cart status
└── Order successfully created

=== STARTING DELHIVERY SHIPMENT CREATION FLOW ===
├── Preparing shipment data from order
├── Calling delivery service
├── Processing shipment response
└── Extracting AWB and tracking URL

=== SHIPMENT CREATION COMPLETED ===
├── AWB number validation
├── Tracking URL validation  
├── Database update with shipping info
└── Final response preparation
```

**Comprehensive Logging Details**:
- **Input Validation**: Log all received parameters with masking for sensitive data
- **Signature Verification**: Log signature construction process and verification result
- **Database Operations**: Log order creation status, cart updates, and any database errors
- **Shipment Request**: Log complete shipment data being sent to Delhivery
- **API Response**: Log full Delhivery response with parsing details
- **AWB Extraction**: Log AWB number extraction process and validation
- **Error Handling**: Log detailed error information with context at each step

**Critical Debug Points**:
- **Line ~180-190**: Request parsing and input validation
- **Line ~200-210**: Razorpay signature construction and verification
- **Line ~220-240**: Database order creation and cart management
- **Line ~250-260**: Delhivery shipment creation initiation
- **Line ~270-290**: Response processing and AWB extraction
- **Line ~300-310**: Final response construction and error handling

### 2. Delivery Service (`internal/services/delivery_service.go`)

**Enhanced Method**: `CreateShipment`

**Method Architecture**:
```go
func (s *DeliveryService) CreateShipment(shipmentData ShipmentRequest) (*ShipmentResponse, error) {
    // Step 1: Configuration Validation
    // Step 2: Request Data Preparation  
    // Step 3: HTTP Client Setup
    // Step 4: API Call Execution
    // Step 5: Response Processing
    // Step 6: AWB and Tracking URL Extraction
    // Step 7: Response Validation and Return
}
```

**Detailed Flow Checkpoints**:
```
=== DELHIVERY CONFIG VALIDATION ===
├── API Token validation (presence, format)
├── Base URL validation (format, accessibility)
├── Warehouse configuration check
├── Environment-specific settings validation
└── Configuration validation complete

=== DELHIVERY API REQUEST PREPARATION ===
├── Shipment data transformation
├── Request payload construction
├── Header preparation (Authorization, Content-Type)
├── Request URL construction
└── Request preparation complete

=== DELHIVERY API CALL EXECUTION ===
├── HTTP client initialization
├── Request execution with timeout
├── Response status code check
├── Response body reading
└── API call execution complete

=== DELHIVERY API RESPONSE PROCESSING ===
├── Response body parsing (JSON)
├── Response structure validation
├── AWB number extraction
├── Tracking URL extraction
├── Response validation
└── Response processing complete
```

**Comprehensive Logging Details**:
- **Configuration Check**: Log all Delhivery config parameters (with token masking)
- **Request Construction**: Log complete request payload, headers, and URL
- **API Communication**: Log HTTP method, status code, response time
- **Response Analysis**: Log complete response body and parsing process
- **Data Extraction**: Log AWB extraction logic and validation steps
- **Error Scenarios**: Log detailed error information for each failure point

**Critical Debug Points**:
- **Line ~40-60**: Configuration validation and error handling
- **Line ~70-90**: Request data preparation and transformation
- **Line ~100-120**: HTTP client setup and request execution
- **Line ~130-150**: Response reading and initial validation
- **Line ~160-180**: JSON parsing and structure validation
- **Line ~190-210**: AWB and tracking URL extraction logic
- **Line ~220-240**: Final response construction and error handling

**Delhivery API Integration Details**:
- **Endpoint**: `POST /api/cmu/create.json`
- **Authentication**: Bearer token in Authorization header
- **Request Format**: JSON with shipment details
- **Response Format**: JSON with AWB and tracking information
- **Timeout Handling**: 30-second timeout with retry logic
- **Error Codes**: Detailed mapping of Delhivery error responses

### 3. Configuration Validation (`internal/config/delhivery.go`)

**New Function**: `ValidateDelhiveryConfig`

**Purpose**: Runtime validation of all Delhivery configuration parameters

**Validation Checks**:
- API Token presence and format
- Base URL validation
- Warehouse configuration
- Environment-specific settings

### Failure Identification Patterns

#### 1. Configuration Issues
```
=== DELHIVERY CONFIG VALIDATION ===
├── API Token: MISSING or INVALID
├── Base URL: INVALID format
└── Configuration validation: FAILED
```

#### 2. API Communication Issues
```
=== DELHIVERY API CALL EXECUTION ===
├── HTTP request timeout
├── Response status: 401 Unauthorized / 500 Internal Server Error
└── API call: FAILED
```

#### 3. Response Processing Issues
```
=== DELHIVERY API RESPONSE PROCESSING ===
├── Response body: empty or malformed JSON
├── AWB number extraction: FAILED - field not found
├── Tracking URL extraction: FAILED - invalid format
└── Response processing: FAILED
```

#### 4. Database Issues
```
=== ORDER CREATION PROCESS ===
├── Cart items fetch: FAILED - database connection error
├── Order creation: FAILED - constraint violation
└── Transaction rollback: COMPLETED
```

## Troubleshooting Common Issues

### Issue 1: Payment Verification Succeeds but No AWB Generated

**Symptoms**:
- Response shows `success: true`
- AWB number is empty or null
- Tracking URL is missing

**Debugging Steps**:
1. Check if flow reaches `=== STARTING DELHIVERY SHIPMENT CREATION FLOW ===`
2. Verify Delhivery configuration validation passes
3. Examine API request payload and response
4. Check AWB extraction logic

**Likely Causes**:
- Delhivery API configuration differences between local and AWS
- Network connectivity issues in AWS environment  
- Environment variable mismatches
- Delhivery API rate limiting or authentication issues

### Issue 2: Delhivery API Call Fails

**Symptoms**:
- Process stops at `=== DELHIVERY API CALL EXECUTION ===`
- HTTP errors (401, 403, 500)
- Timeout errors

**Debugging Steps**:
1. Verify API token is correctly set in AWS environment
2. Check network connectivity to Delhivery endpoints
3. Validate request format against Delhivery documentation

### Issue 3: Response Processing Fails

**Symptoms**:
- API call succeeds but AWB extraction fails
- Malformed response data
- Parsing errors in logs

**Debugging Steps**:
1. Compare response format between local and AWS
2. Check if Delhivery response structure changed
3. Validate JSON parsing logic
4. Review AWB field mapping

## Environment-Specific Configuration

### Local Environment
```yaml
DELHIVERY_API_TOKEN: "your-dev-token"
DELHIVERY_BASE_URL: "https://track.delhivery.com"
DELHIVERY_WAREHOUSE: "Amrti"
```

### AWS Environment  
```yaml
# Verify these match exactly in AWS deployment
DELHIVERY_API_TOKEN: "your-prod-token" 
DELHIVERY_BASE_URL: "https://track.delhivery.com"
DELHIVERY_WAREHOUSE: "Amrti"
```

### Configuration Validation Checklist
- [ ] All environment variables are set
- [ ] API token has correct format and permissions
- [ ] Base URL is accessible from AWS environment
- [ ] Warehouse name matches Delhivery configuration
- [ ] No trailing spaces or special characters in values

## Next Steps

1. **Deploy** the enhanced version to AWS
2. **Test** using the provided test scripts
3. **Monitor** logs for the flow markers and detailed information
4. **Analyze** captured data to identify the root cause
5. **Fix** the identified issue based on the diagnostic information

## Log Locations

- **Application Logs**: Check your AWS log aggregation service (CloudWatch, etc.)
- **Flow Markers**: Search for `=== FLOW_STEP ===` pattern
- **Detailed Data**: Look for structured JSON logs with request/response details
- **Error Messages**: Check for detailed error information with context

