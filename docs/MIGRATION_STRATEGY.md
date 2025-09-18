# Database Migration Strategy for Orders Normalization

## Overview
This document outlines the migration strategy for transitioning from denormalized to normalized orders table structure.

## Migration Phases

### Phase 1: Schema Migration (Current)
- ✅ Add new tables: `order_customers`, `order_addresses`
- ✅ Add new nullable columns to `orders`: `customer_id`, `shipping_address_id`
- ✅ Keep legacy columns for backward compatibility

### Phase 2: Data Migration (Future)
After schema migration is complete, you can:

1. **Migrate existing orders data:**
   ```sql
   -- For each existing order, create customer and address records
   -- Then update the order with the new foreign key references
   ```

2. **Make foreign keys NOT NULL:**
   ```sql
   ALTER TABLE orders ALTER COLUMN customer_id SET NOT NULL;
   ALTER TABLE orders ALTER COLUMN shipping_address_id SET NOT NULL;
   ```

3. **Add foreign key constraints:**
   ```sql
   ALTER TABLE orders ADD CONSTRAINT fk_orders_customer_id 
   FOREIGN KEY (customer_id) REFERENCES order_customers(id);
   
   ALTER TABLE orders ADD CONSTRAINT fk_orders_shipping_address_id 
   FOREIGN KEY (shipping_address_id) REFERENCES order_addresses(id);
   ```

### Phase 3: Cleanup (Future)
- Remove legacy columns after confirming all data is migrated
- Update application code to remove legacy field handling

## Current Status
- ✅ Schema migration completed
- ✅ New tables created
- ✅ New columns added (nullable)
- ⏳ Data migration pending
- ⏳ Constraints pending

## Backward Compatibility
The current implementation maintains full backward compatibility:
- Existing orders continue to work with legacy fields
- New orders can use normalized structure
- API endpoints handle both formats seamlessly
