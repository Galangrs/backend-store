package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"portolio-backend/configs"
	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/db"
	"portolio-backend/internal/model/dto"
	"portolio-backend/internal/util"
)

type ShopHandler struct {
	db *gorm.DB
}

func NewShopHandler(db *gorm.DB) *ShopHandler {
	return &ShopHandler{db: db}
}

func (h *ShopHandler) GetProductsRequest(c *gin.Context) {
	pageStr := c.Query("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 10
	offset := (page - 1) * limit

	roleInterface, _ := c.Get("ROLE")
	role := constants.RoleGuest
	if r, ok := roleInterface.(constants.UserRole); ok {
		role = r
	}

	userID := uint(0)
	if uid, exists := c.Get("ID"); exists {
		if idUint, ok := uid.(uint); ok {
			userID = idUint
		}
	}

	var products []db.Product
	query := h.db.Preload("Images").Preload("Reviews", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at desc").Limit(3).Preload("User")
	})

	switch role {
	case constants.RoleAdmin:
		query = query.Where(
			h.db.Where("visibility = ?", constants.ProductVisibilityAll).
				Or("visibility = ? AND user_id = ?", constants.ProductVisibilityOwnerAdmin, userID).
				Or("visibility = ?", constants.ProductVisibilityAdminOnly),
		)
	case constants.RoleUser:
		query = query.Where(
			h.db.Where("visibility = ?", constants.ProductVisibilityAll).
				Or("visibility = ? AND user_id = ?", constants.ProductVisibilityOwnerAdmin, userID),
		)
	default:
		query = query.Where("visibility = ?", constants.ProductVisibilityAll)
	}

	query = query.Where("deleted_at IS NULL")

	query = query.Order("created_at desc").Limit(limit).Offset(offset)

	if err := query.Find(&products).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	responseProducts := make([]dto.GetProductsResponse, len(products))
	for i, p := range products {
		categories := filterCategories(p.Categories)

		images := make([]dto.ProductImageResponse, len(p.Images))
		for j, img := range p.Images {
			images[j] = dto.ProductImageResponse{
				ID:        img.ID,
				ProductID: img.ProductID,
				ImageURL:  img.ImageURL,
			}
		}

		avgRating := calculateAverageRating(p.Reviews)
		reviewResp := buildReviewResponses(p.Reviews)

		responseProducts[i] = dto.GetProductsResponse{
			Id:         p.ID,
			Title:      p.Title,
			Price:      p.Price,
			Stock:      p.Stock,
			Categories: categories,
			Images:     images,
			Rating:     avgRating,
			Reviews:    reviewResp,
		}
	}
	util.RespondJSON(c, http.StatusOK, responseProducts)
}

