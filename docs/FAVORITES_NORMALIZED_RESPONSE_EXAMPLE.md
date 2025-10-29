# Favorites API - Normalized Response Structure

This document shows the expected response structure for the favorites API after implementing the normalized image and inventory system.

## Updated Response Structure

### GET /api/v1/favorites (Authenticated User)

```json
{
  "success": true,
  "favorites": [
    {
      "id": "FAV1761706764825459200",
      "created_at": "2025-10-29T08:29:24.825459+05:30",
      "updated_at": "2025-10-29T08:29:24.825459+05:30",
      "created_by": "",
      "updated_by": "",
      "user_id": "uUijBaiOAAUSLfWV9F3MwypEaZV2",
      "product_id": "PROD00000002",
      "product": {
        "id": "PROD00000002",
        "created_at": "2025-08-26T19:58:45.501428+05:30",
        "updated_at": "2025-08-26T19:58:45.501428+05:30",
        "created_by": "",
        "updated_by": "",
        "name": "Tomato Powder",
        "description": "Rich in lycopene and antioxidants, perfect for adding natural tomato flavor and nutrition to your meals",
        "category": "Superfoods",
        "price": 249,
        "actual_price": 329,
        "discount_percent": 24,
        "review_count": 33,
        "rating": 4.2,
        "is_active": true,
        "images": [
          {
            "id": "IMG1758178995519444400",
            "image_url": "https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Tomato/tomato+powder.png",
            "image_type": "product",
            "alt_text": "Tomato Powder Close-up",
            "caption": "Tomato Powder Detail",
            "sort_order": 0,
            "is_primary": true,
            "is_active": true,
            "file_size": 0,
            "width": 0,
            "height": 0,
            "mime_type": "image/png"
          },
          {
            "id": "IMG1758178995521578400",
            "image_url": "https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Tomato/web+tomato+powder.jpg",
            "image_type": "product",
            "alt_text": "Tomato Powder Web Image",
            "caption": "Tomato Powder Product",
            "sort_order": 1,
            "is_primary": false,
            "is_active": true,
            "file_size": 0,
            "width": 0,
            "height": 0,
            "mime_type": "image/jpeg"
          }
        ],
        "inventory": {
          "id": "INV1758255906904239700",
          "current_stock": 42,
          "reserved_stock": 0,
          "available_stock": 42,
          "min_stock_level": 10,
          "max_stock_level": 1000,
          "reorder_point": 20,
          "stock_status": "In Stock",
          "is_low_stock": false,
          "is_out_of_stock": false,
          "needs_reorder": false
        }
      }
    },
    {
      "id": "FAV1761706764814911100",
      "created_at": "2025-10-29T08:29:24.814911+05:30",
      "updated_at": "2025-10-29T08:29:24.814911+05:30",
      "created_by": "",
      "updated_by": "",
      "user_id": "uUijBaiOAAUSLfWV9F3MwypEaZV2",
      "product_id": "PROD00000005",
      "product": {
        "id": "PROD00000005",
        "created_at": "2025-08-26T19:58:45.501428+05:30",
        "updated_at": "2025-09-06T13:16:40.709687+05:30",
        "created_by": "",
        "updated_by": "",
        "name": "Amla Powder",
        "description": "Pure amla (Indian gooseberry) powder rich in Vitamin C and natural antioxidants for immunity boost.",
        "category": "Superfoods",
        "price": 95,
        "actual_price": 110,
        "discount_percent": 14,
        "review_count": 35,
        "rating": 4.6,
        "is_active": true,
        "images": [
          {
            "id": "IMG1758178995515021000",
            "image_url": "https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Amla/10.png",
            "image_type": "product",
            "alt_text": "Amla Powder Product",
            "caption": "Premium Amla Powder",
            "sort_order": 0,
            "is_primary": false,
            "is_active": true,
            "file_size": 0,
            "width": 0,
            "height": 0,
            "mime_type": "image/png"
          },
          {
            "id": "IMG1758178995517250000",
            "image_url": "https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Amla/amla+backside.jpg",
            "image_type": "product",
            "alt_text": "Amla Powder Back View",
            "caption": "Amla Powder Back View",
            "sort_order": 1,
            "is_primary": true,
            "is_active": true,
            "file_size": 0,
            "width": 0,
            "height": 0,
            "mime_type": "image/jpeg"
          },
          {
            "id": "IMG1758178995519444400",
            "image_url": "https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Amla/amla+powder.png",
            "image_type": "product",
            "alt_text": "Amla Powder Close-up",
            "caption": "Amla Powder Detail",
            "sort_order": 2,
            "is_primary": false,
            "is_active": true,
            "file_size": 0,
            "width": 0,
            "height": 0,
            "mime_type": "image/png"
          },
          {
            "id": "IMG1758178995521578400",
            "image_url": "https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Amla/web+amla+powder.jpg",
            "image_type": "product",
            "alt_text": "Amla Powder Web Image",
            "caption": "Amla Powder Product",
            "sort_order": 3,
            "is_primary": false,
            "is_active": true,
            "file_size": 0,
            "width": 0,
            "height": 0,
            "mime_type": "image/jpeg"
          },
          {
            "id": "IMG1759124940420731300",
            "image_url": "https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/PROD00000005/product_1759124940.jpeg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Checksum-Mode=ENABLED&X-Amz-Credential=AKIAWR4ABJZZTPJPBQQU%2F20250929%2Fap-south-1%2Fs3%2Faws4_request&X-Amz-Date=20250929T054900Z&X-Amz-Expires=86400&X-Amz-SignedHeaders=host&x-id=GetObject&X-Amz-Signature=7da30fc15437c71acfb18b8686d87c847c1e4a621e13821dff6c365e127c4166",
            "image_type": "product",
            "alt_text": "",
            "caption": "",
            "sort_order": 4,
            "is_primary": false,
            "is_active": true,
            "file_size": 107795,
            "width": 0,
            "height": 0,
            "mime_type": "image/jpeg"
          },
          {
            "id": "IMG1759124980496517600",
            "image_url": "https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/PROD00000005/product_1759124980.png?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Checksum-Mode=ENABLED&X-Amz-Credential=AKIAWR4ABJZZTPJPBQQU%2F20250929%2Fap-south-1%2Fs3%2Faws4_request&X-Amz-Date=20250929T054940Z&X-Amz-Expires=86400&X-Amz-SignedHeaders=host&x-id=GetObject&X-Amz-Signature=3fab5b9dffab7a00325c92b1d95101b9080820814a83fe41ee9f348110700f50",
            "image_type": "product",
            "alt_text": "",
            "caption": "",
            "sort_order": 5,
            "is_primary": false,
            "is_active": true,
            "file_size": 64969,
            "width": 0,
            "height": 0,
            "mime_type": "image/png"
          }
        ],
        "inventory": {
          "id": "INV1758255906904239700",
          "current_stock": 45,
          "reserved_stock": 0,
          "available_stock": 45,
          "min_stock_level": 10,
          "max_stock_level": 1000,
          "reorder_point": 20,
          "stock_status": "In Stock",
          "is_low_stock": false,
          "is_out_of_stock": false,
          "needs_reorder": false
        }
      }
    }
  ],
  "is_authenticated": true
}
```

