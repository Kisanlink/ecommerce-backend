-- Add multiple image URL columns to products table
-- This migration adds 4 additional image URL fields to support multiple images per product

-- Add the new image columns
ALTER TABLE products 
ADD COLUMN IF NOT EXISTS image_url_1 TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS image_url_2 TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS image_url_3 TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS image_url_4 TEXT DEFAULT '';

-- Update existing records: if they have an image_url, copy it to image_url_1 as the first additional image
-- This maintains backward compatibility
UPDATE products 
SET image_url_1 = image_url 
WHERE image_url IS NOT NULL 
  AND image_url != '' 
  AND (image_url_1 IS NULL OR image_url_1 = '');

-- Add indexes for better query performance (optional)
CREATE INDEX IF NOT EXISTS idx_products_image_url_1 ON products(image_url_1) WHERE image_url_1 != '';
CREATE INDEX IF NOT EXISTS idx_products_image_url_2 ON products(image_url_2) WHERE image_url_2 != '';
CREATE INDEX IF NOT EXISTS idx_products_image_url_3 ON products(image_url_3) WHERE image_url_3 != '';
CREATE INDEX IF NOT EXISTS idx_products_image_url_4 ON products(image_url_4) WHERE image_url_4 != '';

-- Add a comment to document the schema change
COMMENT ON COLUMN products.image_url IS 'Primary product image URL (backward compatibility)';
COMMENT ON COLUMN products.image_url_1 IS 'Additional product image 1';
COMMENT ON COLUMN products.image_url_2 IS 'Additional product image 2';
COMMENT ON COLUMN products.image_url_3 IS 'Additional product image 3';
COMMENT ON COLUMN products.image_url_4 IS 'Additional product image 4';
