# AWS Cost Optimization Plan for Ecommerce Backend

## Current Architecture Analysis
- **Tech Stack**: Go with Gin framework, PostgreSQL, Redis, DynamoDB
- **AWS Services**: ECS Fargate, RDS, ElastiCache, NAT Gateway, ALB
- **Estimated Monthly Cost**: $207-255

## Immediate Cost Reductions (Implementation Priority)

### 1. NAT Gateway Optimization (Save $45-50/month)
**Current Setup**: 2 NAT Gateways for high availability
**Optimization**: Reduce to 1 NAT Gateway for non-production environments

```yaml
# Modify cloudformation-infrastructure.yaml
NATGateway:
  Type: AWS::EC2::NatGateway
  Properties:
    AllocationId: !GetAtt NATElasticIP.AllocationId
    SubnetId: !Ref PublicSubnet1

# Update both private route tables to use single NAT
PrivateRouteTable1:
  Properties:
    Routes:
      - DestinationCidrBlock: 0.0.0.0/0
        NatGatewayId: !Ref NATGateway

PrivateRouteTable2:
  Properties:
    Routes:
      - DestinationCidrBlock: 0.0.0.0/0
        NatGatewayId: !Ref NATGateway  # Same NAT for both
```

### 2. Switch to ARM-based Instances (Save $15-20/month)
**RDS Migration**:
```yaml
DBInstance:
  Type: AWS::RDS::DBInstance
  Properties:
    DBInstanceClass: db.t4g.micro  # ARM-based, 20% cheaper
    Engine: postgres
    EngineVersion: "15.4"
```

**ElastiCache Migration**:
```yaml
CacheCluster:
  Type: AWS::ElastiCache::CacheCluster
  Properties:
    CacheNodeType: cache.t4g.micro  # ARM-based
```

### 3. ECS Fargate Spot Instances (Save $10-15/month)
```yaml
Service:
  Type: AWS::ECS::Service
  Properties:
    CapacityProviderStrategy:
      - Base: 0
        Weight: 1
        CapacityProvider: FARGATE_SPOT
    DesiredCount: 2
```

### 4. S3 Lifecycle Policies (Save $5-10/month)
```yaml
# Add to S3 bucket configuration
LifecycleConfiguration:
  Rules:
    - Id: MoveToIA
      Status: Enabled
      Transitions:
        - TransitionInDays: 30
          StorageClass: STANDARD_IA
    - Id: MoveToGlacier
      Status: Enabled
      Transitions:
        - TransitionInDays: 90
          StorageClass: GLACIER
```

## Advanced Optimization Options

### Option A: Serverless Migration (70% Cost Reduction)
**Architecture**: API Gateway → Lambda → RDS Proxy → Aurora Serverless v2

**Implementation Steps**:
1. Convert Gin routes to Lambda functions
2. Use AWS SAM or Serverless Framework
3. Configure RDS Proxy for connection pooling
4. Migrate to Aurora Serverless v2

**Sample Lambda Handler**:
```go
package main

import (
    "context"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

var ginLambda *ginadapter.GinLambda

func init() {
    // Initialize Gin router
    router := setupRouter()
    ginLambda = ginadapter.New(router)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
    lambda.Start(Handler)
}
```

### Option B: Single EC2 Instance (90% Cost Reduction)
**Setup**: t4g.small Spot Instance running all services

**Docker Compose for EC2**:
```yaml
version: '3.8'
services:
  backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_PASSWORD=${DB_PASSWORD}

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - /etc/letsencrypt:/etc/letsencrypt

volumes:
  postgres_data:
  redis_data:
```

**Systemd Service**:
```ini
[Unit]
Description=Ecommerce Backend
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/app
ExecStart=/usr/local/bin/docker-compose up
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Option C: Kubernetes with EKS + Karpenter (For Scale)
**Benefits**: Auto-scaling, spot instance management, better resource utilization

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
spec:
  replicas: 2
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
        karpenter.sh/instance-category: spot
    spec:
      containers:
      - name: backend
        image: your-ecr-repo/backend:latest
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
```

## Database Optimization Strategies

### 1. RDS Reserved Instances (Save 30-40%)
- 1-year term: ~30% discount
- 3-year term: ~50% discount
- Upfront payment: Additional 10% discount

### 2. Aurora Serverless v2 for Variable Load
```sql
-- Minimum ACU: 0.5 (scales to zero)
-- Maximum ACU: 1
-- Cost: $0.12 per ACU-hour
-- Estimated: $10-30/month based on usage
```

