package middlewares

import (
	"net/http"
	"strings"
	"time"
    "crypto/sha256"
    "encoding/hex"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"portolio-backend/configs"
	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/db"
	"portolio-backend/internal/util"
)

func AdminAuth(dbConn *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
			return
		}

		token := strings.TrimPrefix(tokenString, "Bearer ")

		userIDRaw, exists := c.Get("ID")
		if !exists {
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
			return
		}
		userID := userIDRaw.(uint)

		var adminSession db.AdminSession
		if err := dbConn.Where("user_id = ?", userID).First(&adminSession).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgAdminSessionInvalid)
			} else {
				util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
			}
			return
		}

		if !compareSessionToken(adminSession.TokenHash, token) {
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgAdminSessionExpired)
			return
		}

		if time.Now().After(adminSession.ExpiresAt) {
			dbConn.Delete(&adminSession)
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgAdminSessionExpired)
			return
		}

		var user db.User
		if err := dbConn.First(&user, adminSession.UserID).Error; err != nil {
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
			return
		}

		if user.Role != constants.RoleAdmin || user.Status != constants.UserStatusActive || user.DeletedAt.Valid {
			dbConn.Delete(&adminSession)
			util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
			return
		}

		tokenJWTConfig := configs.GetTokenJWTConfig()
		sessionToken, err := util.GenerateAdminSessionToken(user.ID, string(user.Role), tokenJWTConfig.AdminExpireDuration, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
			return
		}
	
		hashed := sha256.Sum256([]byte(sessionToken))
		newTokenHash := hex.EncodeToString(hashed[:])

		adminSession.TokenHash = newTokenHash
		adminSession.ExpiresAt = time.Now().Add(tokenJWTConfig.AdminExpireDuration)
		adminSession.LastUsedAt = time.Now()
		adminSession.IPAddress = c.ClientIP()
		adminSession.UserAgent = c.Request.UserAgent()

		if err := dbConn.Save(&adminSession).Error; err != nil {
			util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
			return
		}

		c.Header("Authorization", "Bearer "+sessionToken)

		// Set ke context
		c.Set("ID", user.ID)
		c.Set("ROLE", user.Role)
		c.Next()
	}
}

func AuthorizeAdmin(dbConn *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userROLE, ok := c.Get("ROLE")
		if !ok {
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
			return
		}

		if userROLE != constants.RoleAdmin {
			util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
			return
		}

		c.Next()
	}
}

func compareSessionToken(storedHash, plainToken string) bool {
	hashed := sha256.Sum256([]byte(plainToken))
	return hex.EncodeToString(hashed[:]) == storedHash
}