func (h *ShopHandler) PostProductsRequest(c *gin.Context) {
	var req dto.RequestPostProduct
	var productWithImages db.Product

	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	visibility := req.Visibility
	if visibility == "" {
		visibility = constants.ProductVisibilityAll
	}
	if visibility != constants.ProductVisibilityAll && visibility != constants.ProductVisibilityOwnerAdmin {
		util.RespondJSON(c, http.StatusBadRequest, "Visibilitas tidak valid. Harus 'all' atau 'owner_admin'.")
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		product := db.Product{
			Title:      req.Title,
			Price:      req.Price,
			Stock:      req.Stock,
			Visibility: visibility,
			Categories: req.Categories,
			UserID:     userID,
			IsActive:   true,
		}

		if err := tx.Create(&product).Error; err != nil {
			return fmt.Errorf("Gagal membuat produk: %v", err)
		}

		var productImages []db.ProductImage

		form, err := c.MultipartForm()
		if err == nil && form != nil {
			files := form.File["images"]
			for _, file := range files {
				filePath, err := util.SaveUploadedFile(file, "media/products", constants.MaxImageSizeMB*1024*1024, util.AllowedImageExtensions)
				if err != nil {
					return fmt.Errorf("Gagal mengunggah gambar: %v", err)
				}
				productImages = append(productImages, db.ProductImage{
					ProductID: product.ID,
					ImageURL:  filepath.ToSlash(filePath),
				})
			}
		}

		for _, link := range req.ImagesLinks {
			filePath, err := util.DownloadImage(link, "media/products", constants.MaxImageSizeMB*1024*1024)
			if err != nil {
				return fmt.Errorf("Gagal mengunduh gambar dari URL: %v", err)
			}
			productImages = append(productImages, db.ProductImage{
				ProductID: product.ID,
				ImageURL:  filepath.ToSlash(filePath),
			})
		}

		if len(productImages) > 0 {
			if err := tx.Create(&productImages).Error; err != nil {
				return fmt.Errorf("Gagal menyimpan gambar produk: %v", err)
			}
		}

		if err := tx.Preload("Images").First(&productWithImages, product.ID).Error; err != nil {
			return fmt.Errorf("Gagal mengambil produk setelah dibuat: %v", err)
		}

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	responseImages := make([]dto.ProductImageResponse, len(productWithImages.Images))
	for i, img := range productWithImages.Images {
		responseImages[i] = dto.ProductImageResponse{
			ID:        img.ID,
			ProductID: img.ProductID,
			ImageURL:  img.ImageURL,
		}
	}
	responseProduct := dto.ProductDetailResponse{
		ID:         productWithImages.ID,
		UserID:     productWithImages.UserID,
		Title:      productWithImages.Title,
		Price:      productWithImages.Price,
		Stock:      productWithImages.Stock,
		Visibility: productWithImages.Visibility,
		Categories: filterCategories(productWithImages.Categories),
		IsActive:   productWithImages.IsActive,
		Images:     responseImages,
		CreatedAt:  productWithImages.CreatedAt,
		UpdatedAt:  productWithImages.UpdatedAt,
		DeletedAt:  &productWithImages.DeletedAt.Time,
	}
	if !productWithImages.DeletedAt.Valid {
		responseProduct.DeletedAt = nil
	}

	util.RespondJSON(c, http.StatusCreated, responseProduct)
}

func (h *ShopHandler) PutProductsRequest(c *gin.Context) {
	var req dto.RequestPutProduct

	productIDStr := c.Param("id")
	if productIDStr == "" {
		util.RespondJSON(c, http.StatusBadRequest, "ID produk diperlukan.")
		return
	}

	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var product db.Product
		if err := tx.First(&product, "id = ? AND user_id = ?", productIDStr, userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf(constants.ErrMsgProductNotFound)
			}
			return fmt.Errorf(constants.ErrMsgInternalServerError)
		}

		updates := make(map[string]interface{})
		if req.Title != "" {
			updates["title"] = req.Title
		}
		if req.Price != 0 {
			updates["price"] = req.Price
		}
		if req.Stock != 0 {
			updates["stock"] = req.Stock
			if req.Stock == 0 {
				updates["visibility"] = constants.ProductVisibilityOwnerAdmin
			} else if product.Stock == 0 && req.Stock > 0 && product.Visibility == constants.ProductVisibilityOwnerAdmin {
				updates["visibility"] = constants.ProductVisibilityAll
			}
		}
		if req.Visibility != "" {
			if req.Visibility != constants.ProductVisibilityAll && req.Visibility != constants.ProductVisibilityOwnerAdmin {
				return fmt.Errorf("Visibilitas tidak valid. Harus 'all' atau 'owner_admin'.")
			}
			if req.Visibility == constants.ProductVisibilityAll && product.Stock == 0 {
				return fmt.Errorf("Produk tidak dapat ditampilkan ke publik karena stok 0. Mohon perbarui stok terlebih dahulu.")
			}
			updates["visibility"] = req.Visibility
		}
		if req.Categories != "" {
			updates["categories"] = req.Categories
		}
		if req.IsActive != nil {
			updates["is_active"] = *req.IsActive
		}

		if len(updates) == 0 {
			return fmt.Errorf(constants.ErrMsgNoFieldsToUpdate)
		}

		if err := tx.Model(&product).Updates(updates).Error; err != nil {
			return fmt.Errorf("Gagal memperbarui produk: %v", err)
		}

		var productImages []db.ProductImage

		form, err := c.MultipartForm()
		if err == nil && form != nil {
			files := form.File["images"]
			for _, file := range files {
				filePath, err := util.SaveUploadedFile(file, "media/products", constants.MaxImageSizeMB*1024*1024, util.AllowedImageExtensions)
				if err != nil {
					return fmt.Errorf("Gagal mengunggah gambar: %v", err)
				}
				productImages = append(productImages, db.ProductImage{
					ProductID: product.ID,
					ImageURL:  filepath.ToSlash(filePath),
				})
			}
		}

		for _, link := range req.ImagesLinks {
			filePath, err := util.DownloadImage(link, "media/products", constants.MaxImageSizeMB*1024*1024)
			if err != nil {
				return fmt.Errorf("Gagal mengunduh gambar dari URL: %v", err)
			}
			productImages = append(productImages, db.ProductImage{
				ProductID: product.ID,
				ImageURL:  filepath.ToSlash(filePath),
			})
		}

		if len(productImages) > 0 {
			if err := tx.Create(&productImages).Error; err != nil {
				return fmt.Errorf("Gagal menyimpan gambar produk baru: %v", err)
			}
		}

		var productWithImages db.Product
		if err := tx.Preload("Images").First(&productWithImages, product.ID).Error; err != nil {
			return fmt.Errorf("Gagal mengambil produk setelah diperbarui: %v", err)
		}

		responseImages := make([]dto.ProductImageResponse, len(productWithImages.Images))
		for i, img := range productWithImages.Images {
			responseImages[i] = dto.ProductImageResponse{
				ID:        img.ID,
				ProductID: img.ProductID,
				ImageURL:  img.ImageURL,
			}
		}
		responseProduct := dto.ProductDetailResponse{
			ID:         productWithImages.ID,
			UserID:     productWithImages.UserID,
			Title:      productWithImages.Title,
			Price:      productWithImages.Price,
			Stock:      productWithImages.Stock,
			Visibility: productWithImages.Visibility,
			Categories: filterCategories(productWithImages.Categories),
			IsActive:   productWithImages.IsActive,
			Images:     responseImages,
			CreatedAt:  productWithImages.CreatedAt,
			UpdatedAt:  productWithImages.UpdatedAt,
			DeletedAt:  &productWithImages.DeletedAt.Time,
		}
		if !productWithImages.DeletedAt.Valid {
			responseProduct.DeletedAt = nil
		}

		util.RespondJSON(c, http.StatusOK, responseProduct)
		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
	}
}

