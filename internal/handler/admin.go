package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"crypto/sha256"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"portolio-backend/configs"
	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/db"
	"portolio-backend/internal/model/dto"
	"portolio-backend/internal/util"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) PostAdminLogin(c *gin.Context) {
	var req dto.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))

	var user db.User
	if err := h.db.Where("email = ? AND role = ?", email, constants.RoleAdmin).First(&user).Error; err != nil {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgAdminLoginFailed)
		return
	}

	if user.Status != constants.UserStatusActive {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgAccountSuspended)
		return
	}
	if user.DeletedAt.Valid {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgAccountDeleted)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgAdminLoginFailed)
		return
	}

	tokenJWTConfig := configs.GetTokenJWTConfig()
	sessionToken, err := util.GenerateAdminSessionToken(user.ID, string(user.Role), tokenJWTConfig.AdminExpireDuration, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	hashed := sha256.Sum256([]byte(sessionToken))
	tokenHash := hex.EncodeToString(hashed[:])

	if err := h.db.Where("user_id = ?", user.ID).Delete(&db.AdminSession{}).Error; err != nil {
		fmt.Printf("Failed to invalidate old admin sessions for user %d: %v\n", user.ID, err)
	}

	adminSession := db.AdminSession{
		UserID:    user.ID,
		TokenHash: string(tokenHash),
		ExpiresAt: time.Now().Add(tokenJWTConfig.AdminExpireDuration),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		LastUsedAt: time.Now(),
	}
	if err := h.db.Create(&adminSession).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	adminLog := db.AdminLog{
		AdminID:    user.ID,
		Action:     "admin_login",
		TargetType: "admin",
		TargetID:   &user.ID,
		Details:    db.JSONB{"email": user.Email, "ip_address": c.ClientIP()},
		IPAddress:  c.ClientIP(),
	}
	h.db.Create(&adminLog)

	c.Header("Authorization", "Bearer "+sessionToken)

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessAdminLogin)
}

func (h *AdminHandler) GetUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	search := c.Query("search")
	status := c.Query("status")
	role := constants.UserRole(c.Query("role"))
	if role == "" {
		role = constants.RoleUser
	}
	includeDeleted := c.DefaultQuery("include_deleted", "false") == "true"

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var users []db.User
	query := h.db.Order("created_at DESC")

	if includeDeleted {
		query = query.Unscoped()
	}

	if search != "" {
		query = query.Where("LOWER(full_name) LIKE ? OR LOWER(email) LIKE ?", "%"+strings.ToLower(search)+"%", "%"+strings.ToLower(search)+"%")
	}

	if status != "" {
		if status != string(constants.UserStatusActive) && status != string(constants.UserStatusSuspended) && status != string(constants.UserStatusBanned) {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid status. Must be 'active', 'suspended', or 'banned'.")
			return
		}
		query = query.Where("status = ?", status)
	}

	if role != "" {
		if role != constants.RoleUser && role != constants.RoleAdmin {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid role. Must be 'user' or 'admin'.")
			return
		}
		query = query.Where("role = ?", role)
	}

	var total int64
	query.Model(&db.User{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	userResponses := make([]dto.UserDetailResponse, len(users))
	for i, user := range users {
		userResponses[i] = dto.UserDetailResponse{
			ID:              user.ID,
			FullName:        user.FullName,
			Email:           user.Email,
			Role:            user.Role,
			Balance:         user.Balance,
			Status:          user.Status,
			BanUntil:        user.BanUntil,
			BanReason:       user.BanReason,
			PenaltyWarnings: user.PenaltyWarnings,
			CreatedAt:       user.CreatedAt,
			UpdatedAt:       user.UpdatedAt,
			DeletedAt:       &user.DeletedAt.Time,
		}
		if !user.DeletedAt.Valid {
			userResponses[i].DeletedAt = nil
		}
	}

	response := dto.GetUsersResponse{
		TotalRecords: total,
		Page:         page,
		Limit:        limit,
		Users:        userResponses,
	}

	util.RespondJSON(c, http.StatusOK, response)
}

func (h *AdminHandler) SuspendUser(c *gin.Context) {
	adminIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	adminID := adminIDRaw.(uint)

	userIDStr := c.Param("id")
	targetUserID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var req dto.RequestSuspendUser
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	var user db.User
	if err := h.db.First(&user, targetUserID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgUserNotFound)
		return
	}

	if user.Role == constants.RoleAdmin {
		util.RespondJSON(c, http.StatusForbidden, "Tidak dapat menangguhkan akun admin.")
		return
	}

	if user.Status == constants.UserStatusSuspended {
		util.RespondJSON(c, http.StatusOK, "Pengguna sudah ditangguhkan.")
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		user.Status = constants.UserStatusSuspended
		user.BanUntil = nil
		user.BanReason = req.Reason
		user.PenaltyWarnings = 0
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to suspend user: %v", err)
		}

		var products []db.Product
		if err := tx.Where("user_id = ?", user.ID).Find(&products).Error; err != nil {
			return fmt.Errorf("failed to find products for user: %v", err)
		}

		for _, product := range products {
			product.DeletedAt = gorm.DeletedAt{
				Time:  time.Now(),
				Valid: true,
			}
			product.Visibility = constants.ProductVisibilityOwnerAdmin
			if err := tx.Save(&product).Error; err != nil {
				return fmt.Errorf("failed to soft delete product %d: %v", product.ID, err)
			}
			handleProductTransactionsOnDelete(tx, product)
		}

		notification := db.Notification{
			UserID:    user.ID,
			Type:      constants.NotifTypeAccount,
			Message:   fmt.Sprintf("Akun Anda telah ditangguhkan secara permanen karena: %s. Mohon hubungi dukungan untuk informasi lebih lanjut.", req.Reason),
			RelatedID: &user.ID,
		}
		util.SendNotificationToUser(user.ID, dto.NotificationResponse{
			ID:        notification.ID,
			Type:      notification.Type,
			Message:   notification.Message,
			RelatedID: notification.RelatedID,
			CreatedAt: notification.CreatedAt,
			IsRead:    notification.IsRead,
		})
		tx.Create(&notification)

		adminLog := db.AdminLog{
			AdminID:    adminID,
			Action:     "suspend_user",
			TargetType: "user",
			TargetID:   &user.ID,
			Details:    db.JSONB{"user_email": user.Email, "reason": req.Reason},
			IPAddress:  c.ClientIP(),
		}
		tx.Create(&adminLog)

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessUserSuspended)
}

