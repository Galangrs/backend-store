package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/db"
	"portolio-backend/internal/model/dto"
	"portolio-backend/internal/util"
)

type AccountHandler struct {
	db *gorm.DB
}

func NewAccountHandler(db *gorm.DB) *AccountHandler {
	return &AccountHandler{db: db}
}

func validatePassword(password string) bool {
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSymbol := regexp.MustCompile(`[\W_]`).MatchString(password)
	isLongEnough := len(password) >= 8
	return hasNumber && hasSymbol && isLongEnough
}

func validateEmail(email string) bool {
	var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return emailRegex.MatchString(email)
}

func validateFullName(name string) bool {
	var nameRegex = regexp.MustCompile(`^[a-zA-Z\s]+$`)
	return nameRegex.MatchString(name)
}

func (h *AccountHandler) PostRegisterRequest(c *gin.Context) {
	var request dto.RequestPostRegister

	if err := c.ShouldBindJSON(&request); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	email := strings.TrimSpace(strings.ToLower(request.Email))
	if !validateEmail(email) {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgInvalidEmailFormat)
		return
	}

	if !validateFullName(request.FullName) {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgInvalidFullNameFormat)
		return
	}

	if !validatePassword(request.Password) {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgPasswordWeak)
		return
	}

	validate := validator.New()
	if err := validate.Struct(request); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	var existing db.User
	if err := h.db.Where("email = ?", email).First(&existing).Error; err == nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgEmailAlreadyRegistered)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	user := db.User{
		FullName:        request.FullName,
		Email:           email,
		Password:        string(hashedPassword),
		Role:            constants.RoleUser,
		Balance:         0,
		Status:          constants.UserStatusActive,
		PenaltyWarnings: 0,
	}

	if err := h.db.Create(&user).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	responseUser := dto.PostRegisterResponse{
		Fullname:  user.FullName,
		Email:     user.Email,
		Balance:   user.Balance,
		CreatedAt: user.CreatedAt,
	}

	util.RespondJSON(c, http.StatusCreated, responseUser)
}

func (h *AccountHandler) PostLoginRequest(c *gin.Context) {
	var request dto.RequestPostLogin

	if err := c.ShouldBindJSON(&request); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	email := strings.TrimSpace(strings.ToLower(request.Email))

	var user db.User
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgInvalidCredentials)
		return
	}

	if user.Status == constants.UserStatusSuspended {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgAccountSuspended)
		return
	}
	if user.Status == constants.UserStatusBanned {
		if user.BanUntil != nil && time.Now().Unix() > *user.BanUntil {
			user.Status = constants.UserStatusActive
			user.BanUntil = nil
			user.BanReason = ""
			user.PenaltyWarnings = 0
			h.db.Save(&user)

			notification := dto.NotificationResponse{
				UserID:    user.ID,
				Type:      constants.NotifTypeAccount,
				Message:   "Masa pemblokiran akun Anda telah berakhir. Akun Anda sekarang aktif kembali.",
				RelatedID: &user.ID,
				CreatedAt: time.Now(),
			}
			util.SendNotificationToUser(user.ID, notification)
			h.db.Create(&db.Notification{
				UserID:    notification.UserID,
				Type:      notification.Type,
				Message:   notification.Message,
				RelatedID: notification.RelatedID,
			})

		} else {
			util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgAccountBanned)
			return
		}
	}
	if user.DeletedAt.Valid {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgAccountDeleted)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)); err != nil {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgInvalidCredentials)
		return
	}

	token, err := util.GenerateJWTToken(user.ID, string(user.Role))
	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	c.Header("Authorization", "Bearer "+token)

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessLogin)
}