func (h *ShopHandler) DeleteProductsRequest(c *gin.Context) {
	idVal, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := idVal.(uint)

	roleVal, exists := c.Get("ROLE")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	role := roleVal.(constants.UserRole)

	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var product db.Product
	if err := h.db.Unscoped().First(&product, productID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgProductNotFound)
		} else {
			util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		}
		return
	}

	if product.DeletedAt.Valid && role != constants.RoleAdmin {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgProductNotFound)
		return
	}

	if product.Visibility == constants.ProductVisibilityOwnerAdmin && role != constants.RoleAdmin && product.UserID != userID {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgProductNotFound)
		return
	}

	if role != constants.RoleAdmin && product.UserID != userID {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		product.Visibility = constants.ProductVisibilityOwnerAdmin
		product.DeletedAt = gorm.DeletedAt{
			Time:  time.Now(),
			Valid: true,
		}
		if err := tx.Save(&product).Error; err != nil {
			return fmt.Errorf("failed to soft delete product: %v", err)
		}

		var pendingTransactions []db.TransactionHistory
		if err := tx.Where("product_id = ? AND status IN (?, ?)", product.ID, constants.TrxStatusPending, constants.TrxStatusWaitingOwner).Find(&pendingTransactions).Error; err != nil {
			return fmt.Errorf("failed to find pending transactions: %v", err)
		}

		for _, trx := range pendingTransactions {
			var buyer db.User
			if err := tx.First(&buyer, trx.UserID).Error; err != nil {
				return fmt.Errorf("failed to find buyer for transaction %d: %v", trx.ID, err)
			}
			refundAmount := trx.TotalPrice + trx.GovtTax
			newBalance := buyer.Balance + refundAmount
			if err := tx.Model(&buyer).Update("balance", newBalance).Error; err != nil {
				return fmt.Errorf("failed to refund buyer for transaction %d: %v", trx.ID, err)
			}
			tx.Create(&db.BalanceHistory{
				UserID:       buyer.ID,
				Description:  fmt.Sprintf("Refund dari pembatalan produk '%s' (ID: %d) karena produk dihapus", product.Title, product.ID),
				Amount:       int(refundAmount),
				LastBalance:  buyer.Balance,
				FinalBalance: newBalance,
				Status:       constants.BalanceStatusRefund,
			})

			tx.Model(&trx).Updates(map[string]interface{}{
				"status":        constants.TrxStatusCancel,
				"is_solved":     true,
				"receipt_status": constants.ReceiptCanceled,
			})

			notificationBuyer := db.Notification{
				UserID:    buyer.ID,
				Type:      constants.NotifTypePurchase,
				Message:   fmt.Sprintf("Pembelian produk '%s' Anda (ID: %d) dibatalkan karena produk dihapus. Dana telah dikembalikan.", product.Title, trx.ID),
				RelatedID: &trx.ID,
			}
			util.SendNotificationToUser(buyer.ID, dto.NotificationResponse{
				ID:        notificationBuyer.ID,
				Type:      notificationBuyer.Type,
				Message:   notificationBuyer.Message,
				RelatedID: notificationBuyer.RelatedID,
				CreatedAt: notificationBuyer.CreatedAt,
				IsRead:    notificationBuyer.IsRead,
			})
			tx.Create(&notificationBuyer)
		}

		var waitingUserTransactions []db.TransactionHistory
		if err := tx.Where("product_id = ? AND status = ?", product.ID, constants.TrxStatusWaitingUser).Find(&waitingUserTransactions).Error; err != nil {
			return fmt.Errorf("failed to find waiting_users transactions: %v", err)
		}
		for _, trx := range waitingUserTransactions {
			var owner db.User
			if err := tx.First(&owner, product.UserID).Error; err != nil {
				return fmt.Errorf("failed to find owner for transaction %d: %v", trx.ID, err)
			}
			sellerReceiveAmount := trx.TotalPrice - trx.EcommerceTax
			newOwnerBalance := owner.Balance + sellerReceiveAmount
			if err := tx.Model(&owner).Update("balance", newOwnerBalance).Error; err != nil {
				return fmt.Errorf("failed to pay owner for transaction %d: %v", trx.ID, err)
			}
			tx.Create(&db.BalanceHistory{
				UserID:       owner.ID,
				Description:  fmt.Sprintf("Pembayaran penjualan produk '%s' (ID: %d) karena produk dihapus", product.Title, product.ID),
				Amount:       int(sellerReceiveAmount),
				LastBalance:  owner.Balance,
				FinalBalance: newOwnerBalance,
				Status:       constants.BalanceStatusCredit,
			})

			tx.Model(&trx).Updates(map[string]interface{}{
				"status":        constants.TrxStatusSuccess,
				"is_solved":     true,
				"receipt_status": constants.ReceiptCompleted,
			})

			notificationBuyer := db.Notification{
				UserID:    trx.UserID,
				Type:      constants.NotifTypePurchase,
				Message:   fmt.Sprintf("Pembelian produk '%s' Anda (ID: %d) telah diselesaikan karena produk dihapus. Penjual telah dibayar.", product.Title, trx.ID),
				RelatedID: &trx.ID,
			}
			util.SendNotificationToUser(trx.UserID, dto.NotificationResponse{
				ID:        notificationBuyer.ID,
				Type:      notificationBuyer.Type,
				Message:   notificationBuyer.Message,
				RelatedID: notificationBuyer.RelatedID,
				CreatedAt: notificationBuyer.CreatedAt,
				IsRead:    notificationBuyer.IsRead,
			})
			tx.Create(&notificationBuyer)

			notificationSeller := db.Notification{
				UserID:    owner.ID,
				Type:      constants.NotifTypeSale,
				Message:   fmt.Sprintf("Pembayaran produk '%s' Anda (ID: %d) telah diproses karena produk dihapus.", product.Title, trx.ID),
				RelatedID: &trx.ID,
			}
			util.SendNotificationToUser(owner.ID, dto.NotificationResponse{
				ID:        notificationSeller.ID,
				Type:      notificationSeller.Type,
				Message:   notificationSeller.Message,
				RelatedID: notificationSeller.RelatedID,
				CreatedAt: notificationSeller.CreatedAt,
				IsRead:    notificationSeller.IsRead,
			})
			tx.Create(&notificationSeller)
		}

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessProductDeleted)
}