func (h *AdminHandler) BanUser(c *gin.Context) {
	adminIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	adminID := adminIDRaw.(uint)

	userIDStr := c.Param("id")
	targetUserID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var req dto.RequestBanUser
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	var user db.User
	if err := h.db.First(&user, targetUserID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgUserNotFound)
		return
	}

	if user.Role == constants.RoleAdmin {
		util.RespondJSON(c, http.StatusForbidden, "Tidak dapat memblokir akun admin.")
		return
	}

	if user.Status == constants.UserStatusBanned {
		util.RespondJSON(c, http.StatusOK, "Pengguna sudah diblokir.")
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		banUntil := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
		banUntilUnix := banUntil.Unix()

		user.Status = constants.UserStatusBanned
		user.BanUntil = &banUntilUnix
		user.BanReason = req.Reason
		user.PenaltyWarnings = 0
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to ban user: %v", err)
		}

		var products []db.Product
		if err := tx.Where("user_id = ?", user.ID).Find(&products).Error; err != nil {
			return fmt.Errorf("failed to find products for user: %v", err)
		}

		for _, product := range products {
			product.DeletedAt = gorm.DeletedAt{
				Time:  time.Now(),
				Valid: true,
			}
			product.Visibility = constants.ProductVisibilityOwnerAdmin
			if err := tx.Save(&product).Error; err != nil {
				return fmt.Errorf("failed to soft delete product %d: %v", product.ID, err)
			}
			handleProductTransactionsOnDelete(tx, product)
		}

		notification := db.Notification{
			UserID:    user.ID,
			Type:      constants.NotifTypeAccount,
			Message:   fmt.Sprintf("Akun Anda telah diblokir selama %d jam karena: %s. Anda tidak dapat login selama periode ini.", req.DurationHours, req.Reason),
			RelatedID: &user.ID,
		}
		util.SendNotificationToUser(user.ID, dto.NotificationResponse{
			ID:        notification.ID,
			Type:      notification.Type,
			Message:   notification.Message,
			RelatedID: notification.RelatedID,
			CreatedAt: notification.CreatedAt,
			IsRead:    notification.IsRead,
		})
		tx.Create(&notification)

		adminLog := db.AdminLog{
			AdminID:    adminID,
			Action:     "ban_user",
			TargetType: "user",
			TargetID:   &user.ID,
			Details:    db.JSONB{"user_email": user.Email, "duration_hours": req.DurationHours, "reason": req.Reason},
			IPAddress:  c.ClientIP(),
		}
		tx.Create(&adminLog)

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessUserBanned)
}

