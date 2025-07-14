package dto

import (
	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/db"
	"time"
)

type RequestPostProduct struct {
	Title       string                    `json:"title" binding:"required,min=5,max=100"`
	Price       uint                      `json:"price" binding:"required,gte=1000,lte=100000000"` // Min 1000, Max 100 juta
	Stock       uint                      `json:"stock" binding:"required,gte=0,lte=10000"`         // Max 10000
	Visibility  constants.ProductVisibility `json:"visibility,omitempty" binding:"omitempty,oneof=all owner_admin"`
	Categories  string                    `json:"categories,omitempty" binding:"omitempty,max=255"`
	ImagesLinks []string                  `json:"images_links,omitempty" binding:"omitempty,dive,url"`
}

type RequestPutProduct struct {
	Title       string                    `json:"title,omitempty" binding:"omitempty,min=5,max=100"`
	Price       uint                      `json:"price,omitempty" binding:"omitempty,gte=1000,lte=100000000"`
	Stock       uint                      `json:"stock,omitempty" binding:"omitempty,gte=0,lte=10000"`
	Visibility  constants.ProductVisibility `json:"visibility,omitempty" binding:"omitempty,oneof=all owner_admin"`
	Categories  string                    `json:"categories,omitempty" binding:"omitempty,max=255"`
	ImagesLinks []string                  `json:"images_links,omitempty" binding:"omitempty,dive,url"`
	IsActive    *bool                     `json:"is_active,omitempty"`
}

type RequestPurchaseItem struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  uint `json:"quantity" binding:"required,gt=0"`
}

type ProductImageResponse struct {
	ID        uint   `json:"id"`
	ProductID uint   `json:"product_id"`
	ImageURL  string `json:"image_url"`
}

type ProductDetailInCart struct {
	Title  string             `json:"title"`
	Price  uint               `json:"price"`
	Images []db.ProductImage `json:"images"` // Using db.ProductImage directly here for simplicity, could be ProductImageResponse
}

type UserDetailInCart struct {
	FullName string `json:"full_name"`
}

type CartProduct struct {
	ID            uint      `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ProductID     uint      `json:"product_id"`
	UserID        uint      `json:"user_id"`
	Quantity      uint      `json:"quantity"`
	TotalPrice    uint      `json:"total_price"`
	GovtTax       uint      `json:"govt_tax"`
	EcommerceTax  uint      `json:"ecommerce_tax"`
	Status        string    `json:"status"`
	IsSolved      bool      `json:"is_solved"`
	ReceiptStatus string    `json:"receipt_status"`
	Product       ProductDetailInCart `json:"product"`
	User          UserDetailInCart    `json:"user"`
	ImageURL      string              `json:"image_url,omitempty"`
}

type ReviewResponse struct {
	UserID   uint   `json:"user_id"`
	FullName string `json:"full_name"`
	Rating   uint   `json:"rating"`
	Comment  string `json:"comment"`
}

type GetProductsResponse struct {
	Id         uint                   `json:"id"`
	Title      string                 `json:"title"`
	Price      uint                   `json:"price"`
	Stock      uint                   `json:"stock"`
	Categories []string               `json:"categories"`
	Images     []ProductImageResponse `json:"images"`
	Rating     float64                `json:"rating"`
	Reviews    []ReviewResponse       `json:"reviews"`
}

type GetProductsRequestAdmin struct {
	Id         uint                   `json:"id"`
	Title      string                 `json:"title"`
	Price      uint                   `json:"price"`
	Stock      uint                   `json:"stock"`
	Visibility string                 `json:"visibility"`
	Categories []string               `json:"categories"`
	Images     []ProductImageResponse `json:"images"`
	Rating     float64                `json:"rating"`
	Reviews    []ReviewResponse       `json:"reviews"`
	UserID     uint                   `json:"user_id"`
	UserEmail  string                 `json:"user_email"`
	IsActive   bool                   `json:"is_active"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	DeletedAt  time.Time              `json:"deleted_at,omitempty"`
}

type ProductDetailResponse struct {
	ID         uint                      `json:"id"`
	UserID     uint                      `json:"user_id"`
	Title      string                    `json:"title"`
	Price      uint                      `json:"price"`
	Stock      uint                      `json:"stock"`
	Visibility constants.ProductVisibility `json:"visibility"`
	Categories []string                  `json:"categories"`
	IsActive   bool                      `json:"is_active"`
	Images     []ProductImageResponse    `json:"images"`
	CreatedAt  time.Time                 `json:"created_at"`
	UpdatedAt  time.Time                 `json:"updated_at"`
	DeletedAt  *time.Time                `json:"deleted_at,omitempty"`
}

type RequestConfirmTransactionByOwner struct {
	TransactionIDs []uint `json:"transaction_ids" binding:"required,min=1,dive,gt=0"`
}

type ReviewItem struct {
	TransactionID uint   `json:"transaction_id" binding:"required"`
	Rating        *uint  `json:"rating" binding:"omitempty,gte=1,lte=5"`
	Comment       string `json:"comment,omitempty" binding:"omitempty,max=500"`
}

type RequestConfirmTransactionByUser struct {
	TransactionIDs []uint       `json:"transaction_ids" binding:"required,min=1,dive,gt=0"`
	Reviews        []ReviewItem `json:"reviews,omitempty"`
}

type RequestCancelTransaction struct {
	TransactionIDs []uint `json:"transaction_ids" binding:"required,min=1,dive,gt=0"`
}

type GetOwnerProductOrdersResponse struct {
	TotalRecords int64                       `json:"total_records"`
	Page         int                         `json:"page"`
	Limit        int                         `json:"limit"`
	Orders       []db.TransactionHistory `json:"orders"` 
}