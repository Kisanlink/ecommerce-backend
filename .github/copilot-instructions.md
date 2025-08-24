[byterover-mcp]

# important 
always use byterover-retrieve-knowledge tool to get the related context before any tasks 
always use byterover-store-knowledge to store all the critical informations after sucessful tasks

## ✅ Development Rules

- Handlers → Services → Repositories → Models → DbManager.  
- Models are **consistently reused everywhere**.  
- Repositories **must extend BaseFilterableRepository**.  
- DbManager is the **only DB connection entrypoint**.  
- Middleware handles **cross-cutting concerns** only.  
- No business logic inside Handlers or Repositories.  

---