func (h *AccountHandler) GetBalanceRequest(c *gin.Context) {
	idVal, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}

	userID, ok := idVal.(uint)
	if !ok {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}

	limit := 50
	if fieldStr := c.Query("limit"); fieldStr != "" {
		if parsed, err := strconv.Atoi(fieldStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	var user db.User
	if err := h.db.First(&user, userID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgUserNotFound)
		return
	}

	var histories []db.BalanceHistory
	if err := h.db.Where("user_id = ?", userID).Order("created_at desc").Limit(limit).Find(&histories).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	historiesResp := make([]dto.BalanceHistoryResponse, 0, len(histories))
	for _, h := range histories {
		historiesResp = append(historiesResp, dto.BalanceHistoryResponse{
			ID:           h.ID,
			CreatedAt:    h.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    h.UpdatedAt.Format("2006-01-02 15:04:05"),
			Description:  h.Description,
			Amount:       h.Amount,
			LastBalance:  h.LastBalance,
			FinalBalance: h.FinalBalance,
			Status:       string(h.Status),
		})
	}

	response := dto.GetBalanceResponse{
		Balance:   user.Balance,
		Histories: historiesResp,
	}

	util.RespondJSON(c, http.StatusOK, response)
}

func (h *AccountHandler) PostTopUpBalance(c *gin.Context) {
	userIDInterface, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}

	userID := userIDInterface.(uint)

	var req dto.RequestTopUp
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	if req.Amount <= 0 {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgAmountInvalid)
		return
	}

	var responseData dto.PostTopUpResponse

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var user db.User
		if err := tx.First(&user, userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf(constants.ErrMsgUserNotFound)
			}
			return fmt.Errorf(constants.ErrMsgInternalServerError)
		}

		lastBalance := user.Balance
		finalBalance := lastBalance + req.Amount

		user.Balance = finalBalance
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to update balance: %v", err)
		}

		balanceHistory := db.BalanceHistory{
			UserID:       user.ID,
			Description:  fmt.Sprintf("Top Up saldo sebesar %d", req.Amount),
			Amount:       int(req.Amount),
			LastBalance:  lastBalance,
			FinalBalance: finalBalance,
			Status:       constants.BalanceStatusCredit,
		}

		if err := tx.Create(&balanceHistory).Error; err != nil {
			return fmt.Errorf("failed to create balance history: %v", err)
		}

		notification := db.Notification{
			UserID:    user.ID,
			Type:      constants.NotifTypeTopUp,
			Message:   fmt.Sprintf("Top up sebesar Rp%d berhasil! Saldo Anda sekarang Rp%d.", req.Amount, finalBalance),
			RelatedID: &balanceHistory.ID,
		}
		if err := tx.Create(&notification).Error; err != nil {
			return fmt.Errorf("failed to create notification: %v", err)
		}
		util.SendNotificationToUser(user.ID, dto.NotificationResponse{
			ID:        notification.ID,
			Type:      notification.Type,
			Message:   notification.Message,
			RelatedID: notification.RelatedID,
			CreatedAt: notification.CreatedAt,
			IsRead:    notification.IsRead,
		})

		balanceResp := dto.BalanceHistoryResponse{
			ID:           balanceHistory.ID,
			CreatedAt:    balanceHistory.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    balanceHistory.UpdatedAt.Format("2006-01-02 15:04:05"),
			Description:  balanceHistory.Description,
			Amount:       balanceHistory.Amount,
			LastBalance:  balanceHistory.LastBalance,
			FinalBalance: balanceHistory.FinalBalance,
			Status:       string(balanceHistory.Status),
		}

		responseData = dto.PostTopUpResponse{
			Message:       constants.MsgSuccessTopUp,
			FullName:      user.FullName,
			NewBalance:    finalBalance,
			BalanceEntry: balanceResp,
		}

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, responseData)
}