func (h *AdminHandler) UnbanUser(c *gin.Context) {
	adminIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	adminID := adminIDRaw.(uint)

	userIDStr := c.Param("id")
	targetUserID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var user db.User
	if err := h.db.First(&user, targetUserID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgUserNotFound)
		return
	}

	if user.Role == constants.RoleAdmin {
		util.RespondJSON(c, http.StatusForbidden, "Tidak dapat mengaktifkan kembali akun admin.")
		return
	}

	if user.Status == constants.UserStatusActive {
		util.RespondJSON(c, http.StatusOK, "Pengguna sudah aktif.")
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		user.Status = constants.UserStatusActive
		user.BanUntil = nil
		user.BanReason = ""
		user.PenaltyWarnings = 0
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to unban user: %v", err)
		}

		var products []db.Product
		if err := tx.Unscoped().Where("user_id = ? AND deleted_at IS NOT NULL", user.ID).Find(&products).Error; err != nil {
			return fmt.Errorf("failed to find soft-deleted products for user: %v", err)
		}

		for _, product := range products {
			product.DeletedAt = gorm.DeletedAt{}
			product.Visibility = constants.ProductVisibilityAll
			if err := tx.Save(&product).Error; err != nil {
				return fmt.Errorf("failed to reactivate product %d: %v", product.ID, err)
			}
		}

		notification := db.Notification{
			UserID:    user.ID,
			Type:      constants.NotifTypeAccount,
			Message:   "Akun Anda telah diaktifkan kembali. Anda sekarang dapat login dan menggunakan layanan kami.",
			RelatedID: &user.ID,
		}
		util.SendNotificationToUser(user.ID, dto.NotificationResponse{
			ID:        notification.ID,
			Type:      notification.Type,
			Message:   notification.Message,
			RelatedID: notification.RelatedID,
			CreatedAt: notification.CreatedAt,
			IsRead:    notification.IsRead,
		})
		tx.Create(&notification)

		adminLog := db.AdminLog{
			AdminID:    adminID,
			Action:     "unban_user",
			TargetType: "user",
			TargetID:   &user.ID,
			Details:    db.JSONB{"user_email": user.Email},
			IPAddress:  c.ClientIP(),
		}
		tx.Create(&adminLog)

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessUserUnbanned)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	adminIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	adminID := adminIDRaw.(uint)

	userIDStr := c.Param("id")
	targetUserID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var user db.User
	if err := h.db.Unscoped().First(&user, targetUserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgUserNotFound)
		} else {
			util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		}
		return
	}

	if user.DeletedAt.Valid {
		util.RespondJSON(c, http.StatusOK, "Pengguna sudah dihapus sebelumnya.")
		return
	}

	if user.ID == adminID {
		util.RespondJSON(c, http.StatusForbidden, "Admin tidak dapat menghapus akunnya sendiri.")
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		user.DeletedAt = gorm.DeletedAt{
			Time:  time.Now(),
			Valid: true,
		}
		user.Status = constants.UserStatusSuspended
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to soft delete user: %v", err)
		}

		var products []db.Product
		if err := tx.Where("user_id = ?", user.ID).Find(&products).Error; err != nil {
			return fmt.Errorf("failed to find products for user: %v", err)
		}

		for _, product := range products {
			product.DeletedAt = gorm.DeletedAt{
				Time:  time.Now(),
				Valid: true,
			}
			product.Visibility = constants.ProductVisibilityOwnerAdmin
			if err := tx.Save(&product).Error; err != nil {
				return fmt.Errorf("failed to soft delete product %d: %v", product.ID, err)
			}
			handleProductTransactionsOnDelete(tx, product)
		}

		adminLog := db.AdminLog{
			AdminID:    adminID,
			Action:     "delete_user",
			TargetType: "user",
			TargetID:   &user.ID,
			Details:    db.JSONB{"user_email": user.Email, "reason": "Admin menghapus akun"},
			IPAddress:  c.ClientIP(),
		}
		tx.Create(&adminLog)

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessAccountDeleted)
}

func (h *AdminHandler) GetProductsAdmin(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	search := c.Query("search")
	visibility := c.Query("visibility")
	userIDStr := c.Query("user_id")
	includeDeleted := c.DefaultQuery("include_deleted", "false") == "true"

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var products []db.Product
	query := h.db.Preload("Images").Preload("User").Preload("Reviews", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at desc").Limit(3).Preload("User")
	})

	if includeDeleted {
		query = query.Unscoped()
	}

	if search != "" {
		query = query.Where("LOWER(title) LIKE ? OR LOWER(categories) LIKE ?", "%"+strings.ToLower(search)+"%", "%"+strings.ToLower(search)+"%")
	}

	if visibility != "" {
		if visibility != string(constants.ProductVisibilityAll) && visibility != string(constants.ProductVisibilityOwnerAdmin) && visibility != string(constants.ProductVisibilityAdminOnly) {
			util.RespondJSON(c, http.StatusBadRequest, "Visibilitas tidak valid. Harus 'all', 'owner_admin', atau 'admin_only'.")
			return
		}
		query = query.Where("visibility = ?", visibility)
	}

	if userIDStr != "" {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid user ID.")
			return
		}
		query = query.Where("user_id = ?", userID)
	}

	var total int64
	query.Model(&db.Product{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&products).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	responseProducts := make([]dto.GetProductsRequestAdmin, len(products))
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

		responseProducts[i] = dto.GetProductsRequestAdmin{
			Id:         p.ID,
			Title:      p.Title,
			Price:      p.Price,
			Stock:      p.Stock,
			Visibility: string(p.Visibility),
			Categories: categories,
			Images:     images,
			Rating:     avgRating,
			Reviews:    reviewResp,
			UserID:     p.UserID,
			UserEmail:  p.User.Email,
			IsActive:   p.IsActive,
			CreatedAt:  p.CreatedAt,
			UpdatedAt:  p.UpdatedAt,
			DeletedAt:  p.DeletedAt.Time,
		}
	}

	response := dto.GetProductsAdminResponse{
		TotalRecords: total,
		Page:         page,
		Limit:        limit,
		Products:     responseProducts,
	}

	util.RespondJSON(c, http.StatusOK, response)
}

