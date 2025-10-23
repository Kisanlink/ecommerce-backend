# Admin Product Image Management Implementation

## Overview
This document describes the implementation of admin product image management functionality with S3 integration. The implementation allows admins to upload, delete, and manage product images through a RESTful API.

## Features Implemented

### 1. Image Upload
- **Endpoint**: `POST /api/v1/products/:id/images`
- **Authentication**: Admin only (Firebase Auth + Admin Guard)
- **Content-Type**: `multipart/form-data`
- **Parameters**:
  - `image` (file, required): Image file (JPEG, PNG, WebP)
  - `alt_text` (string, optional): Alt text for accessibility
  - `caption` (string, optional): Image caption
- **File Validation**:
  - Maximum file size: 10MB
  - Supported formats: JPEG, PNG, WebP
- **S3 Integration**: Automatically uploads to S3 bucket with organized folder structure
- **Database**: Creates image record with S3 key and presigned URL

### 2. Image Retrieval
- **Endpoint**: `GET /api/v1/products/:id/images`
- **Authentication**: Admin only
- **Response**: Returns all active images for a product with metadata

### 3. Image Deletion
- **Endpoint**: `DELETE /api/v1/products/:id/images/:imageId`
- **Authentication**: Admin only
- **Functionality**: 
  - Deletes image from S3 bucket
  - Removes image record from database
  - Automatically sets new primary image if deleted image was primary

### 4. Primary Image Management
- **Endpoint**: `PUT /api/v1/products/:id/images/:imageId/primary`
- **Authentication**: Admin only
- **Functionality**: Sets specified image as primary for the product

## Technical Implementation

### Database Schema
The `images` table includes:
- `id`: Unique image identifier
- `product_id`: Foreign key to products table
- `image_url`: Presigned URL for image access
- `s3_key`: S3 object key for direct access
- `image_type`: Type of image (product, thumbnail, etc.)
- `alt_text`: Alt text for accessibility
- `caption`: Image caption
- `sort_order`: Display order
- `is_active`: Active status
- `is_primary`: Primary image flag
- `file_size`: File size in bytes
- `width`: Image width in pixels
- `height`: Image height in pixels
- `mime_type`: MIME type of the image

### S3 Integration
- **Bucket Structure**: `images/product-images/{product_id}/product_{timestamp}.{ext}`
- **Presigned URLs**: 24-hour expiration for secure access
- **Automatic Cleanup**: Failed uploads are cleaned up from S3

### Security Features
- **Admin Authentication**: All endpoints require admin role
- **File Validation**: Strict file type and size validation
- **S3 Security**: Presigned URLs with expiration
- **Input Sanitization**: Proper validation of all inputs

## API Endpoints Summary

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/products/:id/images` | Upload product image | Admin |
| GET | `/api/v1/products/:id/images` | Get product images | Admin |
| DELETE | `/api/v1/products/:id/images/:imageId` | Delete product image | Admin |
| PUT | `/api/v1/products/:id/images/:imageId/primary` | Set primary image | Admin |

## Error Handling
- **400 Bad Request**: Invalid file type, size, or missing parameters
- **401 Unauthorized**: Missing or invalid authentication
- **403 Forbidden**: Non-admin user attempting admin operations
- **404 Not Found**: Product or image not found
- **500 Internal Server Error**: S3 upload failure or database errors

## Usage Examples

### Upload Image
```bash
curl -X POST "http://localhost:8080/api/v1/products/PROD123/images" \
  -H "Authorization: Bearer <admin_token>" \
  -F "image=@product_image.jpg" \
  -F "alt_text=Product front view" \
  -F "caption=Main product image"
```

### Get Product Images
```bash
curl -X GET "http://localhost:8080/api/v1/products/PROD123/images" \
  -H "Authorization: Bearer <admin_token>"
```

### Delete Image
```bash
curl -X DELETE "http://localhost:8080/api/v1/products/PROD123/images/IMG456" \
  -H "Authorization: Bearer <admin_token>"
```

### Set Primary Image
```bash
curl -X PUT "http://localhost:8080/api/v1/products/PROD123/images/IMG456/primary" \
  -H "Authorization: Bearer <admin_token>"
```

## Benefits
1. **Automated S3 Management**: No manual S3 bucket operations required
2. **Secure Access**: Presigned URLs with expiration
3. **Organized Storage**: Structured folder hierarchy in S3
4. **Database Consistency**: Proper foreign key relationships
5. **Admin Control**: Full CRUD operations for image management
6. **Primary Image Support**: Easy primary image designation
7. **Error Recovery**: Automatic cleanup of failed uploads

## Future Enhancements
- Image resizing and optimization
- Multiple image format support (thumbnails, different sizes)
- Bulk image upload
- Image metadata extraction (EXIF data)
- CDN integration for faster delivery

