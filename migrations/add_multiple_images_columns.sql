-- Add multiple image URL columns to products table
ALTER TABLE products 
ADD COLUMN image_url_1 TEXT DEFAULT '',
ADD COLUMN image_url_2 TEXT DEFAULT '',
ADD COLUMN image_url_3 TEXT DEFAULT '',
ADD COLUMN image_url_4 TEXT DEFAULT '';

-- Update existing records: if they have an image_url, copy it to image_url_1
UPDATE products 
SET image_url_1 = image_url 
WHERE image_url IS NOT NULL AND image_url != '';

-- Add indexes for better query performance (optional)
CREATE INDEX IF NOT EXISTS idx_products_image_url_1 ON products(image_url_1);
CREATE INDEX IF NOT EXISTS idx_products_image_url_2 ON products(image_url_2);
CREATE INDEX IF NOT EXISTS idx_products_image_url_3 ON products(image_url_3);
CREATE INDEX IF NOT EXISTS idx_products_image_url_4 ON products(image_url_4);