func (h *AdminHandler) PatchProductAdmin(c *gin.Context) {
	adminIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	adminID := adminIDRaw.(uint)

	productIDStr := c.Param("id")
	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var req dto.RequestPutProduct
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	var product db.Product
	if err := h.db.First(&product, productID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgProductNotFound)
		return
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
		if req.Visibility != constants.ProductVisibilityAll && req.Visibility != constants.ProductVisibilityOwnerAdmin && req.Visibility != constants.ProductVisibilityAdminOnly {
			util.RespondJSON(c, http.StatusBadRequest, "Visibilitas tidak valid. Harus 'all', 'owner_admin', atau 'admin_only'.")
			return
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
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgNoFieldsToUpdate)
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&product).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update product: %v", err)
		}

		adminLog := db.AdminLog{
			AdminID:    adminID,
			Action:     "patch_product",
			TargetType: "product",
			TargetID:   &product.ID,
			Details:    db.JSONB{"product_title": product.Title, "updates": updates},
			IPAddress:  c.ClientIP(),
		}
		tx.Create(&adminLog)

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessProductUpdated)
}

func (h *AdminHandler) GetTransactions(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	status := c.Query("status")
	userIDStr := c.Query("user_id")
	productIDStr := c.Query("product_id")
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

	var transactions []db.TransactionHistory
	query := h.db.Preload("User").Preload("Product").Order("created_at DESC")

	if status != "" {
		validStatuses := map[string]bool{
			string(constants.TrxStatusPending):      true,
			string(constants.TrxStatusWaitingOwner): true,
			string(constants.TrxStatusWaitingUser):  true,
			string(constants.TrxStatusSuccess):      true,
			string(constants.TrxStatusCancel):       true,
		}
		if !validStatuses[status] {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid transaction status.")
			return
		}
		query = query.Where("status = ?", status)
	}

	if userIDStr != "" {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid user ID.")
			return
		}
		query = query.Where("user_id = ?", userID)
	}

	if productIDStr != "" {
		productID, err := strconv.ParseUint(productIDStr, 10, 64)
		if err != nil {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid product ID.")
			return
		}
		query = query.Where("product_id = ?", productID)
	}

	if receiptStatus != "" {
		validReceiptStatuses := map[string]bool{
			string(constants.ReceiptPendingProcess): true,
			string(constants.ReceiptCompleted):      true,
			string(constants.ReceiptCanceled):       true,
		}
		if !validReceiptStatuses[strings.ToUpper(receiptStatus)] {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid receipt status.")
			return
		}
		query = query.Where("receipt_status = ?", strings.ToUpper(receiptStatus))
	}

	var total int64
	query.Model(&db.TransactionHistory{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&transactions).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	transactionResponses := make([]dto.TransactionDetailResponse, len(transactions))
	for i, trx := range transactions {
		transactionResponses[i] = dto.TransactionDetailResponse{
			ID:            trx.ID,
			ProductID:     trx.ProductID,
			UserID:        trx.UserID,
			Quantity:      trx.Quantity,
			TotalPrice:    trx.TotalPrice,
			GovtTax:       trx.GovtTax,
			EcommerceTax:  trx.EcommerceTax,
			Status:        trx.Status,
			IsSolved:      trx.IsSolved,
			ReceiptStatus: trx.ReceiptStatus,
			CreatedAt:     trx.CreatedAt,
			UpdatedAt:     trx.UpdatedAt,
			Product: dto.ProductDetailResponse{
				ID:         trx.Product.ID,
				Title:      trx.Product.Title,
				Price:      trx.Product.Price,
				Stock:      trx.Product.Stock,
				Visibility: trx.Product.Visibility,
				Categories: strings.Split(strings.TrimSpace(trx.Product.Categories), ","),
				IsActive:   trx.Product.IsActive,
				UserID:     trx.Product.UserID,
				CreatedAt:  trx.Product.CreatedAt,
				UpdatedAt:  trx.Product.UpdatedAt,
				DeletedAt:  &trx.Product.DeletedAt.Time,
			},
			User: dto.UserDetailResponse{
				ID:        trx.User.ID,
				FullName:  trx.User.FullName,
				Email:     trx.User.Email,
				Role:      trx.User.Role,
				Balance:   trx.User.Balance,
				Status:    trx.User.Status,
				CreatedAt: trx.User.CreatedAt,
				UpdatedAt: trx.User.UpdatedAt,
			},
		}
		if !trx.Product.DeletedAt.Valid {
			transactionResponses[i].Product.DeletedAt = nil
		}
		if !trx.User.DeletedAt.Valid {
			transactionResponses[i].User.DeletedAt = nil
		}
	}

	response := dto.GetTransactionsResponse{
		TotalRecords: total,
		Page:         page,
		Limit:        limit,
		Transactions: transactionResponses,
	}

	util.RespondJSON(c, http.StatusOK, response)
}

