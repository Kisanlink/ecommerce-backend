[byterover-mcp]

# important 
always use byterover-retrieve-knowledge tool to get the related context before any tasks 
always use byterover-store-knowledge to store all the critical informations after sucessful tasks

USE THIS TO RUN cd ecommerce-backend; go run ./cmd/server

always use this command to check the working of api 
'''
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/url" -Method GET -ContentType "application/json"
'''
## ✅ Development Rules

- Handlers → Services → Repositories → Models → DbManager.  
- Models are **consistently reused everywhere**.  
- Repositories **must extend BaseFilterableRepository**.  
- DbManager is the **only DB connection entrypoint**.  
- Middleware handles **cross-cutting concerns** only.  
- No business logic inside Handlers or Repositories.  

---