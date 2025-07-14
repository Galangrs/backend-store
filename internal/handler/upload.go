package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"portolio-backend/configs/constants"
	"portolio-backend/internal/util"
	"portolio-backend/internal/model/dto"
)

type UploadHandler struct {
	db *gorm.DB
}

func NewUploadHandler(db *gorm.DB) *UploadHandler {
	return &UploadHandler{db: db}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, "Gambar tidak ditemukan dalam request.")
		return
	}

	uploadType := c.DefaultQuery("type", "general")
	destDir := "media/temp"

	switch strings.ToLower(uploadType) {
	case "chat":
		destDir = "media/chat"
	case "support":
		destDir = "media/support"
	case "product":
		destDir = "media/products"
	default:
		destDir = "media/general"
	}

	filePath, err := util.SaveUploadedFile(file, destDir, constants.MaxImageSizeMB*1024*1024, util.AllowedImageExtensions)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	fileURL := "/" + filepath.ToSlash(filePath)

	util.RespondJSON(c, http.StatusOK, dto.UploadFileResponse{ // Changed to DTO
		Message:  constants.MsgSuccessFileUpload,
		FileURL: fileURL,
		FileName: file.Filename,
		FileSize: file.Size,
	})
}

func (h *UploadHandler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, "File tidak ditemukan dalam request.")
		return
	}

	uploadType := c.DefaultQuery("type", "general")
	destDir := "media/temp"

	switch strings.ToLower(uploadType) {
	case "chat":
		destDir = "media/chat"
	case "support":
		destDir = "media/support"
	case "product":
		destDir = "media/products"
	default:
		destDir = "media/general"
	}

	filePath, err := util.SaveUploadedFile(file, destDir, constants.MaxDocumentSizeMB*1024*1024, util.AllowedDocumentExtensions)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	fileURL := "/" + filepath.ToSlash(filePath)

	util.RespondJSON(c, http.StatusOK, dto.UploadFileResponse{
		Message:  constants.MsgSuccessFileUpload,
		FileURL: fileURL,
		FileName: file.Filename,
		FileSize: file.Size,
	})
}