func (h *AdminHandler) PatchTransactionStatus(c *gin.Context) {
	adminIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	adminID := adminIDRaw.(uint)

	transactionIDStr := c.Param("id")
	transactionID, err := strconv.ParseUint(transactionIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var req dto.RequestPatchTransactionStatus
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	var trx db.TransactionHistory
	if err := h.db.Preload("Product").Preload("User").First(&trx, transactionID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgTransactionNotFound)
		return
	}

	validStatuses := map[string]bool{
		string(constants.TrxStatusPending):      true,
		string(constants.TrxStatusWaitingOwner): true,
		string(constants.TrxStatusWaitingUser):  true,
		string(constants.TrxStatusSuccess):      true,
		string(constants.TrxStatusCancel):       true,
	}
	if !validStatuses[req.Status] {
		util.RespondJSON(c, http.StatusBadRequest, "Status tidak valid. Harus 'pending', 'waiting_owner', 'waiting_users', 'success', atau 'cancel'.")
		return
	}

	oldStatus := trx.Status

	err = h.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status": req.Status,
		}

		switch constants.TransactionStatus(req.Status) {
		case constants.TrxStatusSuccess:
			updates["is_solved"] = true
			updates["receipt_status"] = constants.ReceiptCompleted

			if oldStatus != constants.TrxStatusSuccess {
				var owner db.User
				if err := tx.First(&owner, trx.Product.UserID).Error; err != nil {
					return fmt.Errorf("failed to find product owner: %v", err)
				}
				sellerReceiveAmount := trx.TotalPrice - trx.EcommerceTax
				newOwnerBalance := owner.Balance + sellerReceiveAmount
				if err := tx.Model(&owner).Update("balance", newOwnerBalance).Error; err != nil {
					return fmt.Errorf("failed to update seller balance: %v", err)
				}
				tx.Create(&db.BalanceHistory{
					UserID:       owner.ID,
					Description:  fmt.Sprintf("Pembayaran penjualan produk '%s' (ID: %d) oleh admin", trx.Product.Title, trx.ProductID),
					Amount:       int(sellerReceiveAmount),
					LastBalance:  owner.Balance,
					FinalBalance: newOwnerBalance,
					Status:       constants.BalanceStatusCredit,
				})
				notificationSeller := db.Notification{
					UserID:    owner.ID,
					Type:      constants.NotifTypeSale,
					Message:   fmt.Sprintf("Transaksi produk '%s' (ID: %d) berhasil diselesaikan oleh admin. Dana telah ditransfer ke saldo Anda.", trx.Product.Title, trx.ProductID),
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
			notificationBuyer := db.Notification{
				UserID:    trx.UserID,
				Type:      constants.NotifTypePurchase,
				Message:   fmt.Sprintf("Transaksi produk '%s' (ID: %d) Anda berhasil diselesaikan oleh admin.", trx.Product.Title, trx.ProductID),
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

		case constants.TrxStatusCancel:
			updates["is_solved"] = true
			updates["receipt_status"] = constants.ReceiptCanceled

			if oldStatus != constants.TrxStatusCancel {
				var buyer db.User
				if err := tx.First(&buyer, trx.UserID).Error; err != nil {
					return fmt.Errorf("failed to find buyer: %v", err)
				}
				refundAmount := trx.TotalPrice + trx.GovtTax
				newBalance := buyer.Balance + refundAmount
				if err := tx.Model(&buyer).Update("balance", newBalance).Error; err != nil {
					return fmt.Errorf("failed to refund buyer: %v", err)
				}
				tx.Create(&db.BalanceHistory{
					UserID:       buyer.ID,
					Description:  fmt.Sprintf("Refund dari pembatalan produk '%s' (ID: %d) oleh admin", trx.Product.Title, trx.ProductID),
					Amount:       int(refundAmount),
					LastBalance:  buyer.Balance,
					FinalBalance: newBalance,
					Status:       constants.BalanceStatusRefund,
				})
				notificationBuyer := db.Notification{
					UserID:    buyer.ID,
					Type:      constants.NotifTypePurchase,
					Message:   fmt.Sprintf("Transaksi produk '%s' Anda (ID: %d) dibatalkan oleh admin. Dana telah dikembalikan.", trx.Product.Title, trx.ProductID),
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

			if oldStatus == constants.TrxStatusWaitingUser || oldStatus == constants.TrxStatusSuccess {
				var owner db.User
				if err := tx.First(&owner, trx.Product.UserID).Error; err != nil {
					return fmt.Errorf("failed to find product owner for debit: %v", err)
				}
				debitAmount := trx.TotalPrice - trx.EcommerceTax
				newOwnerBalance := owner.Balance - debitAmount
				if newOwnerBalance < 0 {
					newOwnerBalance = 0
				}
				if err := tx.Model(&owner).Update("balance", newOwnerBalance).Error; err != nil {
					return fmt.Errorf("failed to debit seller balance: %v", err)
				}
				tx.Create(&db.BalanceHistory{
					UserID:       owner.ID,
					Description:  fmt.Sprintf("Debit dari pembatalan paksa produk '%s' (ID: %d) oleh admin", trx.Product.Title, trx.ProductID),
					Amount:       -int(debitAmount),
					LastBalance:  owner.Balance,
					FinalBalance: newOwnerBalance,
					Status:       constants.BalanceStatusDebit,
				})
				notificationSeller := db.Notification{
					UserID:    owner.ID,
					Type:      constants.NotifTypeSale,
					Message:   fmt.Sprintf("Transaksi produk '%s' (ID: %d) Anda dibatalkan paksa oleh admin. Saldo Anda dikurangi Rp%d.", trx.Product.Title, trx.ProductID, debitAmount),
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

		default:
			updates["is_solved"] = false
			updates["receipt_status"] = constants.ReceiptPendingProcess
		}

		if err := tx.Model(&trx).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update transaction status: %v", err)
		}

		adminLog := db.AdminLog{
			AdminID:    adminID,
			Action:     "patch_transaction_status",
			TargetType: "transaction",
			TargetID:   &trx.ID,
			Details:    db.JSONB{"old_status": oldStatus, "new_status": req.Status, "reason": req.Reason, "product_title": trx.Product.Title, "buyer_email": trx.User.Email},
			IPAddress:  c.ClientIP(),
		}
		tx.Create(&adminLog)

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessTransactionConfirmed)
}

func (h *AdminHandler) GetBalanceHistories(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	userIDStr := c.Query("user_id")
	status := c.Query("status")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var histories []db.BalanceHistory
	query := h.db.Preload("User").Order("created_at DESC")

	if userIDStr != "" {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid user ID.")
			return
		}
		query = query.Where("user_id = ?", userID)
	}

	if status != "" {
		if status != string(constants.BalanceStatusCredit) && status != string(constants.BalanceStatusDebit) && status != string(constants.BalanceStatusRefund) {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid status. Must be 'credit', 'debit', or 'refund'.")
			return
		}
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&db.BalanceHistory{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&histories).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	historyResponses := make([]dto.BalanceHistoryDetailResponse, len(histories))
	for i, h := range histories {
		historyResponses[i] = dto.BalanceHistoryDetailResponse{
			ID:           h.ID,
			UserID:       h.UserID,
			Description:  h.Description,
			Amount:       h.Amount,
			LastBalance:  h.LastBalance,
			FinalBalance: h.FinalBalance,
			Status:       h.Status,
			CreatedAt:    h.CreatedAt,
			UpdatedAt:    h.UpdatedAt,
			User: dto.UserDetailResponse{
				ID:        h.User.ID,
				FullName:  h.User.FullName,
				Email:     h.User.Email,
				Role:      h.User.Role,
				Balance:   h.User.Balance,
				Status:    h.User.Status,
				CreatedAt: h.User.CreatedAt,
				UpdatedAt: h.User.UpdatedAt,
			},
		}
		if !h.User.DeletedAt.Valid {
			historyResponses[i].User.DeletedAt = nil
		}
	}

	response := dto.GetBalanceHistoriesResponse{
		TotalRecords: total,
		Page:         page,
		Limit:        limit,
		Histories:    historyResponses,
	}

	util.RespondJSON(c, http.StatusOK, response)
}

func (h *AdminHandler) GetTopUpWithdrawLogs(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	userIDStr := c.Query("user_id")
	actionType := c.Query("type")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var histories []db.BalanceHistory
	query := h.db.Preload("User").Order("created_at DESC")

	if userIDStr != "" {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid user ID.")
			return
		}
		query = query.Where("user_id = ?", userID)
	}

	if actionType != "" {
		if actionType == "topup" {
			query = query.Where("status = ?", constants.BalanceStatusCredit)
		} else if actionType == "withdraw" {
			query = query.Where("status = ?", constants.BalanceStatusDebit)
		} else {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid action type. Must be 'topup' or 'withdraw'.")
			return
		}
	} else {
		query = query.Where("status IN (?, ?)", constants.BalanceStatusCredit, constants.BalanceStatusDebit)
	}

	var total int64
	query.Model(&db.BalanceHistory{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&histories).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	historyResponses := make([]dto.BalanceHistoryDetailResponse, len(histories))
	for i, h := range histories {
		historyResponses[i] = dto.BalanceHistoryDetailResponse{
			ID:           h.ID,
			UserID:       h.UserID,
			Description:  h.Description,
			Amount:       h.Amount,
			LastBalance:  h.LastBalance,
			FinalBalance: h.FinalBalance,
			Status:       h.Status,
			CreatedAt:    h.CreatedAt,
			UpdatedAt:    h.UpdatedAt,
			User: dto.UserDetailResponse{
				ID:        h.User.ID,
				FullName:  h.User.FullName,
				Email:     h.User.Email,
				Role:      h.User.Role,
				Balance:   h.User.Balance,
				Status:    h.User.Status,
				CreatedAt: h.User.CreatedAt,
				UpdatedAt: h.User.UpdatedAt,
			},
		}
		if !h.User.DeletedAt.Valid {
			historyResponses[i].User.DeletedAt = nil
		}
	}

	response := dto.GetTopUpWithdrawLogsResponse{
		TotalRecords: total,
		Page:         page,
		Limit:        limit,
		Logs:         historyResponses,
	}

	util.RespondJSON(c, http.StatusOK, response)
}

func (h *AdminHandler) GetSupportTickets(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	status := c.Query("status")
	search := c.Query("search")
	userIDStr := c.Query("user_id")
	adminIDStr := c.Query("admin_id")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var tickets []db.SupportTicket
	query := h.db.Preload("User").Preload("AssignedAdmin")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if search != "" {
		query = query.Joins("JOIN users ON users.id = support_ticket.user_id").
			Where("LOWER(support_ticket.subject) LIKE ? OR LOWER(users.email) LIKE ?", "%"+strings.ToLower(search)+"%", "%"+strings.ToLower(search)+"%")
	}

	if userIDStr != "" {
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid user ID.")
			return
		}
		query = query.Where("user_id = ?", userID)
	}

	if adminIDStr != "" {
		adminID, err := strconv.ParseUint(adminIDStr, 10, 64)
		if err != nil {
			util.RespondJSON(c, http.StatusBadRequest, "Invalid admin ID.")
			return
		}
		query = query.Where("assigned_admin_id = ?", adminID)
	}

	orderClause := sortBy + " " + sortOrder
	if sortBy == "queue_number" {
		orderClause = "CASE WHEN status = 'open' THEN queue_pos ELSE 999999 END ASC, " + orderClause
	}
	query = query.Order(orderClause)

	var total int64
	query.Model(&db.SupportTicket{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&tickets).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	var responseTickets []dto.SupportTicketResponse
	for _, ticket := range tickets {
		responseTickets = append(responseTickets, dto.SupportTicketResponse{
			ID:                ticket.ID,
			UserID:            ticket.UserID,
			Subject:           ticket.Subject,
			Status:            ticket.Status,
			AssignedAdminID:   ticket.AssignedAdminID,
			AssignedAdminName: ticket.AssignedAdmin.FullName,
			QueueNumber:       ticket.QueuePos,
			CreatedAt:         ticket.CreatedAt,
			UpdatedAt:         ticket.UpdatedAt,
		})
	}

	util.RespondJSON(c, http.StatusOK, dto.GetSupportTicketsResponse{
		TotalRecords: total,
		Page:         page,
		Limit:        limit,
		Tickets:      responseTickets,
	})
}

func (h *AdminHandler) GetSupportTicketMessages(c *gin.Context) {
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var ticket db.SupportTicket
	if err := h.db.First(&ticket, ticketID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgTicketNotFound)
		return
	}

	var messages []db.SupportMessage
	if err := h.db.Preload("Sender").Where("ticket_id = ?", ticketID).Order("created_at ASC").Find(&messages).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	var responseMessages []dto.SupportMessageResponse
	for _, msg := range messages {
		senderName := "Unknown"
		if msg.Sender.ID != 0 {
				senderName = msg.Sender.FullName
		}
		responseMessages = append(responseMessages, dto.SupportMessageResponse{
			ID:          msg.ID,
			TicketID:    msg.TicketID,
			SenderID:    msg.SenderID,
			SenderName:  senderName,
			IsAdmin:     msg.Sender.Role == constants.RoleAdmin,
			Content:     msg.Content,
			MessageType: msg.MessageType,
			FileURL:     msg.FileURL,
			CreatedAt:   msg.CreatedAt,
		})
	}

	util.RespondJSON(c, http.StatusOK, dto.GetSupportTicketMessagesResponse{
		TicketID: ticket.ID,
		Subject:  ticket.Subject,
		Status:   ticket.Status,
		Messages: responseMessages,
	})
}

func (h *AdminHandler) ClaimSupportTicket(c *gin.Context) {
	adminIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	adminID := adminIDRaw.(uint)

	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var ticket db.SupportTicket
	if err := h.db.First(&ticket, ticketID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgTicketNotFound)
		return
	}

	if ticket.Status != constants.TicketStatusOpen && ticket.Status != constants.TicketStatusPendingAdmin {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgTicketNotClaimable)
		return
	}

	if ticket.AssignedAdminID != nil && *ticket.AssignedAdminID != 0 {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgTicketAlreadyClaimed)
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		ticket.Status = constants.TicketStatusPendingUser
		ticket.AssignedAdminID = &adminID
		ticket.QueuePos = 0
		if err := tx.Save(&ticket).Error; err != nil {
			return fmt.Errorf("failed to claim ticket: %v", err)
		}

		var adminUser db.User
		tx.First(&adminUser, adminID)

		notification := db.Notification{
			UserID:    ticket.UserID,
			Type:      constants.NotifTypeSupport,
			Message:   fmt.Sprintf("Tiket dukungan Anda #%d ('%s') telah diklaim oleh admin %s. Mohon tunggu balasan dari kami.", ticket.ID, ticket.Subject, adminUser.FullName),
			RelatedID: &ticket.ID,
		}
		util.SendNotificationToUser(ticket.UserID, dto.NotificationResponse{
			ID:        notification.ID,
			Type:      notification.Type,
			Message:   notification.Message,
			RelatedID: notification.RelatedID,
			CreatedAt: notification.CreatedAt,
			IsRead:    notification.IsRead,
		})
		tx.Create(&notification)

		adminLog := db.AdminLog{
			AdminID:    adminID,
			Action:     "claim_support_ticket",
			TargetType: "support_ticket",
			TargetID:   &ticket.ID,
			Details:    db.JSONB{"ticket_subject": ticket.Subject, "claimed_by": adminUser.FullName},
			IPAddress:  c.ClientIP(),
		}
		tx.Create(&adminLog)

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessTicketClaimed)
}

