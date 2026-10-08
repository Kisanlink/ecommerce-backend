# Product Images Seeding

This directory contains the script to seed product images into the database.

## Overview

The `main.go` script seeds product images for all 7 products in the system:

- **Amla Powder** (PROD00000005) - 4 images
- **Ashwagandha Powder** (PROD00000003) - 4 images  
- **Beetroot Powder** (PROD00000006) - 4 images
- **Moringa Powder** (PROD00000004) - 6 images
- **Papaya Powder** (PROD00000001) - 4 images
- **Spinach Powder** (PROD00000007) - 4 images
- **Tomato Powder** (PROD00000002) - 4 images

**Total: 30 images across 7 products**

## Image Data Structure

Each image is stored with the following metadata:
- **Product ID**: Links to the specific product
- **Image URL**: S3 URL to the image file
- **Image Type**: Set to "product" for all images
- **Alt Text**: Descriptive text for accessibility
- **Caption**: Display caption for the image
- **Sort Order**: Order of display (0-based)
- **Is Primary**: First image of each product is marked as primary
- **MIME Type**: Automatically detected from file extension
- **Is Active**: All images are active by default

## Prerequisites

1. **Database Setup**: PostgreSQL database must be running
2. **Docker Compose**: Use the provided docker-compose.yml to start the database
3. **Environment Variables**: Database connection settings

## Running the Seeding Script

### Option 1: Using Docker Compose (Recommended)

1. Start the database:
   ```bash
   docker compose up -d postgres
   ```

2. Wait for the database to be ready (check health status)

3. Run the seeding script:
   ```bash
   go run cmd/seed-images/main.go
   ```

### Option 2: Direct Database Connection

If you have PostgreSQL running locally or have different connection settings:

1. Set environment variables:
   ```bash
   export POSTGRES_DSN="postgresql://neondb_owner:password@ep-your-neon-host.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
   ```

2. Run the seeding script:
   ```bash
   go run cmd/seed-images/main.go
   ```

## Database Configuration

The script uses the following default database settings:
- **Host**: localhost
- **Port**: 5555 (Docker Compose) or 5432 (default PostgreSQL)
- **User**: postgres
- **Password**: 12345678
- **Database**: DemoDB
- **SSL Mode**: require (override `DB_POSTGRES_SSLMODE` for local testing)

## Image URLs

All images are hosted on AWS S3 at:
```
https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/
```

### Product Image Structure:
- **Amla**: `/Amla/` (4 images)
- **Ashwagandha**: `/Aswagandha/` (4 images)
- **Beetroot**: `/Beetroot/` (4 images)
- **Moringa**: `/Moringa/` (6 images)
- **Papaya**: `/Papaya/` (4 images)
- **Spinach**: `/Spinach/` (4 images)
- **Tomato**: `/Tomato/` (4 images)

## Features

- **Automatic ID Generation**: Uses the kisanlink-db package for unique ID generation
- **MIME Type Detection**: Automatically detects image type from file extension
- **Primary Image Selection**: First image of each product is marked as primary
- **Sort Order Management**: Images are ordered sequentially
- **Error Handling**: Continues processing even if individual images fail
- **Comprehensive Logging**: Detailed logs for each operation

## Output

The script provides detailed logging including:
- Total images processed per product
- Individual image details (URL, alt text, caption, MIME type)
- Success/failure status for each image
- Final summary with total counts

## Troubleshooting

### Database Connection Issues
- Ensure PostgreSQL is running
- Check database credentials
- Verify network connectivity
- Check if port 5555 (Docker) or 5432 (default) is accessible

### Image URL Issues
- Verify S3 URLs are accessible
- Check image file existence
- Ensure proper URL encoding for special characters

### Permission Issues
- Ensure database user has INSERT permissions
- Check if images table exists and is properly migrated

## Database Schema

The script creates records in the `images` table with the following structure:
```sql
CREATE TABLE images (
    id VARCHAR PRIMARY KEY,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    product_id VARCHAR NOT NULL,
    image_url VARCHAR NOT NULL,
    image_type VARCHAR DEFAULT 'product',
    alt_text VARCHAR(255),
    caption VARCHAR(500),
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    is_primary BOOLEAN DEFAULT false,
    file_size BIGINT,
    width INTEGER,
    height INTEGER,
    mime_type VARCHAR(100)
);
```

## Cleanup

The script clears existing images before seeding new ones. To preserve existing data, comment out or remove this line:
```go
database.WithContext(ctx).Where("1 = 1").Delete(&models.Image{})
```

## Support

For issues or questions regarding the image seeding process, check:
1. Database connection logs
2. Image URL accessibility
3. Database permissions
4. Network connectivity