func (h *ShopHandler) PatchProductVisibility(c *gin.Context) {
	userIDVal, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	roleVal, exists := c.Get("ROLE")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}

	userID := userIDVal.(uint)
	role := roleVal.(constants.UserRole)

	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var product db.Product
	if err := h.db.First(&product, productID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgProductNotFound)
		} else {
			util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		}
		return
	}

	if role != constants.RoleAdmin && product.UserID != userID {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
		return
	}

	newVisibility := constants.ProductVisibilityAll
	message := constants.MsgSuccessVisibilityUpdated

	if product.Visibility == constants.ProductVisibilityAll {
		newVisibility = constants.ProductVisibilityOwnerAdmin
		message = "Visibilitas produk berhasil diubah menjadi 'owner_admin' (tersembunyi dari publik)."
	} else if product.Visibility == constants.ProductVisibilityOwnerAdmin {
		if product.Stock == 0 {
			util.RespondJSON(c, http.StatusBadRequest, "Produk tidak dapat ditampilkan ke publik karena stok 0. Mohon perbarui stok terlebih dahulu.")
			return
		}
		newVisibility = constants.ProductVisibilityAll
		message = "Visibilitas produk berhasil diubah menjadi 'all' (terlihat publik)."
	} else {
		util.RespondJSON(c, http.StatusBadRequest, "Visibilitas produk tidak dapat diubah melalui endpoint ini.")
		return
	}

	if err := h.db.Model(&product).Update("visibility", newVisibility).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	util.RespondJSON(c, http.StatusOK, message)
}