### GET /api/v1/favorites (Anonymous User with Session)

```json
{
  "success": true,
  "favorites": {
    "session_id": "session_abc123def456",
    "favorites": [
      {
        "id": "SFAV1761706764825459200",
        "session_id": "session_abc123def456",
        "product_id": "PROD00000002",
        "added_at": "2025-10-29T08:29:24.825459+05:30",
        "product": {
          "id": "PROD00000002",
          "name": "Tomato Powder",
          "description": "Rich in lycopene and antioxidants, perfect for adding natural tomato flavor and nutrition to your meals",
          "category": "Superfoods",
          "price": 249,
          "actual_price": 329,
          "discount_percent": 24,
          "review_count": 33,
          "rating": 4.2,
          "is_active": true,
          "images": [
            {
              "id": "IMG1758178995519444400",
              "image_url": "https://amrti.s3.ap-south-1.amazonaws.com/images/product-images/Tomato/tomato+powder.png",
              "image_type": "product",
              "alt_text": "Tomato Powder Close-up",
              "caption": "Tomato Powder Detail",
              "sort_order": 0,
              "is_primary": true,
              "is_active": true,
              "file_size": 0,
              "width": 0,
              "height": 0,
              "mime_type": "image/png"
            }
          ],
          "inventory": {
            "id": "INV1758255906904239700",
            "current_stock": 42,
            "reserved_stock": 0,
            "available_stock": 42,
            "min_stock_level": 10,
            "max_stock_level": 1000,
            "reorder_point": 20,
            "stock_status": "In Stock",
            "is_low_stock": false,
            "is_out_of_stock": false,
            "needs_reorder": false
          }
        }
      }
    ],
    "total_count": 1,
    "last_updated": "2025-10-29T08:29:24.825459+05:30"
  },
  "is_authenticated": false,
  "session_id": "session_abc123def456"
}
```

## Key Changes Made

### 1. **Removed Deprecated Fields**
- ❌ `image_url`, `image_url_1`, `image_url_2`, `image_url_3`, `image_url_4`
- ❌ `stock`, `stock_status`

### 2. **Added Normalized Structure**
- ✅ `images[]` - Array of image objects with full metadata
- ✅ `inventory` - Complete inventory information object

### 3. **Image Object Structure**
```json
{
  "id": "IMG1758178995519444400",
  "image_url": "https://...",
  "image_type": "product",
  "alt_text": "Product description",
  "caption": "Image caption",
  "sort_order": 0,
  "is_primary": true,
  "is_active": true,
  "file_size": 0,
  "width": 0,
  "height": 0,
  "mime_type": "image/png"
}
```

### 4. **Inventory Object Structure**
```json
{
  "id": "INV1758255906904239700",
  "current_stock": 45,
  "reserved_stock": 0,
  "available_stock": 45,
  "min_stock_level": 10,
  "max_stock_level": 1000,
  "reorder_point": 20,
  "stock_status": "In Stock",
  "is_low_stock": false,
  "is_out_of_stock": false,
  "needs_reorder": false
}
```

## Benefits of This Structure

1. **Unlimited Images**: No longer limited to 5 images per product
2. **Rich Metadata**: Each image includes alt text, captions, file size, dimensions, etc.
3. **Proper Sorting**: Images are sorted by `sort_order` and creation date
4. **Primary Image Flag**: Clear indication of which image is primary
5. **Complete Inventory**: Full stock management information
6. **Computed Fields**: `available_stock` is calculated as `current_stock - reserved_stock`
7. **Status Flags**: Easy to check if product is low stock, out of stock, or needs reorder

## Migration Notes

- The old deprecated fields (`image_url_1`, `stock`, etc.) are still present in the Product model for backward compatibility
- They are no longer populated in the favorites API responses
- All new code should use the normalized `images[]` and `inventory` objects
- The deprecated fields will be removed in a future version after all clients are migrated