func (h *AccountHandler) PostWithDrawBalance(c *gin.Context) {
	userIDInterface, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}

	userID := userIDInterface.(uint)

	var req dto.RequestWithdraw
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	if req.Amount < 10000 {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgMinWithdrawAmount)
		return
	}

	var responseData dto.PostWithdrawResponse

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var user db.User
		if err := tx.First(&user, userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf(constants.ErrMsgUserNotFound)
			}
			return fmt.Errorf(constants.ErrMsgInternalServerError)
		}

		if user.Balance < req.Amount {
			return fmt.Errorf(constants.ErrMsgInsufficientBalance)
		}

		lastBalance := user.Balance
		finalBalance := lastBalance - req.Amount

		user.Balance = finalBalance
		if err := tx.Save(&user).Error; err != nil {
			return fmt.Errorf("failed to update balance: %v", err)
		}

		balanceHistory := db.BalanceHistory{
			UserID:       user.ID,
			Description:  fmt.Sprintf("Withdraw saldo sebesar %d", req.Amount),
			Amount:       -int(req.Amount),
			LastBalance:  lastBalance,
			FinalBalance: finalBalance,
			Status:       constants.BalanceStatusDebit,
		}

		if err := tx.Create(&balanceHistory).Error; err != nil {
			return fmt.Errorf("failed to create balance history: %v", err)
		}

		notification := db.Notification{
			UserID:    user.ID,
			Type:      constants.NotifTypeWithdraw,
			Message:   fmt.Sprintf("Penarikan dana sebesar Rp%d berhasil! Saldo Anda sekarang Rp%d.", req.Amount, finalBalance),
			RelatedID: &balanceHistory.ID,
		}
		if err := tx.Create(&notification).Error; err != nil {
			return fmt.Errorf("failed to create notification: %v", err)
		}
		util.SendNotificationToUser(user.ID, dto.NotificationResponse{
			ID:        notification.ID,
			Type:      notification.Type,
			Message:   notification.Message,
			RelatedID: notification.RelatedID,
			CreatedAt: notification.CreatedAt,
			IsRead:    notification.IsRead,
		})

		balanceResp := dto.BalanceHistoryResponse{
			ID:           balanceHistory.ID,
			CreatedAt:    balanceHistory.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    balanceHistory.UpdatedAt.Format("2006-01-02 15:04:05"),
			Description:  balanceHistory.Description,
			Amount:       balanceHistory.Amount,
			LastBalance:  balanceHistory.LastBalance,
			FinalBalance: balanceHistory.FinalBalance,
			Status:       string(balanceHistory.Status),
		}

		responseData = dto.PostWithdrawResponse{
			Message:       constants.MsgSuccessWithdraw,
			FullName:      user.FullName,
			NewBalance:    finalBalance,
			BalanceEntry: balanceResp,
		}

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, responseData)
}

func (h *AccountHandler) PatchAccount(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	var req dto.RequestPatchAccount
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	var user db.User
	if err := h.db.First(&user, userID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgUserNotFound)
		return
	}

	updates := make(map[string]interface{})

	if req.FullName != "" {
		if !validateFullName(req.FullName) {
			util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgInvalidFullNameFormat)
			return
		}
		updates["full_name"] = req.FullName
	}

	if req.Email != "" {
		newEmail := strings.TrimSpace(strings.ToLower(req.Email))
		if !validateEmail(newEmail) {
			util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgInvalidEmailFormat)
			return
		}
		var existingUser db.User
		if err := h.db.Where("email = ? AND id != ?", newEmail, userID).First(&existingUser).Error; err == nil {
			util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgEmailAlreadyRegistered)
			return
		}
		updates["email"] = newEmail
	}

	if req.NewPassword != "" {
		if req.OldPassword == "" {
			util.RespondJSON(c, http.StatusBadRequest, "Kata sandi lama diperlukan untuk mengubah kata sandi.")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgInvalidOldPassword)
			return
		}
		if !validatePassword(req.NewPassword) {
			util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgPasswordWeak)
			return
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
			return
		}
		updates["password"] = string(hashedPassword)
	}

	if len(updates) == 0 {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgNoFieldsToUpdate)
		return
	}

	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessAccountUpdated)
}