func (h *ShopHandler) PostPurchaseProduct(c *gin.Context) {
	var req []dto.RequestPurchaseItem

	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	if len(req) == 0 {
		util.RespondJSON(c, http.StatusBadRequest, "Daftar pembelian tidak boleh kosong.")
		return
	}

	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	var user db.User
	if err := h.db.First(&user, userID).Error; err != nil {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUserNotFound)
		return
	}

	govtTaxPercent := configs.GetEnvFloat("GOVT_TAX_PERCENT", constants.DefaultGovtTaxPercent)
	ecommerceTaxPercent := configs.GetEnvFloat("ECOMMERCE_TAX_PERCENT", constants.DefaultEcommerceTaxPercent)

	err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range req {
			var product db.Product
			if err := tx.Where("id = ? AND visibility = ? AND deleted_at IS NULL", item.ProductID, constants.ProductVisibilityAll).First(&product).Error; err != nil {
				return fmt.Errorf(constants.ErrMsgProductNotFound+" (ID: %d)", item.ProductID)
			}

			if product.UserID == user.ID {
				return fmt.Errorf(constants.ErrMsgProductSelfPurchase+": %s", product.Title)
			}

			if product.Stock < item.Quantity {
				return fmt.Errorf(constants.ErrMsgInsufficientStock+" untuk produk %s", product.Title)
			}

			basePrice := product.Price * item.Quantity
			govtTaxAmount := uint(float64(basePrice) * govtTaxPercent)
			ecommerceTaxAmount := uint(float64(basePrice) * ecommerceTaxPercent)

			totalPriceForBuyer := basePrice + govtTaxAmount

			if user.Balance < totalPriceForBuyer {
				return fmt.Errorf(constants.ErrMsgInsufficientBalance+" untuk produk %s", product.Title)
			}

			if err := tx.Model(&product).Update("stock", product.Stock-item.Quantity).Error; err != nil {
				return err
			}
			if product.Stock-item.Quantity == 0 {
				tx.Model(&product).Update("visibility", constants.ProductVisibilityOwnerAdmin)
			}

			user.Balance -= totalPriceForBuyer
			if err := tx.Model(&user).Update("balance", user.Balance).Error; err != nil {
				return err
			}

			trx := db.TransactionHistory{
				ProductID:    product.ID,
				UserID:       user.ID,
				Quantity:     item.Quantity,
				TotalPrice:   basePrice,
				GovtTax:      govtTaxAmount,
				EcommerceTax: ecommerceTaxAmount,
				Status:       constants.TrxStatusPending,
				IsSolved:     false,
				ReceiptStatus: constants.ReceiptPendingProcess,
			}
			if err := tx.Create(&trx).Error; err != nil {
				return err
			}

			history := db.BalanceHistory{
				UserID:       user.ID,
				Description:  fmt.Sprintf("Pembelian '%s' x%d (termasuk pajak pemerintah Rp%d)", product.Title, item.Quantity, govtTaxAmount),
				Amount:       -int(totalPriceForBuyer),
				LastBalance:  user.Balance + totalPriceForBuyer,
				FinalBalance: user.Balance,
				Status:       constants.BalanceStatusDebit,
			}
			if err := tx.Create(&history).Error; err != nil {
				return err
			}

			notificationSeller := db.Notification{
				UserID:    product.UserID,
				Type:      constants.NotifTypeSale,
				Message:   fmt.Sprintf("Produk '%s' Anda telah dibeli oleh %s (x%d). Menunggu konfirmasi pengiriman.", product.Title, user.FullName, item.Quantity),
				RelatedID: &trx.ID,
			}
			if err := tx.Create(&notificationSeller).Error; err != nil {
				return err
			}
			util.SendNotificationToUser(product.UserID, dto.NotificationResponse{
				ID:        notificationSeller.ID,
				Type:      notificationSeller.Type,
				Message:   notificationSeller.Message,
				RelatedID: notificationSeller.RelatedID,
				CreatedAt: notificationSeller.CreatedAt,
				IsRead:    notificationSeller.IsRead,
			})

			notificationBuyer := db.Notification{
				UserID:    user.ID,
				Type:      constants.NotifTypePurchase,
				Message:   fmt.Sprintf("Pembelian '%s' Anda berhasil! Menunggu konfirmasi pengiriman dari penjual.", product.Title),
				RelatedID: &trx.ID,
			}
			if err := tx.Create(&notificationBuyer).Error; err != nil {
				return err
			}
			util.SendNotificationToUser(user.ID, dto.NotificationResponse{
				ID:        notificationBuyer.ID,
				Type:      notificationBuyer.Type,
				Message:   notificationBuyer.Message,
				RelatedID: notificationBuyer.RelatedID,
				CreatedAt: notificationBuyer.CreatedAt,
				IsRead:    notificationBuyer.IsRead,
			})
		}
		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessPurchase)
}

func (h *ShopHandler) ConfirmTransactionByOwner(c *gin.Context) {
	var req dto.RequestConfirmTransactionByOwner
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	ownerID, ok := userIDRaw.(uint)
	if !ok {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, trxID := range req.TransactionIDs {

			var trx db.TransactionHistory
			if err := tx.Preload("Product").First(&trx, trxID).Error; err != nil {
				return fmt.Errorf(constants.ErrMsgTransactionNotFound+" (ID: %d)", trxID)
			}

			if trx.Product.UserID != ownerID {
				return fmt.Errorf(constants.ErrMsgNotProductOwner+": transaksi %d", trxID)
			}

			if trx.Status != constants.TrxStatusPending {
				return fmt.Errorf(constants.ErrMsgTransactionNotConfirmable+" (ID: %d, status: %s)", trxID, trx.Status)
			}

			if err := tx.Model(&trx).Updates(map[string]interface{}{
				"status":    constants.TrxStatusWaitingUser,
				"is_solved": false,
			}).Error; err != nil {
				return fmt.Errorf("Gagal memperbarui transaksi %d: %v", trxID, err)
			}

			notificationBuyer := db.Notification{
				UserID:    trx.UserID,
				Type:      constants.NotifTypePurchase,
				Message:   fmt.Sprintf("Penjual telah mengkonfirmasi pengiriman produk '%s' (ID: %d). Mohon konfirmasi penerimaan setelah barang sampai.", trx.Product.Title, trx.ID),
				RelatedID: &trx.ID,
			}
			util.SendNotificationToUser(trx.UserID, dto.NotificationResponse{
				ID:        notificationBuyer.ID,
				Type:      notificationBuyer.Type,
				Message:   notificationBuyer.Message,
				RelatedID: notificationBuyer.RelatedID,
				CreatedAt: notificationBuyer.CreatedAt,
				IsRead:    notificationBuyer.IsRead,
			})
			tx.Create(&notificationBuyer)

			notificationSeller := db.Notification{
				UserID:    ownerID,
				Type:      constants.NotifTypeSale,
				Message:   fmt.Sprintf("Anda telah mengkonfirmasi pengiriman produk '%s' (ID: %d). Menunggu konfirmasi pembeli.", trx.Product.Title, trx.ID),
				RelatedID: &trx.ID,
			}
			util.SendNotificationToUser(ownerID, dto.NotificationResponse{
				ID:        notificationSeller.ID,
				Type:      notificationSeller.Type,
				Message:   notificationSeller.Message,
				RelatedID: notificationSeller.RelatedID,
				CreatedAt: notificationSeller.CreatedAt,
				IsRead:    notificationSeller.IsRead,
			})
			tx.Create(&notificationSeller)
		}
		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessTransactionConfirmed)
}