func (h *AdminHandler) ReplySupportTicket(c *gin.Context) {
	adminIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	adminID := adminIDRaw.(uint)

	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var req dto.RequestReplyTicket
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	var ticket db.SupportTicket
	if err := h.db.First(&ticket, ticketID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgTicketNotFound)
		return
	}

	if ticket.AssignedAdminID == nil || *ticket.AssignedAdminID != adminID {
		util.RespondJSON(c, http.StatusForbidden, "Anda tidak ditugaskan untuk tiket ini atau tiket belum diklaim.")
		return
	}

	if ticket.Status == constants.TicketStatusClosed || ticket.Status == constants.TicketStatusCanceled {
		util.RespondJSON(c, http.StatusBadRequest, "Tidak dapat membalas tiket yang sudah ditutup atau dibatalkan.")
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		message := db.SupportMessage{
			TicketID:    ticket.ID,
			SenderID:    adminID,
			MessageType: req.MessageType,
			Content:     req.Content,
			FileURL:     req.FileURL,
		}
		if err := tx.Create(&message).Error; err != nil {
			return fmt.Errorf("failed to create support message: %v", err)
		}

		ticket.Status = constants.TicketStatusPendingUser
		if err := tx.Save(&ticket).Error; err != nil {
			return fmt.Errorf("failed to update ticket status: %v", err)
		}

		notification := db.Notification{
			UserID:    ticket.UserID,
			Type:      constants.NotifTypeSupport,
			Message:   fmt.Sprintf("Ada balasan baru untuk tiket dukungan Anda #%d ('%s').", ticket.ID, ticket.Subject),
			RelatedID: &ticket.ID,
		}
		util.SendNotificationToUser(ticket.UserID, dto.NotificationResponse{
			ID:        notification.ID,
			Type:      notification.Type,
			Message:   notification.Message,
			RelatedID: notification.RelatedID,
			CreatedAt: notification.CreatedAt,
			IsRead:    notification.IsRead,
		})
		tx.Create(&notification)

		adminLog := db.AdminLog{
			AdminID:    adminID,
			Action:     "reply_support_ticket",
			TargetType: "support_ticket",
			TargetID:   &ticket.ID,
			Details:    db.JSONB{"ticket_subject": ticket.Subject, "message_content": req.Content, "message_type": req.MessageType},
			IPAddress:  c.ClientIP(),
		}
		tx.Create(&adminLog)

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessMessageSent)
}

