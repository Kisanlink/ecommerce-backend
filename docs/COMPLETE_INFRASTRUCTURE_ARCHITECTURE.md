# Complete Infrastructure & Architecture Documentation
## Ecommerce Backend System - Full Technical Specification

> **Last Updated**: November 2024
> **Version**: 1.0
> **Environment**: Production, Staging, Development
> **AWS Region**: ap-south-1 (Mumbai)

---

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Technology Stack](#technology-stack)
3. [AWS Infrastructure Architecture](#aws-infrastructure-architecture)
4. [Application Architecture](#application-architecture)
5. [Database Architecture](#database-architecture)
6. [Security Architecture](#security-architecture)
7. [CI/CD Pipeline](#cicd-pipeline)
8. [External Integrations](#external-integrations)
9. [API Documentation](#api-documentation)
10. [Deployment Guide](#deployment-guide)
11. [Monitoring & Logging](#monitoring--logging)
12. [Cost Analysis](#cost-analysis)
13. [Disaster Recovery](#disaster-recovery)
14. [Performance Optimization](#performance-optimization)
15. [Development Guidelines](#development-guidelines)

---

## Executive Summary

### System Overview
This is a production-grade, cloud-native ecommerce platform built with Go and deployed on AWS. The system handles complete B2C ecommerce operations including product catalog, inventory management, order processing, payment gateway integration, and delivery tracking.

### Key Capabilities
- **Multi-vendor marketplace support**
- **Real-time inventory tracking**
- **Integrated payment processing (Razorpay)**
- **Logistics management (Delhivery)**
- **WhatsApp Business API for notifications**
- **Firebase authentication**
- **Admin dashboard for management**
- **Review and rating system**
- **Coupon and discount management**

### Architecture Highlights
- **Microservices-ready architecture**
- **Container-based deployment (Docker/ECS)**
- **Multi-database strategy (PostgreSQL, DynamoDB)**
- **Infrastructure as Code (CloudFormation)**
- **Automated CI/CD pipeline**
- **High availability with multi-AZ deployment**

---

## Technology Stack

### Core Technologies

#### Backend Framework
```yaml
Language: Go 1.24.6
Framework: Gin Web Framework v1.10.1
Architecture: Layered (Handler → Service → Repository → Database)
API Style: RESTful with JSON
Documentation: Swagger/OpenAPI 3.0
```

#### Databases
| Database | Version | Purpose | Driver/ORM |
|----------|---------|---------|------------|
| PostgreSQL | 15.4 | Primary data store | GORM v1.25.12 |
| DynamoDB | - | Pending recipe reviews, NoSQL data | AWS SDK v2 |

**Note**: Redis support exists in codebase but is optional. The application gracefully falls back to PostgreSQL-only operations if Redis is not available. Currently not deployed in production.

#### AWS Services
```yaml
Compute:
  - ECS Fargate (512 CPU, 1024 MB RAM)
  - Lambda (future serverless functions)

Storage:
  - S3 (amrti bucket, AES-256 encryption)
  - EBS (gp3 volumes for RDS)

Database:
  - RDS PostgreSQL (db.t3.micro)
  - DynamoDB (on-demand billing)
  - ElastiCache Redis (optional - code support exists but not currently deployed)

Networking:
  - VPC (10.0.0.0/16)
  - Application Load Balancer
  - NAT Gateway (2x for HA)
  - Route 53 (DNS)

Security:
  - IAM Roles and Policies
  - Security Groups
  - Secrets Manager
  - Systems Manager Parameter Store

DevOps:
  - CodePipeline
  - CodeBuild
  - ECR (Container Registry)
  - CloudWatch (Logs & Metrics)
```

---

## AWS Infrastructure Architecture

### Network Architecture

```
Internet
    ↓
Internet Gateway
    ↓
┌─────────────────────────────────────────────────┐
│                   VPC: 10.0.0.0/16              │
│                                                  │
│  ┌──────────────────┐  ┌──────────────────┐   │
│  │  Public Subnet 1  │  │  Public Subnet 2  │   │
│  │   10.0.1.0/24    │  │   10.0.2.0/24     │   │
│  │                  │  │                   │   │
│  │   ┌─────────┐   │  │    ┌─────────┐   │   │
│  │   │   ALB   │   │  │    │   NAT   │   │   │
│  │   └─────────┘   │  │    │ Gateway │   │   │
│  └──────────────────┘  └────┴─────────┴────┘   │
│           ↓                        ↓            │
│  ┌──────────────────┐  ┌──────────────────┐   │
│  │  Private Subnet 1 │  │  Private Subnet 2 │   │
│  │   10.0.3.0/24    │  │   10.0.4.0/24     │   │
│  │                  │  │                   │   │
│  │  ┌────────────┐  │  │  ┌────────────┐  │   │
│  │  │ ECS Tasks  │  │  │  │    RDS     │  │   │
│  │  └────────────┘  │  │  │ PostgreSQL │  │   │
│  │                  │  │  └────────────┘  │   │
│  │                  │  │  ┌────────────┐  │   │
│  │                  │  │  │  DynamoDB  │  │   │
│  │                  │  │  │   Tables   │  │   │
│  │                  │  │  └────────────┘  │   │
│  └──────────────────┘  └──────────────────┘   │
└─────────────────────────────────────────────────┘
```

### Infrastructure Components

#### VPC Configuration
```yaml
CIDR: 10.0.0.0/16
Availability Zones: 2 (ap-south-1a, ap-south-1b)
Public Subnets: 2 (10.0.1.0/24, 10.0.2.0/24)
Private Subnets: 2 (10.0.3.0/24, 10.0.4.0/24)
NAT Gateways: 2 (one per AZ for HA)
Internet Gateway: 1
```

#### Security Groups
| Security Group | Purpose | Inbound Rules | Outbound Rules |
|---------------|---------|---------------|----------------|
| ALB-SG | Load Balancer | 80, 443 from 0.0.0.0/0 | All traffic |
| ECS-SG | Application | 8080 from ALB-SG | All traffic |
| RDS-SG | Database | 5432 from ECS-SG | None |
| Cache-SG | Redis (optional) | 6379 from ECS-SG | None |

---

## Application Architecture

### Directory Structure
```
ecommerce-backend/
├── cmd/
│   ├── server/          # Main application entry point
│   │   └── main.go      # Server initialization
│   ├── seed-images/     # Image seeding utility
│   └── migration/       # Database migration tools
│
├── internal/
│   ├── config/          # Configuration management
│   │   ├── config.go
│   │   └── env.go
│   │
│   ├── database/        # Database connections
│   │   ├── postgres.go
│   │   ├── redis.go (optional - not currently used)
│   │   └── dynamodb.go
│   │
│   ├── entities/        # Data models/DTOs
│   │   ├── user.go
│   │   ├── product.go
│   │   ├── order.go
│   │   ├── cart.go
│   │   ├── inventory.go
│   │   └── review.go
│   │
│   ├── handlers/        # HTTP handlers (Controllers)
│   │   ├── admin/       # Admin endpoints
│   │   ├── auth/        # Authentication
│   │   ├── cart/        # Cart operations
│   │   ├── order/       # Order management
│   │   ├── product/     # Product catalog
│   │   ├── payment/     # Payment processing
│   │   └── user/        # User management
│   │
│   ├── middleware/      # HTTP middleware
│   │   ├── auth.go      # JWT validation
│   │   ├── cors.go      # CORS handling
│   │   ├── logger.go    # Request logging
│   │   └── ratelimit.go # Rate limiting
│   │
│   ├── repositories/    # Data access layer
│   │   ├── base.go
│   │   ├── user_repository.go
│   │   ├── product_repository.go
│   │   ├── order_repository.go
│   │   └── cart_repository.go
│   │
│   ├── services/        # Business logic layer
│   │   ├── auth_service.go
│   │   ├── cart_service.go
│   │   ├── order_service.go
│   │   ├── payment_service.go
│   │   ├── delivery_service.go
│   │   ├── notification_service.go
│   │   ├── s3_service.go
│   │   └── cache_service.go
│   │
│   ├── routes/          # Route definitions
│   │   ├── routes.go
│   │   ├── admin_routes.go
│   │   └── api_routes.go
│   │
│   └── utils/           # Utility functions
│       ├── validator.go
│       ├── response.go
│       └── errors.go
│
├── pkg/                 # Public packages
│   └── kisanlink-db/    # Custom database package
│
├── docs/                # Documentation
├── scripts/             # Deployment scripts
├── cloudformation/      # IaC templates
├── .github/            # GitHub workflows
├── Dockerfile          # Container definition
├── docker-compose.yml  # Local development
├── go.mod              # Go modules
└── .env.example        # Environment template
```

### Request Flow Architecture

```
Client Request
      ↓
Load Balancer (ALB)
      ↓
ECS Fargate Task
      ↓
Gin Router → Middleware Chain
      ↓
Route Handler
      ↓
Service Layer (Business Logic)
      ↓
Repository Layer (Data Access)
      ↓
Database (PostgreSQL/DynamoDB)
      ↓
Response → Client
```

### Layered Architecture Pattern

#### 1. Handler Layer (Controllers)
```go
// Example: Product Handler
type ProductHandler struct {
    service services.ProductService
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
    productID := c.Param("id")
    product, err := h.service.GetProductByID(productID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, product)
}
```

#### 2. Service Layer (Business Logic)
```go
type ProductService struct {
    repo repositories.ProductRepository
    cache services.CacheService
    s3 services.S3Service
}

func (s *ProductService) GetProductByID(id string) (*entities.Product, error) {
    // Check cache first
    if cached, err := s.cache.Get("product:" + id); err == nil {
        return cached.(*entities.Product), nil
    }

    // Fetch from database
    product, err := s.repo.FindByID(id)
    if err != nil {
        return nil, err
    }

    // Cache for future requests
    s.cache.Set("product:"+id, product, 1*time.Hour)
    return product, nil
}
```

#### 3. Repository Layer (Data Access)
```go
type ProductRepository struct {
    db *gorm.DB
}

func (r *ProductRepository) FindByID(id string) (*entities.Product, error) {
    var product entities.Product
    err := r.db.Preload("Images").First(&product, "id = ?", id).Error
    return &product, err
}
```

---

## Database Architecture

### PostgreSQL Schema

#### Core Tables
```sql
-- Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    firebase_uid VARCHAR(255) UNIQUE,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    role VARCHAR(50) DEFAULT 'customer',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Products Table
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    subcategory VARCHAR(100),
    price DECIMAL(10,2),
    mrp DECIMAL(10,2),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Orders Table
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    order_number VARCHAR(50) UNIQUE,
    status VARCHAR(50) DEFAULT 'pending',
    total_amount DECIMAL(10,2),
    payment_status VARCHAR(50),
    payment_method VARCHAR(50),
    delivery_address JSONB,
    tracking_number VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Inventory Table
CREATE TABLE inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID REFERENCES products(id),
    quantity INTEGER DEFAULT 0,
    reserved_quantity INTEGER DEFAULT 0,
    warehouse_location VARCHAR(100),
    last_restock_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Cart Items Table
CREATE TABLE cart_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    product_id UUID REFERENCES products(id),
    quantity INTEGER DEFAULT 1,
    added_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, product_id)
);
```

### Caching Strategy

**Note**: Redis caching support exists in the codebase but is optional. The application operates without Redis, using PostgreSQL as the primary data store. Cart operations gracefully fall back to database-only when Redis is unavailable.

#### Redis Cache Keys (Optional - When Redis is Deployed)
```yaml
Cart Data:
  Pattern: "user_cart:{userId}"
  TTL: 24 hours

Cart Locks:
  Pattern: "cart_lock:{userId}"
  TTL: 30 seconds
```

### DynamoDB Tables

#### Pending Recipe Reviews Table
```yaml
TableName: pending_recipe_reviews
PartitionKey: recipe_id (String)
SortKey: review_id (String)
Attributes:
  - user_id (String)
  - status (String)
  - admin_notes (String)
  - created_at (String)

GlobalSecondaryIndexes:
  - StatusIndex:
      PartitionKey: status
      SortKey: created_at
```

---

## Security Architecture

### Authentication & Authorization

#### Firebase Authentication
```yaml
Provider: Firebase Auth
Token Type: JWT Bearer
Validation: Middleware on each request
Admin Check: Custom claims in Firebase token
```

#### JWT Token Flow
```
1. Client → Firebase Auth → Get ID Token
2. Client → Backend API (Bearer Token)
3. Backend → Validate with Firebase Admin SDK
4. Backend → Extract user claims
5. Backend → Authorize based on role
```

### API Security Measures

#### Middleware Stack
```go
router.Use(
    middleware.CORS(),          // CORS headers
    middleware.RateLimiter(),    // Rate limiting
    middleware.RequestLogger(),  // Logging
    middleware.ErrorHandler(),   // Error handling
    middleware.AuthValidator(),  // JWT validation
)
```

#### Rate Limiting Configuration
```yaml
Public Endpoints: 100 requests/minute
Authenticated: 500 requests/minute
Admin: Unlimited
Payment Webhooks: 50 requests/minute
```

### Secrets Management

#### AWS Systems Manager Parameter Store
```yaml
Database Credentials:
  - /ecommerce/db/postgres/password
  - /ecommerce/db/redis/password (optional - if Redis is deployed)

API Keys:
  - /ecommerce/razorpay/key_id
  - /ecommerce/razorpay/key_secret
  - /ecommerce/delhivery/api_token
  - /ecommerce/whatsapp/access_token
  - /ecommerce/firebase/service_account

AWS Credentials:
  - /ecommerce/aws/access_key_id
  - /ecommerce/aws/secret_access_key
```

### Encryption

#### Data at Rest
- **RDS**: AES-256 encryption
- **S3**: AES-256 server-side encryption
- **EBS Volumes**: Encrypted by default
- **ElastiCache**: At-rest encryption enabled (if Redis is deployed)

#### Data in Transit
- **ALB → ECS**: TLS 1.2+
- **ECS → RDS**: SSL/TLS
- **ECS → DynamoDB**: HTTPS/TLS
- **ECS → Redis**: TLS enabled (if Redis is deployed)
- **API Responses**: HTTPS only

---

## CI/CD Pipeline

### Pipeline Architecture

```
GitHub Repository
      ↓
AWS CodePipeline (Trigger on push)
      ↓
Source Stage (GitHub webhook)
      ↓
Build Stage (CodeBuild)
      ├── Run tests
      ├── Build Docker image
      ├── Push to ECR
      └── Generate artifacts
      ↓
Deploy Stage (ECS)
      ├── Update task definition
      ├── Deploy to ECS service
      └── Health check validation
      ↓
Post-Deployment
      ├── Run smoke tests
      └── Send notifications
```

### BuildSpec Configuration

```yaml
version: 0.2
phases:
  pre_build:
    commands:
      - echo Logging in to Amazon ECR...
      - aws ecr get-login-password --region $AWS_DEFAULT_REGION | docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com
      - REPOSITORY_URI=$AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/$IMAGE_REPO_NAME
      - COMMIT_HASH=$(echo $CODEBUILD_RESOLVED_SOURCE_VERSION | cut -c 1-7)
      - IMAGE_TAG=${COMMIT_HASH:=latest}

  build:
    commands:
      - echo Build started on `date`
      - echo Running tests...
      - go test ./...
      - echo Building the Docker image...
      - docker build -t $REPOSITORY_URI:latest .
      - docker tag $REPOSITORY_URI:latest $REPOSITORY_URI:$IMAGE_TAG

  post_build:
    commands:
      - echo Build completed on `date`
      - echo Pushing the Docker images...
      - docker push $REPOSITORY_URI:latest
      - docker push $REPOSITORY_URI:$IMAGE_TAG
      - printf '[{"name":"%s","imageUri":"%s"}]' $CONTAINER_NAME $REPOSITORY_URI:$IMAGE_TAG > imagedefinitions.json

artifacts:
  files:
    - imagedefinitions.json
```

### Deployment Strategy

#### Blue-Green Deployment
```yaml
Type: Blue-Green
Health Check Grace Period: 300 seconds
Deployment Circuit Breaker: Enabled
Rollback on Failure: Automatic
Maximum Percent: 200
Minimum Healthy Percent: 100
```

---

## External Integrations

### Payment Gateway - Razorpay

#### Integration Points
```yaml
Endpoints:
  - Create Order: POST /api/v1/payment/create-order
  - Verify Payment: POST /api/v1/payment/verify
  - Webhook: POST /api/v1/webhooks/razorpay
  - Refund: POST /api/v1/payment/refund

Configuration:
  - Key ID: From Parameter Store
  - Key Secret: From Parameter Store
  - Webhook Secret: Configured in Razorpay Dashboard
  - Currency: INR
  - Auto Capture: true
```

#### Payment Flow
```
1. Client requests order creation
2. Backend creates Razorpay order
3. Client completes payment
4. Razorpay sends webhook
5. Backend verifies signature
6. Update order status
7. Trigger fulfillment
```

### Logistics - Delhivery

#### API Integration
```yaml
Endpoints:
  - Create Shipment: POST /api/v1/delivery/create-shipment
  - Track Shipment: GET /api/v1/delivery/track/{awb}
  - Cancel Shipment: POST /api/v1/delivery/cancel
  - Generate Label: GET /api/v1/delivery/label/{awb}

Features:
  - Real-time tracking
  - Automated AWB generation
  - Pickup scheduling
  - COD management
```

### WhatsApp Business API

#### Notification Types
```yaml
Order Confirmation:
  Template: order_confirmation_v1
  Variables: [order_id, amount, delivery_date]

Shipment Updates:
  Template: shipment_update_v1
  Variables: [awb, status, tracking_link]

Payment Confirmation:
  Template: payment_success_v1
  Variables: [amount, payment_id, order_id]
```

---

## API Documentation

### Base URLs
```yaml
Production: https://api.yourdomain.com
Staging: https://staging-api.yourdomain.com
Local: http://localhost:8080
```

### API Versioning
```yaml
Current Version: v1
Path Pattern: /api/v1/{resource}
Header: X-API-Version
```

### Authentication Endpoints

#### User Registration
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123",
  "phone": "+919876543210",
  "name": "John Doe"
}

Response: 201 Created
{
  "success": true,
  "data": {
    "user_id": "uuid",
    "email": "user@example.com",
    "token": "jwt_token"
  }
}
```

#### User Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123"
}

Response: 200 OK
{
  "success": true,
  "data": {
    "user": {...},
    "token": "jwt_token",
    "refresh_token": "refresh_token"
  }
}
```

### Product Endpoints

#### Get Products
```http
GET /api/v1/products?page=1&limit=20&category=electronics
Authorization: Bearer {token}

Response: 200 OK
{
  "success": true,
  "data": {
    "products": [...],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 150,
      "pages": 8
    }
  }
}
```

#### Get Product Details
```http
GET /api/v1/products/{product_id}
Authorization: Bearer {token}

Response: 200 OK
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Product Name",
    "price": 999.99,
    "images": [...],
    "inventory": {
      "available": 50,
      "reserved": 5
    }
  }
}
```

### Cart Operations

#### Add to Cart
```http
POST /api/v1/cart/add
Authorization: Bearer {token}
Content-Type: application/json

{
  "product_id": "uuid",
  "quantity": 2
}

Response: 200 OK
{
  "success": true,
  "data": {
    "cart": {
      "items": [...],
      "total": 1999.98
    }
  }
}
```

### Order Management

#### Create Order
```http
POST /api/v1/orders/create
Authorization: Bearer {token}
Content-Type: application/json

{
  "items": [...],
  "address_id": "uuid",
  "payment_method": "razorpay",
  "coupon_code": "SAVE10"
}

Response: 201 Created
{
  "success": true,
  "data": {
    "order_id": "uuid",
    "order_number": "ORD-2024-0001",
    "payment_url": "https://razorpay.com/...",
    "total": 1799.98
  }
}
```

### Admin Endpoints

#### Dashboard Statistics
```http
GET /api/v1/admin/dashboard
Authorization: Bearer {admin_token}

Response: 200 OK
{
  "success": true,
  "data": {
    "total_orders": 1250,
    "revenue": 125000.50,
    "active_users": 350,
    "pending_orders": 25,
    "low_stock_products": 8
  }
}
```

---

## Deployment Guide

### Prerequisites
```bash
# Required tools
- AWS CLI configured
- Docker installed
- Go 1.24.6+
- GitHub account with repository access
- AWS account with appropriate permissions
```

### Local Development Setup

#### 1. Clone Repository
```bash
git clone https://github.com/Kisanlink/ecommerce-backend.git
cd ecommerce-backend
```

#### 2. Environment Configuration
```bash
cp .env.example .env
# Edit .env with your configuration
```

#### 3. Start Local Services
```bash
docker-compose up -d postgres dynamodb
# Note: Redis is optional - application works without it
```

#### 4. Run Application
```bash
go mod download
go run cmd/server/main.go
```

### Production Deployment

#### 1. Infrastructure Deployment
```bash
# Deploy CloudFormation stack
aws cloudformation create-stack \
  --stack-name ecommerce-infrastructure \
  --template-body file://cloudformation-infrastructure.yaml \
  --parameters ParameterKey=EnvironmentName,ParameterValue=production \
  --capabilities CAPABILITY_IAM

# Wait for stack creation
aws cloudformation wait stack-create-complete \
  --stack-name ecommerce-infrastructure
```

#### 2. Pipeline Deployment
```bash
# Deploy CI/CD pipeline
aws cloudformation create-stack \
  --stack-name ecommerce-pipeline \
  --template-body file://cloudformation-pipeline.yaml \
  --parameters ParameterKey=GitHubToken,ParameterValue=your-token \
  --capabilities CAPABILITY_IAM
```

#### 3. Initial Deployment
```bash
# Push to main branch triggers automatic deployment
git push origin main
```

### Database Migration
```bash
# Run migrations
go run cmd/migration/main.go up

# Seed initial data
go run cmd/seed/main.go
```

---

## Monitoring & Logging

### CloudWatch Metrics

#### Application Metrics
```yaml
Custom Metrics:
  - API Request Count
  - API Response Time
  - Error Rate
  - Business Metrics (Orders, Revenue)

Default Metrics:
  - ECS CPU/Memory Utilization
  - ALB Request Count
  - RDS Connections
  - DynamoDB Read/Write Capacity
  - ElastiCache Hit Rate (if Redis is deployed)
```

#### Log Groups
```yaml
Application Logs: /ecs/ecommerce-backend
Build Logs: /aws/codebuild/ecommerce-build
API Gateway Logs: /aws/apigateway/ecommerce
RDS Logs: /aws/rds/instance/ecommerce-postgres
```

### Alarms Configuration

#### Critical Alarms
```yaml
High Error Rate:
  Threshold: > 1% of requests
  Period: 5 minutes
  Action: SNS notification

Database Connection Pool:
  Threshold: > 80% utilized
  Period: 5 minutes
  Action: Auto-scale RDS

ECS Task Failure:
  Threshold: > 2 failures
  Period: 10 minutes
  Action: Page on-call engineer
```

### Application Performance Monitoring

#### Key Metrics
```yaml
Response Times:
  - p50: < 200ms
  - p95: < 500ms
  - p99: < 1000ms

Availability:
  - Target: 99.9%
  - Measured: Last 30 days

Error Budget:
  - Monthly: 43.2 minutes
  - Remaining: Real-time tracking
```

---

## Cost Analysis

### Current Monthly Costs
| Service | Configuration | Estimated Cost |
|---------|--------------|----------------|
| ECS Fargate | 2 tasks, 512 CPU, 1GB RAM | $36-40 |
| RDS PostgreSQL | db.t3.micro, 20GB | $25-30 |
| NAT Gateways | 2 instances | $90-100 |
| ALB | 1 instance | $22-25 |
| S3 + Transfer | Storage + bandwidth | $10-20 |
| DynamoDB | On-demand | $5-15 |
| ElastiCache Redis | cache.t3.micro (optional) | $17-20 |
| **Total (Current)** | | **$188-230** |
| **Total (with Redis)** | | **$205-250** |

### Cost Optimization Opportunities
1. **Use single NAT Gateway**: Save $45/month
2. **Switch to ARM instances**: Save $15-20/month
3. **Use Fargate Spot**: Save $10-15/month
4. **Implement S3 lifecycle policies**: Save $5-10/month

### Reserved Capacity Options
- **RDS Reserved Instances**: 30-50% savings
- **ElastiCache Reserved Nodes**: 30-55% savings
- **Compute Savings Plans**: 20-30% savings

---

## Disaster Recovery

### Backup Strategy

#### RDS Backups
```yaml
Automated Backups:
  - Retention: 7 days
  - Window: 03:00-04:00 UTC
  - Point-in-time recovery: Yes

Manual Snapshots:
  - Frequency: Weekly
  - Retention: 30 days
  - Cross-region copy: Yes
```

#### Application Data
```yaml
S3 Versioning: Enabled
S3 Cross-Region Replication: Configured
DynamoDB Backups: Daily automated
Redis Snapshots: Every 6 hours (if Redis is deployed)
```

### Recovery Procedures

#### RTO/RPO Targets
```yaml
Recovery Time Objective (RTO): 4 hours
Recovery Point Objective (RPO): 1 hour
```

#### Disaster Recovery Plan
1. **Detection**: CloudWatch alarms trigger
2. **Assessment**: On-call engineer evaluates
3. **Communication**: Notify stakeholders
4. **Recovery**: Execute recovery runbook
5. **Validation**: Run smoke tests
6. **Documentation**: Post-incident report

---

## Performance Optimization

### Database Optimizations

#### Query Optimization
```sql
-- Add indexes for common queries
CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_orders_user_status ON orders(user_id, status);
CREATE INDEX idx_cart_user ON cart_items(user_id);

-- Composite indexes for complex queries
CREATE INDEX idx_products_search ON products(name, category, status);
```

#### Connection Pooling
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### Caching Strategy

#### Cache Layers
1. **CDN Level**: CloudFront for static assets (future)
2. **Application Level**: PostgreSQL-based caching (Redis optional)
3. **Database Level**: Query result caching
4. **Code Level**: In-memory caching for hot data

### API Optimization

#### Response Compression
```go
router.Use(gzip.Gzip(gzip.DefaultCompression))
```

#### Pagination
```go
type PaginationParams struct {
    Page  int `form:"page" binding:"min=1"`
    Limit int `form:"limit" binding:"min=1,max=100"`
}
```

---

## Development Guidelines

### Code Standards

#### Go Best Practices
```go
// Use meaningful variable names
var userAuthToken string // Good
var uat string           // Bad

// Handle errors explicitly
if err != nil {
    log.Error("Failed to process order", err)
    return fmt.Errorf("process order: %w", err)
}

// Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### Testing Strategy

#### Test Coverage Requirements
```yaml
Unit Tests: > 80% coverage
Integration Tests: Critical paths
End-to-End Tests: User journeys
Performance Tests: Load testing
```

#### Test Examples
```go
func TestCreateOrder(t *testing.T) {
    // Arrange
    service := NewOrderService(mockRepo, mockPayment)

    // Act
    order, err := service.CreateOrder(testOrder)

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, order.ID)
    assert.Equal(t, "pending", order.Status)
}
```

### Git Workflow

#### Branch Strategy
```yaml
main: Production-ready code
develop: Integration branch
feature/*: New features
bugfix/*: Bug fixes
hotfix/*: Emergency fixes
```

#### Commit Convention
```bash
feat: Add new feature
fix: Bug fix
docs: Documentation
style: Formatting
refactor: Code restructuring
test: Add tests
chore: Maintenance
```

---

## Troubleshooting Guide

### Common Issues

#### Database Connection Issues
```bash
# Check RDS status
aws rds describe-db-instances --db-instance-identifier ecommerce-postgres

# Test connection
psql -h <rds-endpoint> -U dbadmin -d ecommerce

# Check security groups
aws ec2 describe-security-groups --group-ids sg-xxxxx
```

#### ECS Task Failures
```bash
# Check task status
aws ecs describe-tasks --cluster ecommerce-cluster --tasks <task-arn>

# View logs
aws logs get-log-events --log-group-name /ecs/ecommerce-backend --log-stream-name <stream>

# Check task definition
aws ecs describe-task-definition --task-definition ecommerce-backend
```

#### High Memory Usage
```bash
# Analyze Go memory
go tool pprof http://localhost:8080/debug/pprof/heap

# Check RDS connections
aws rds describe-db-instances --db-instance-identifier ecommerce-postgres

# Check ECS task memory
aws ecs describe-tasks --cluster ecommerce-cluster --tasks <task-arn>
```

---

## Appendix

### Environment Variables

```bash
# Database
DB_POSTGRES_HOST=localhost
DB_POSTGRES_PORT=5432
DB_POSTGRES_USER=dbadmin
DB_POSTGRES_PASSWORD=secret
DB_POSTGRES_DBNAME=ecommerce

# Redis (Optional - not currently used in production)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# AWS
AWS_REGION=ap-south-1
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
DB_S3_BUCKET=amrti
DB_DYNAMO_TABLE=product_reviews

# External Services
RAZORPAY_KEY_ID=
RAZORPAY_KEY_SECRET=
DELHIVERY_API_TOKEN=
WHATSAPP_ACCESS_TOKEN=
FIREBASE_SERVICE_ACCOUNT_PATH=

# Application
PORT=8080
ENV=development
LOG_LEVEL=debug
JWT_SECRET=
```

### Useful Commands

```bash
# Build Docker image
docker build -t ecommerce-backend .

# Run with docker-compose
docker-compose up -d

# Run tests
go test ./... -cover

# Generate swagger docs
swag init -g cmd/server/main.go

# Database migration
migrate -path migrations -database postgres://... up

# Check code quality
golangci-lint run

# Format code
go fmt ./...
```

### Support & Contact

- **Documentation**: [Internal Wiki](https://wiki.company.com/ecommerce)
- **Monitoring Dashboard**: [CloudWatch](https://console.aws.amazon.com/cloudwatch)
- **On-Call**: PagerDuty rotation
- **Slack Channel**: #ecommerce-backend
- **Email**: devops@company.com

---

## Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | Nov 2024 | Initial documentation | DevOps Team |
| 0.9 | Oct 2024 | Beta release | Development Team |
| 0.8 | Sep 2024 | Alpha testing | QA Team |

---

*This document is maintained by the DevOps team and should be updated with any infrastructure changes.*