func (h *ShopHandler) ConfirmTransactionByUser(c *gin.Context) {
	var req dto.RequestConfirmTransactionByUser
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, trxID := range req.TransactionIDs {
			var trx db.TransactionHistory
			if err := tx.Preload("Product").First(&trx, "id = ? AND user_id = ?", trxID, userID).Error; err != nil {
				return fmt.Errorf(constants.ErrMsgTransactionNotFound+" (ID: %d) atau bukan milik pengguna", trxID)
			}

			if trx.Status != constants.TrxStatusWaitingUser {
				return fmt.Errorf(constants.ErrMsgTransactionNotConfirmable+" (ID: %d, status: %s)", trxID, trx.Status)
			}

			var product db.Product
			if err := tx.First(&product, trx.ProductID).Error; err != nil {
				return fmt.Errorf(constants.ErrMsgProductNotFound+" (ID: %d)", trx.ProductID)
			}

			var owner db.User
			if err := tx.First(&owner, product.UserID).Error; err != nil {
				return fmt.Errorf("Gagal mengambil info pemilik produk untuk transaksi %d: %v", trxID, err)
			}

			sellerReceiveAmount := trx.TotalPrice - trx.EcommerceTax
			newOwnerBalance := owner.Balance + sellerReceiveAmount

			history := db.BalanceHistory{
				UserID:       owner.ID,
				Description:  fmt.Sprintf("Pembayaran penjualan produk '%s' (ID: %d) dari pembeli ID %d", product.Title, product.ID, userID),
				Amount:       int(sellerReceiveAmount),
				LastBalance:  owner.Balance,
				FinalBalance: newOwnerBalance,
				Status:       constants.BalanceStatusCredit,
			}
			if err := tx.Create(&history).Error; err != nil {
				return fmt.Errorf("Gagal membuat riwayat saldo untuk transaksi %d: %v", trxID, err)
			}

			if err := tx.Model(&owner).Update("balance", newOwnerBalance).Error; err != nil {
				return fmt.Errorf("Gagal memperbarui saldo penjual untuk transaksi %d: %v", trxID, err)
			}

			if err := tx.Model(&trx).Updates(map[string]interface{}{
				"status":        constants.TrxStatusSuccess,
				"is_solved":     true,
				"receipt_status": constants.ReceiptCompleted,
			}).Error; err != nil {
				return fmt.Errorf("Gagal memperbarui transaksi %d: %v", trxID, err)
			}

			for _, r := range req.Reviews {
				if r.TransactionID == trxID && r.Rating != nil {
					review := db.Review{
						UserID:    userID,
						ProductID: trx.ProductID,
						Rating:    *r.Rating,
						Comment:   r.Comment,
					}
					if err := tx.Create(&review).Error; err != nil {
						return fmt.Errorf("Gagal menyimpan ulasan untuk transaksi %d: %v", trxID, err)
					}
				}
			}

			notificationSeller := db.Notification{
				UserID:    owner.ID,
				Type:      constants.NotifTypeSale,
				Message:   fmt.Sprintf("Pembeli telah mengkonfirmasi penerimaan produk '%s' (ID: %d). Dana telah ditransfer ke saldo Anda.", product.Title, product.ID),
				RelatedID: &trx.ID,
			}
			util.SendNotificationToUser(owner.ID, dto.NotificationResponse{
				ID:        notificationSeller.ID,
				Type:      notificationSeller.Type,
				Message:   notificationSeller.Message,
				RelatedID: notificationSeller.RelatedID,
				CreatedAt: notificationSeller.CreatedAt,
				IsRead:    notificationSeller.IsRead,
			})
			tx.Create(&notificationSeller)

			notificationBuyer := db.Notification{
				UserID:    userID,
				Type:      constants.NotifTypePurchase,
				Message:   fmt.Sprintf("Transaksi produk '%s' (ID: %d) telah berhasil diselesaikan.", product.Title, product.ID),
				RelatedID: &trx.ID,
			}
			util.SendNotificationToUser(userID, dto.NotificationResponse{
				ID:        notificationBuyer.ID,
				Type:      notificationBuyer.Type,
				Message:   notificationBuyer.Message,
				RelatedID: notificationBuyer.RelatedID,
				CreatedAt: notificationBuyer.CreatedAt,
				IsRead:    notificationBuyer.IsRead,
			})
			tx.Create(&notificationBuyer)
		}
		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessTransactionConfirmed)
}