func handleProductTransactionsOnDelete(tx *gorm.DB, product db.Product) error {
	var pendingTransactions []db.TransactionHistory
	if err := tx.Where("product_id = ? AND status IN (?, ?)", product.ID, constants.TrxStatusPending, constants.TrxStatusWaitingOwner).Find(&pendingTransactions).Error; err != nil {
		return fmt.Errorf("failed to find pending transactions for product %d: %v", product.ID, err)
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
			Description:  fmt.Sprintf("Refund dari pembatalan produk '%s' (ID: %d) karena penjual dihapus/diblokir", product.Title, product.ID),
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
			Message:   fmt.Sprintf("Pembelian produk '%s' Anda (ID: %d) dibatalkan karena penjual dihapus/diblokir. Dana telah dikembalikan.", product.Title, trx.ID),
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
		return fmt.Errorf("failed to find waiting_users transactions for product %d: %v", product.ID, err)
	}
	for _, trx := range waitingUserTransactions {
		sellerReceiveAmount := trx.TotalPrice - trx.EcommerceTax
		tx.Create(&db.BalanceHistory{
			UserID:       product.UserID,
			Description:  fmt.Sprintf("Pembayaran penjualan produk '%s' (ID: %d) karena penjual dihapus/diblokir", product.Title, product.ID),
			Amount:       int(sellerReceiveAmount),
			LastBalance:  0,
			FinalBalance: 0,
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
			Message:   fmt.Sprintf("Pembelian produk '%s' Anda (ID: %d) telah diselesaikan karena penjual dihapus/diblokir. Penjual telah dibayar.", product.Title, trx.ID),
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
	}
	return nil
}