### 3. Database Query Optimization
```go
// Use database connection pooling efficiently
db.SetMaxOpenConns(5)
db.SetMaxIdleConns(2)
db.SetConnMaxLifetime(time.Hour)

// Implement query caching
type CachedRepository struct {
    db    *gorm.DB
    cache *redis.Client
}

func (r *CachedRepository) GetProduct(id string) (*Product, error) {
    // Check cache first
    cached, err := r.cache.Get(ctx, "product:"+id).Result()
    if err == nil {
        var product Product
        json.Unmarshal([]byte(cached), &product)
        return &product, nil
    }

    // Fallback to database
    var product Product
    err = r.db.First(&product, "id = ?", id).Error
    if err == nil {
        // Cache for 1 hour
        data, _ := json.Marshal(product)
        r.cache.Set(ctx, "product:"+id, data, time.Hour)
    }
    return &product, err
}
```

## Monitoring and Cost Tracking

### 1. AWS Cost Explorer Tags
```yaml
Tags:
  - Key: Environment
    Value: !Ref EnvironmentName
  - Key: Service
    Value: backend
  - Key: CostCenter
    Value: engineering
```

### 2. CloudWatch Alarms for Cost
```yaml
BillingAlarm:
  Type: AWS::CloudWatch::Alarm
  Properties:
    AlarmName: MonthlyBillingAlarm
    MetricName: EstimatedCharges
    Namespace: AWS/Billing
    Statistic: Maximum
    Period: 86400
    EvaluationPeriods: 1
    Threshold: 100  # Alert if bill exceeds $100
    ComparisonOperator: GreaterThanThreshold
```

### 3. Implement Application-Level Metrics
```go
// Track API usage per endpoint
func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()

        // Log metrics
        duration := time.Since(start)
        status := c.Writer.Status()
        path := c.Request.URL.Path

        // Send to CloudWatch
        cloudwatch.PutMetricData(&cloudwatch.PutMetricDataInput{
            Namespace: aws.String("Backend/API"),
            MetricData: []*cloudwatch.MetricDatum{
                {
                    MetricName: aws.String("RequestDuration"),
                    Value:      aws.Float64(duration.Seconds()),
                    Dimensions: []*cloudwatch.Dimension{
                        {
                            Name:  aws.String("Path"),
                            Value: aws.String(path),
                        },
                        {
                            Name:  aws.String("StatusCode"),
                            Value: aws.String(strconv.Itoa(status)),
                        },
                    },
                },
            },
        })
    }
}
```

## Implementation Roadmap

### Week 1: Quick Wins
- [ ] Reduce NAT Gateways from 2 to 1
- [ ] Switch RDS to t4g.micro
- [ ] Switch ElastiCache to t4g.micro
- [ ] Implement S3 lifecycle policies

### Week 2: Service Optimization
- [ ] Configure Fargate Spot instances
- [ ] Optimize RDS backup retention
- [ ] Review and optimize Security Groups
- [ ] Implement CloudWatch cost alarms

### Week 3: Application-Level Changes
- [ ] Implement database query caching
- [ ] Optimize connection pooling
- [ ] Add request-level caching
- [ ] Reduce unnecessary S3 operations

### Week 4: Evaluation & Next Steps
- [ ] Analyze cost reduction achieved
- [ ] Plan serverless migration (if applicable)
- [ ] Consider Reserved Instances
- [ ] Document optimizations

## Expected Savings Summary

| Optimization | Monthly Savings | Implementation Effort |
|--------------|-----------------|----------------------|
| Reduce NAT Gateways | $45-50 | Low (1 hour) |
| ARM-based instances | $15-20 | Low (2 hours) |
| Fargate Spot | $10-15 | Low (1 hour) |
| S3 Lifecycle | $5-10 | Low (30 mins) |
| Query Optimization | $5-10 | Medium (4 hours) |
| **Total Quick Wins** | **$80-105** | **~1 day** |
| Serverless Migration | $150-180 | High (2 weeks) |
| Single EC2 Setup | $180-200 | Medium (3 days) |

## Recommended Approach

For your use case, I recommend:

1. **Immediate**: Implement all quick wins (save $80-105/month)
2. **Short-term** (1 month): Evaluate traffic patterns
3. **Long-term** (3 months):
   - If traffic is variable/low: Migrate to serverless
   - If traffic is consistent: Consider single EC2 or Reserved Instances

## Additional Resources

- [AWS Well-Architected Cost Optimization](https://docs.aws.amazon.com/wellarchitected/latest/cost-optimization-pillar/welcome.html)
- [AWS Compute Optimizer](https://aws.amazon.com/compute-optimizer/)
- [AWS Trusted Advisor](https://aws.amazon.com/premiumsupport/technology/trusted-advisor/)
- [Cost Explorer](https://aws.amazon.com/aws-cost-management/aws-cost-explorer/)

## Questions to Consider

1. What's your average daily traffic volume?
2. Do you have predictable traffic patterns?
3. Is this production or development environment?
4. What's your uptime requirement (99.9% vs 99.99%)?
5. Do you need multi-region deployment?

Based on these answers, we can further refine the optimization strategy.