func (h *ShopHandler) CancelTransaction(c *gin.Context) {
	var req dto.RequestCancelTransaction
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID, ok := userIDRaw.(uint)
	if !ok {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, trxID := range req.TransactionIDs {

			var trx db.TransactionHistory
			if err := tx.Preload("Product").First(&trx, trxID).Error; err != nil {
				return fmt.Errorf(constants.ErrMsgTransactionNotFound+" (ID: %d)", trxID)
			}

			if userID != trx.Product.UserID && userID != trx.UserID {
				return fmt.Errorf(constants.ErrMsgForbidden+" untuk membatalkan transaksi %d", trxID)
			}

			if trx.Status != constants.TrxStatusPending && trx.Status != constants.TrxStatusWaitingOwner {
				return fmt.Errorf(constants.ErrMsgTransactionNotCancellable+" (ID: %d, status: %s)", trxID, trx.Status)
			}

			var buyer db.User
			if err := tx.First(&buyer, trx.UserID).Error; err != nil {
				return fmt.Errorf("Pembeli ID %d tidak ditemukan", trx.UserID)
			}

			refundAmount := trx.TotalPrice + trx.GovtTax
			newBalance := buyer.Balance + refundAmount
			if err := tx.Model(&buyer).Update("balance", newBalance).Error; err != nil {
				return fmt.Errorf("Gagal mengembalikan dana ke pembeli %d", buyer.ID)
			}

			log := db.BalanceHistory{
				UserID:       buyer.ID,
				Description:  fmt.Sprintf("Refund dari pembatalan transaksi %d produk '%s'", trxID, trx.Product.Title),
				Amount:       int(refundAmount),
				LastBalance:  buyer.Balance,
				FinalBalance: newBalance,
				Status:       constants.BalanceStatusRefund,
			}
			if err := tx.Create(&log).Error; err != nil {
				return fmt.Errorf("Gagal membuat log refund untuk transaksi %d", trxID)
			}

			var product db.Product
			if err := tx.First(&product, trx.ProductID).Error; err != nil {
				return fmt.Errorf("Produk ID %d tidak ditemukan", trx.ProductID)
			}
			product.Stock += trx.Quantity
			if err := tx.Save(&product).Error; err != nil {
				return fmt.Errorf("Gagal mengembalikan stok produk %d", product.ID)
			}
			if product.Stock > 0 && product.Visibility == constants.ProductVisibilityOwnerAdmin {
				tx.Model(&product).Update("visibility", constants.ProductVisibilityAll)
			}

			if err := tx.Model(&trx).Updates(map[string]interface{}{
				"status":        constants.TrxStatusCancel,
				"is_solved":     true,
				"receipt_status": constants.ReceiptCanceled,
			}).Error; err != nil {
				return fmt.Errorf("Gagal membatalkan transaksi %d", trxID)
			}

			notificationBuyer := db.Notification{
				UserID:    buyer.ID,
				Type:      constants.NotifTypePurchase,
				Message:   fmt.Sprintf("Pembelian produk '%s' Anda (ID: %d) telah dibatalkan. Dana telah dikembalikan.", trx.Product.Title, trx.ID),
				RelatedID: &trx.ID,
			}
			util.SendNotificationToUser(buyer.ID, dto.NotificationResponse{
				ID:        notificationBuyer.ID,
				Type:      notificationBuyer.Type,
				Message:   notificationBuyer.Message,
				RelatedID: notificationBuyer.RelatedID,
				CreatedAt: notificationBuyer.CreatedAt,
				IsRead:    notificationBuyer.IsRead,
			})
			tx.Create(&notificationBuyer)

			notificationSeller := db.Notification{
				UserID:    trx.Product.UserID,
				Type:      constants.NotifTypeSale,
				Message:   fmt.Sprintf("Transaksi produk '%s' (ID: %d) dibatalkan oleh pembeli. Stok produk telah dikembalikan.", trx.Product.Title, trx.ID),
				RelatedID: &trx.ID,
			}
			util.SendNotificationToUser(trx.Product.UserID, dto.NotificationResponse{
				ID:        notificationSeller.ID,
				Type:      notificationSeller.Type,
				Message:   notificationSeller.Message,
				RelatedID: notificationSeller.RelatedID,
				CreatedAt: notificationSeller.CreatedAt,
				IsRead:    notificationSeller.IsRead,
			})
			tx.Create(&notificationSeller)
		}
		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err.Error())
		return
	}
	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessTransactionCanceled)
}

