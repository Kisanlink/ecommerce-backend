-- Migration script to add Delhivery tracking fields to orders table
-- Run this in your PostgreSQL database

-- Add Delhivery tracking columns
ALTER TABLE orders ADD COLUMN IF NOT EXISTS delhivery_waybill VARCHAR(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS delhivery_status VARCHAR(50) DEFAULT '';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS tracking_url TEXT DEFAULT '';

-- Add indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_orders_delhivery_waybill ON orders(delhivery_waybill);
CREATE INDEX IF NOT EXISTS idx_orders_delhivery_status ON orders(delhivery_status);

-- Add comments for documentation
COMMENT ON COLUMN orders.delhivery_waybill IS 'AWB number from Delhivery API';
COMMENT ON COLUMN orders.delhivery_status IS 'Current shipment status from Delhivery';
COMMENT ON COLUMN orders.tracking_url IS 'Direct tracking URL for the shipment';

-- Verify columns were added
SELECT column_name, data_type, is_nullable, column_default 
FROM information_schema.columns 
WHERE table_name = 'orders' 
AND column_name IN ('delhivery_waybill', 'delhivery_status', 'tracking_url');

-- Show sample of orders table structure
\d orders;
