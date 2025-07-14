package dto

type UploadFileResponse struct {
	Message  string `json:"message"`
	FileURL  string `json:"file_url"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}