func (h *ShopHandler) GetCartHandler(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	const limit = 50
	offset := (page - 1) * limit

	var transactions []db.TransactionHistory
	if err := h.db.Preload("Product").Preload("Product.Images").Preload("User").
		Where("user_id = ? AND status IN ?", userID, []constants.TransactionStatus{constants.TrxStatusPending, constants.TrxStatusSuccess, constants.TrxStatusCancel, constants.TrxStatusWaitingUser, constants.TrxStatusWaitingOwner}).
		Order("CASE WHEN status = 'pending' THEN 1 WHEN status = 'waiting_owner' THEN 2 WHEN status = 'waiting_users' THEN 3 WHEN status = 'success' THEN 4 ELSE 5 END, created_at DESC").
		Offset(offset).Limit(limit).
		Find(&transactions).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	now := time.Now()
	var result []dto.CartProduct

	for _, t := range transactions {
		if t.Status == constants.TrxStatusCancel && now.Sub(t.CreatedAt) > 30*24*time.Hour {
			continue
		}

		var imageURL string
		if len(t.Product.Images) > 0 {
			imageURL = t.Product.Images[0].ImageURL
		}

		result = append(result, dto.CartProduct{
			ID:            t.ID,
			CreatedAt:     t.CreatedAt,
			UpdatedAt:     t.UpdatedAt,
			ProductID:     t.ProductID,
			UserID:        t.UserID,
			Quantity:      t.Quantity,
			TotalPrice:    t.TotalPrice,
			GovtTax:       t.GovtTax,
			EcommerceTax:  t.EcommerceTax,
			Status:        string(t.Status),
			IsSolved:      t.IsSolved,
			ReceiptStatus: string(t.ReceiptStatus),
			Product: dto.ProductDetailInCart{
				Title:  t.Product.Title,
				Price:  t.Product.Price,
				Images: t.Product.Images,
			},
			User: dto.UserDetailInCart{
				FullName: t.User.FullName,
			},
			ImageURL: imageURL,
		})
	}

	util.RespondJSON(c, http.StatusOK, result)
}

func (h *ShopHandler) GetOwnerProductOrders(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	status := c.Query("status")
	buyerIDStr := c.Query("buyer_id")
	receiptStatus := c.Query("receipt_status")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var ownedProductIDs []uint
	if err := h.db.Model(&db.Product{}).
		Where("user_id = ?", userID).
		Pluck("id", &ownedProductIDs).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	if len(ownedProductIDs) == 0 {
		util.RespondJSON(c, http.StatusOK, dto.GetOwnerProductOrdersResponse{
			Orders:  []db.TransactionHistory{},
		})
		return
	}

	var orders []db.TransactionHistory
	query := h.db.Where("product_id IN ?", ownedProductIDs).
		Preload("Product").
		Preload("User").
		Order("created_at DESC")

	if status != "" {
		validStatuses := map[string]bool{
			string(constants.TrxStatusPending):      true,
			string(constants.TrxStatusWaitingOwner): true,
			string(constants.TrxStatusWaitingUser):  true,
			string(constants.TrxStatusSuccess):      true,
			string(constants.TrxStatusCancel):       true,
		}
		if !validStatuses[status] {
			util.RespondJSON(c, http.StatusBadRequest, "Status transaksi tidak valid.")
			return
		}
		query = query.Where("status = ?", status)
	}

	if buyerIDStr != "" {
		buyerID, err := strconv.ParseUint(buyerIDStr, 10, 64)
		if err != nil {
			util.RespondJSON(c, http.StatusBadRequest, "ID pembeli tidak valid.")
			return
		}
		query = query.Where("user_id = ?", buyerID)
	}

	if receiptStatus != "" {
		validReceiptStatuses := map[string]bool{
			string(constants.ReceiptPendingProcess): true,
			string(constants.ReceiptCompleted):      true,
			string(constants.ReceiptCanceled):       true,
		}
		if !validReceiptStatuses[receiptStatus] {
			util.RespondJSON(c, http.StatusBadRequest, "Status receipt tidak valid.")
			return
		}
		query = query.Where("receipt_status = ?", receiptStatus)
	}

	var total int64
	query.Model(&db.TransactionHistory{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&orders).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	response := dto.GetOwnerProductOrdersResponse{
		TotalRecords: total,
		Page:          page,
		Limit:         limit,
		Orders:        orders,
	}

	util.RespondJSON(c, http.StatusOK, response)
}

func filterCategories(cats string) []string {
	raw := strings.Split(strings.TrimSpace(cats), ",")
	result := make([]string, 0)
	for _, cat := range raw {
		if val := strings.TrimSpace(cat); val != "" {
			result = append(result, val)
		}
	}
	return result
}

func calculateAverageRating(reviews []db.Review) float64 {
	if len(reviews) == 0 {
		return 0.0
	}
	var total uint
	for _, r := range reviews {
		total += r.Rating
	}
	return float64(total) / float64(len(reviews))
}

func buildReviewResponses(reviews []db.Review) []dto.ReviewResponse {
	resp := make([]dto.ReviewResponse, len(reviews))
	for i, r := range reviews {
		resp[i] = dto.ReviewResponse{
			UserID:   r.UserID,
			FullName: r.User.FullName,
			Rating:   r.Rating,
			Comment:  r.Comment,
		}
	}
	return resp
}