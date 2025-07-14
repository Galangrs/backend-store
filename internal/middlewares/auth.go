package middlewares

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"portolio-backend/configs"
	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/db"
	"portolio-backend/internal/model/dto"
	"portolio-backend/internal/util"
)

const GracePeriod = 7 * 24 * time.Hour

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		var tokenString string

		// Cek apakah ini request WebSocket (Upgrade)
		if strings.ToLower(c.GetHeader("Upgrade")) == "websocket" {
			tokenString = c.Query("token")
		} else {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
				return
			}

			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
				util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
				return
			}

			tokenString = tokenParts[1]
		}

		if tokenString == "" {
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
			return
		}

		tokenJWTConfig := configs.GetTokenJWTConfig()

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(tokenJWTConfig.JWT), nil
		})

		if err != nil {
			if ve, ok := err.(*jwt.ValidationError); ok && ve.Errors == jwt.ValidationErrorExpired {
				claims := jwt.MapClaims{}
				_, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims)
				if err != nil {
					util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
					return
				}

				exp, ok := claims["exp"].(float64)
				if !ok {
					util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
					return
				}

				expiredAt := time.Unix(int64(exp), 0)
				expiredDuration := time.Since(expiredAt)

				if expiredDuration <= GracePeriod {
					userID, ok1 := claims["ID"].(float64)
					role, ok2 := claims["role"].(string)
					if !ok1 || !ok2 {
						util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
						return
					}

					if constants.UserRole(role) == constants.RoleAdmin {
						c.Set("ID", uint(userID))
						c.Set("ROLE", constants.UserRole(role))
						c.Next()
						return
					}

					newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
						"ID":   uint(userID),
						"role": role,
						"exp":  time.Now().Add(tokenJWTConfig.ExpireDuration).Unix(),
						"iat":  time.Now().Unix(),
					})

					newTokenString, err := newToken.SignedString([]byte(tokenJWTConfig.JWT))
					if err != nil {
						util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
						return
					}

					c.Header("Authorization", "Bearer "+newTokenString)
					c.Set("ID", uint(userID))
					c.Set("ROLE", constants.UserRole(role))
					c.Next()
					return
				} else {
					util.RespondJSON(c, http.StatusUnauthorized, "Token kadaluarsa, mohon login kembali.")
					return
				}
			}

			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, ok := claims["ID"].(float64)
			if !ok {
				util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
				return
			}

			role, ok := claims["role"].(string)
			if !ok {
				util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
				return
			}

			c.Set("ID", uint(userID))
			c.Set("ROLE", constants.UserRole(role))
			c.Next()
			return
		}

		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
	}
}

func Authorize(dbConn *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get("ID")
		if !ok {
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
			return
		}

		var user db.User
		resultUser := dbConn.Where("id = ?", userID).First(&user)
		if resultUser.Error != nil {
			util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
			return
		}

		c.Set("ROLE", user.Role)
		c.Next()
	}
}

func CheckRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Set("ROLE", constants.RoleGuest)
			c.Next()
			return
		}

		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
			c.Set("ROLE", constants.RoleGuest)
			c.Next()
			return
		}

		tokenString := tokenParts[1]

		parser := &jwt.Parser{SkipClaimsValidation: true}
		token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			c.Set("ROLE", constants.RoleGuest)
			c.Next()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Set("ROLE", constants.RoleGuest)
			c.Next()
			return
		}

		role, ok := claims["role"].(string)
		if !ok || role == "" {
			role = string(constants.RoleUser)
		}

		c.Set("ROLE", constants.UserRole(role))
		c.Next()
	}
}

func CheckUserStatus(dbConn *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDRaw, exists := c.Get("ID")
		if !exists {
			c.Next()
			return
		}
		userID := userIDRaw.(uint)

		var user db.User
		if err := dbConn.First(&user, userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgUserNotFound)
				return
			}
			util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
			return
		}

		if user.DeletedAt.Valid {
			util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgAccountDeleted)
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
				dbConn.Save(&user)

				notification := dto.NotificationResponse{
					UserID:    user.ID,
					Type:      constants.NotifTypeAccount,
					Message:   "Masa pemblokiran akun Anda telah berakhir. Akun Anda sekarang aktif kembali.",
					RelatedID: &user.ID,
					CreatedAt: time.Now(),
				}
				util.SendNotificationToUser(user.ID, notification)
				dbConn.Create(&db.Notification{
					UserID:    notification.UserID,
					Type:      notification.Type,
					Message:   notification.Message,
					RelatedID: notification.RelatedID,
				})

				c.Next()
				return
			}
			util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgAccountBanned)
			return
		}

		penaltyLimit := configs.GetEnvInt("PENALTY_WARNING_LIMIT", constants.DefaultPenaltyWarningLimit)
		if user.PenaltyWarnings >= uint(penaltyLimit) {
			user.Status = constants.UserStatusSuspended
			user.BanUntil = nil
			user.BanReason = fmt.Sprintf("Akun ditangguhkan otomatis karena mencapai %d peringatan penalti.", penaltyLimit)
			dbConn.Save(&user)

			notification := dto.NotificationResponse{
				UserID:    user.ID,
				Type:      constants.NotifTypeAccount,
				Message:   fmt.Sprintf("Akun Anda telah ditangguhkan secara permanen karena mencapai batas %d peringatan penalti. Mohon hubungi dukungan.", penaltyLimit),
				RelatedID: &user.ID,
				CreatedAt: time.Now(),
			}
			util.SendNotificationToUser(user.ID, notification)
			dbConn.Create(&db.Notification{
				UserID:    notification.UserID,
				Type:      notification.Type,
				Message:   notification.Message,
				RelatedID: notification.RelatedID,
			})

			util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgAccountSuspended)
			return
		}

		c.Next()
	}
}