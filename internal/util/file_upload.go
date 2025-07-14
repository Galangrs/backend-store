package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"portolio-backend/configs/constants"
)

func GenerateRandomFilename() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "file"
	}
	return hex.EncodeToString(b)
}

func SaveUploadedFile(file *multipart.FileHeader, destDir string, maxSizeBytes int64, allowedExtensions []string) (string, error) {
	if file.Size > maxSizeBytes {
		return "", fmt.Errorf(constants.ErrMsgFileTooLarge)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	isValidExt := false
	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		return "", fmt.Errorf(constants.ErrMsgInvalidFileType + ": " + ext + " tidak diizinkan.")
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return "", err
	}

	filename := GenerateRandomFilename() + ext
	filePath := filepath.Join(destDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return filePath, nil
}

func DownloadImage(url, destDir string, maxSizeBytes int64) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gagal mengunduh gambar: %s", resp.Status)
	}

	if resp.ContentLength > 0 && resp.ContentLength > maxSizeBytes {
		return "", fmt.Errorf(constants.ErrMsgFileTooLarge)
	}

	ext := strings.ToLower(filepath.Ext(strings.Split(url, "?")[0]))
	if ext == "" {
		contentType := resp.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "image/") {
			ext = "." + strings.Split(contentType, "/")[1]
		} else {
			return "", fmt.Errorf(constants.ErrMsgInvalidFileType + ": tidak dapat menentukan tipe gambar dari URL atau Content-Type")
		}
	}

	isValidExt := false
	for _, allowedExt := range AllowedImageExtensions {
		if ext == allowedExt {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		return "", fmt.Errorf(constants.ErrMsgInvalidFileType + ": " + ext + " bukan tipe gambar yang diizinkan.")
	}

	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return "", err
	}

	filename := GenerateRandomFilename() + ext
	filePath := filepath.Join(destDir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	n, err := io.Copy(out, io.LimitReader(resp.Body, maxSizeBytes+1))
	if err != nil {
		return "", err
	}
	if n > maxSizeBytes {
		os.Remove(filePath)
		return "", fmt.Errorf(constants.ErrMsgFileTooLarge)
	}

	return filePath, nil
}

var GetMaxImageSizeMB = func() int {
	return constants.MaxImageSizeMB
}

var GetMaxDocumentSizeMB = func() int {
	return constants.MaxDocumentSizeMB
}

var AllowedImageExtensions = []string{".jpg", ".jpeg", ".png", ".gif"}

var AllowedDocumentExtensions = []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".csv"}