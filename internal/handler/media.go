package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/db"
	"portolio-backend/internal/util"
)

type MediaHandler struct {
	db *gorm.DB
}

func NewMediaHandler(db *gorm.DB) *MediaHandler {
	return &MediaHandler{db: db}
}

// ServeProtectedMedia menangani penyajian file dari direktori media terproteksi (chat, support)
func (h *MediaHandler) ServeProtectedMedia(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	mediaType := c.Param("mediaType") // e.g., "chat", "support"
	filename := c.Param("filename")

	// Validasi dasar untuk mediaType
	if mediaType != "chat" && mediaType != "support" {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgNotFound)
		return
	}

	fullPath := filepath.Join("./media", mediaType, filename)

	// Periksa apakah file ada di disk
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgNotFound)
		return
	}

	// Logika otorisasi
	isAuthorized := false
	var err error

	if mediaType == "chat" {
		// Periksa apakah pengguna adalah pengirim atau penerima pesan chat dengan file ini
		var chatMessage db.ChatMessage
		// Pastikan FileURL di DB disimpan dengan format /media/chat/namafile.ext
		err = h.db.Where("file_url = ? AND (sender_id = ? OR receiver_id = ?)", "/media/chat/"+filename, userID, userID).First(&chatMessage).Error
		if err == nil {
			isAuthorized = true
		} else if err != gorm.ErrRecordNotFound {
			util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
			return
		}
	} else if mediaType == "support" {
		// Periksa apakah pengguna adalah pembuat tiket atau admin yang ditugaskan untuk pesan support dengan file ini
		var supportMessage db.SupportMessage
		// Pastikan FileURL di DB disimpan dengan format /media/support/namafile.ext
		err = h.db.
			Joins("JOIN support_ticket ON support_ticket.id = support_message.ticket_id").
			Where("support_message.file_url = ? AND (support_ticket.user_id = ? OR support_ticket.assigned_admin_id = ?)", "/media/support/"+filename, userID, userID).
			First(&supportMessage).Error
		if err == nil {
			isAuthorized = true
		} else if err != gorm.ErrRecordNotFound {
			util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
			return
		}
	}

	if !isAuthorized {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
		return
	}

	// Sajikan file
	c.File(fullPath)
}