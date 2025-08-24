# Roadmap – Next Steps After Authentication

This roadmap lists the next features to implement end-to-end following the rules: Handlers → Services → Repositories → Models → DbManager, using BaseFilterableRepository and kisanlink-db patterns.

## Priority 1 — Products CRUD (public read, admin write)
- Handlers
  - GET /api/v1/products (list + filters + pagination)
  - GET /api/v1/products/{id}
  - POST /api/v1/products (admin)
  - PUT /api/v1/products/{id} (admin)
  - DELETE /api/v1/products/{id} (admin)
- Services
  - CreateProduct, GetProductByID, ListProducts (with base.Filter), UpdateProduct, SoftDeleteProduct
- Repository
  - ProductRepository methods using BaseFilterableRepository: Find, FindOne, CountWithFilter
- Requests/Responses
  - Use docs/COPILOT_REQUESTS.md patterns for Create/Update/List/Get/Delete
- Wiring
  - Routes in internal/routes/products/product_routes.go
  - Add to routes.RegisterRoutes
- Tests
  - Unit tests for service methods; handler smoke tests

## Priority 2 — Cart (per-user)
- Endpoints
  - GET /api/v1/cart, POST /api/v1/cart/items, PUT /api/v1/cart/items/{id}, DELETE /api/v1/cart/items/{id}
- Service rules
  - Only owner can mutate their cart; validate product existence/stock
- Repository
  - CartRepository queries by user_id (filter + pagination for items)

## Priority 3 — Orders
- Endpoints
  - POST /api/v1/orders (create from cart), GET /api/v1/orders (user), GET /api/v1/orders/{id}, PUT /api/v1/orders/{id}/status (admin)
- Service rules
  - Stock reservation, totals, status transitions (created → paid → shipped → delivered)
- Repository
  - Filter by user_id; status sort; count for pagination

## Priority 4 — Payments (stub → provider)
- Endpoints
  - POST /api/v1/payments, GET /api/v1/payments/{id}
- Service
  - Start with mock provider; return txn id and status

## Priority 5 — Recipes (content)
- Basic CRUD similar to products; public read, admin write

## Cross-cutting tasks
- AuthZ
  - Admin middleware (role/claim) for write endpoints
- Validation
  - Use struct tags and request DTOs from COPILOT_REQUESTS.md
- Pagination & Filtering
  - Use base.FilterBuilder; return base.NewPaginatedResponse
- Errors
  - Use base.NewErrorResponse with specific error types
- Observability
  - Request logging, error logging, timing
- Rate limiting
  - Simple IP/token-based limiter for auth + write routes
- Migrations
  - Ensure AutoMigrate covers new models; keep string IDs (varchar)
- CI/Test
  - Lint + unit tests for services; minimal handler tests

## Suggested sprint order
1) Products CRUD (incl. list filters, pagination)
2) Cart flows
3) Orders (create + list + status)
4) Payments (mock)
5) Recipes
6) Admin middleware, rate limiting, docs polish

## References
- docs/COPILOT_REQUESTS.md for DTO/response patterns
- kisanlink-db/pkg/base for BaseModel, Filter, responses