package util

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/dto"
)

var validate = validator.New()

// FieldErrorResponse untuk detail error per field
type FieldErrorResponse struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// ErrorResponse untuk respons error yang konsisten
type ErrorResponse struct {
	Status         string               `json:"status"`
	Code           int                  `json:"code"`
	Message        string               `json:"message"`
	Fields         []FieldErrorResponse `json:"fields,omitempty"`
	ExampleRequest interface{}          `json:"example_request,omitempty"` // Contoh JSON lengkap
	Timestamp      string               `json:"timestamp"`
}

// SuccessResponse untuk respons sukses yang konsisten
type SuccessResponse struct {
	Status    string      `json:"status"`
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// contoh model yang kamu punya, update sesuai model kamu
// Tambahkan semua request model yang mungkin divalidasi
var modelExamples = map[string]interface{}{
	"RequestPostRegister": dto.RequestPostRegister{
		FullName: "John Doe",
		Email:    "john.doe@example.com",
		Password: "Password123!",
	},
	"RequestPostLogin": dto.RequestPostLogin{
		Email:    "john.doe@example.com",
		Password: "Password123!",
	},
	"RequestTopUp": dto.RequestTopUp{
		Amount: 50000,
	},
	"RequestWithdraw": dto.RequestWithdraw{
		Amount: 25000,
	},
	"RequestPatchAccount": dto.RequestPatchAccount{
		FullName:    "Jane Doe",
		Email:       "jane.doe@example.com",
		OldPassword: "Password123!",
		NewPassword: "NewPassword123!",
	},
	"RequestPostProduct": dto.RequestPostProduct{
		Title:      "Gaming Keyboard RGB",
		Price:      750000,
		Stock:      10,
		Visibility: constants.ProductVisibilityAll,
		Categories: "Electronics,Gaming",
		ImagesLinks: []string{
			"https://example.com/image1.jpg",
			"https://example.com/image2.jpg",
		},
	},
	"RequestPutProduct": dto.RequestPutProduct{ // Asumsi sama dengan PostProduct untuk contoh
		Title:      "Gaming Keyboard Pro",
		Price:      800000,
		Stock:      8,
		Visibility: constants.ProductVisibilityAll,
		Categories: "Electronics,Gaming,Peripherals",
		ImagesLinks: []string{
			"https://example.com/new_image.jpg",
		},
		IsActive: func(b bool) *bool { return &b }(true),
	},
	"RequestPurchaseItem": []dto.RequestPurchaseItem{ // Contoh untuk slice
		{ProductID: 1, Quantity: 1},
		{ProductID: 2, Quantity: 2},
	},
	"RequestCreateTicket": dto.RequestCreateTicket{
		Subject: "Problem with recent purchase",
		Message: "My order #123 has not arrived yet.",
	},
	"RequestSendMessage": dto.RequestSendMessage{
		MessageType: constants.ChatTypeText,
		Content:     "Hello, is this still available?",
		// FileURL: "optional_file_url_if_type_is_image_or_file"
	},
	// --- NEW REQUEST DTOs ---
	"RequestSuspendUser": dto.RequestSuspendUser{
		Reason: "Melanggar kebijakan penggunaan berulang kali.",
	},
	"RequestBanUser": dto.RequestBanUser{
		DurationHours: 24,
		Reason:        "Melakukan penipuan.",
	},
	"RequestPatchTransactionStatus": dto.RequestPatchTransactionStatus{
		Status: "success",
		Reason: "Konfirmasi manual oleh admin.",
	},
	"RequestConfirmTransactionByOwner": dto.RequestConfirmTransactionByOwner{
		TransactionIDs: []uint{1, 2},
	},
	"RequestConfirmTransactionByUser": dto.RequestConfirmTransactionByUser{
		TransactionIDs: []uint{3},
		Reviews: []dto.ReviewItem{
			{
				TransactionID: 3,
				Rating:        func(u uint) *uint { return &u }(5),
				Comment:       "Produk sangat bagus dan sesuai deskripsi!",
			},
		},
	},
	"RequestCancelTransaction": dto.RequestCancelTransaction{
		TransactionIDs: []uint{4},
	},
	"RequestReplyTicket": dto.RequestReplyTicket{
		Content:     "Terima kasih atas informasinya, kami akan segera menindaklanjuti.",
		MessageType: constants.ChatTypeText,
	},
}

func RespondJSON(c *gin.Context, statusCode int, message any) {
	if statusCode >= 400 {
		errorResponse := ErrorResponse{
			Status:    "error",
			Code:      statusCode,
			Message:   constants.ErrMsgInternalServerError, // Default
			Timestamp: time.Now().Format(time.RFC3339),
		}

		switch err := message.(type) {
		case validator.ValidationErrors:
			errorResponse.Message = constants.ErrMsgValidationFailed
			// Dapatkan nama struct dari tipe data yang divalidasi
			var structName string
			if len(err) > 0 {
				// Mengambil nama struct dari StructNamespace field pertama
				// Contoh: "model.RequestPostRegister.FullName" -> "RequestPostRegister"
				parts := strings.Split(err[0].StructNamespace(), ".")
				if len(parts) > 1 {
					structName = parts[len(parts)-2] // Ambil bagian sebelum field name
				}
			}

			for _, e := range err {
				fieldName := e.Field()
				errorResponse.Fields = append(errorResponse.Fields, FieldErrorResponse{
					Field:  fieldName,
					Reason: validationMessage(e),
				})
			}
			// Tambahkan example_request jika ada
			if example, ok := modelExamples[structName]; ok {
				errorResponse.ExampleRequest = example
			}

		case *json.UnmarshalTypeError:
			errorResponse.Message = constants.ErrMsgJSONTypeMismatch
			errorResponse.Fields = append(errorResponse.Fields, FieldErrorResponse{
				Field:  err.Field,
				Reason: fmt.Sprintf("Expected type %s but got %s for field '%s'", err.Type.String(), err.Value, err.Field),
			})
			// Try to generate example based on the expected type
			if err.Type.Kind() == reflect.Struct {
				errorResponse.ExampleRequest = generateExample(err.Type)
			} else {
				errorResponse.ExampleRequest = map[string]any{
					err.Field: defaultValueForType(err.Type),
				}
			}

		case *json.SyntaxError:
			errorResponse.Message = constants.ErrMsgJSONSyntaxError
			errorResponse.Fields = append(errorResponse.Fields, FieldErrorResponse{
				Field:  "JSON",
				Reason: err.Error(),
			})
			if strings.Contains(err.Error(), "EOF") {
				errorResponse.ExampleRequest = map[string]any{} // Empty JSON for EOF
			}

		case error:
			// Generic error, try to extract more specific message
			errMsg := err.Error()
			if strings.Contains(errMsg, "json: cannot unmarshal") {
				expectedType, receivedType := parseUnmarshalError(errMsg)
				errorResponse.Message = constants.ErrMsgJSONTypeMismatch
				errorResponse.Fields = append(errorResponse.Fields, FieldErrorResponse{
					Field:  "JSON",
					Reason: fmt.Sprintf("Expected type %s but got %s", expectedType, receivedType),
				})
				// Attempt to provide a generic example if possible
				errorResponse.ExampleRequest = map[string]any{} // Cannot infer specific field
			} else {
				errorResponse.Message = errMsg // Use the error's message directly
				errorResponse.Fields = append(errorResponse.Fields, FieldErrorResponse{
					Field:  "General",
					Reason: errMsg,
				})
			}

		case string: // Jika pesan error adalah string biasa
			errorResponse.Message = message.(string)
			errorResponse.Fields = append(errorResponse.Fields, FieldErrorResponse{
				Field:  "General",
				Reason: message.(string),
			})

		case map[string]interface{}: // Jika pesan error adalah map (misal dari custom validation)
			errorResponse.Message = constants.ErrMsgBadRequest
			for k, v := range err {
				errorResponse.Fields = append(errorResponse.Fields, FieldErrorResponse{
					Field:  k,
					Reason: fmt.Sprintf("%v", v),
				})
			}

		default:
			errorResponse.Message = fmt.Sprintf("An unexpected error occurred: %v", message)
			errorResponse.Fields = append(errorResponse.Fields, FieldErrorResponse{
				Field:  "Unknown",
				Reason: fmt.Sprintf("%v", message),
			})
		}
		c.JSON(statusCode, errorResponse)
		c.Abort()
		return
	}

	// Jika statusCode < 400, berarti sukses
	successResponse := SuccessResponse{
		Status:    "success",
		Code:      statusCode,
		Message:   "Operasi berhasil.", // Default message, akan di-override
		Timestamp: time.Now().Format(time.RFC3339),
	}

	switch msg := message.(type) {
	case string:
		successResponse.Message = msg
	case gin.H: // Jika handler mengirim gin.H, asumsikan itu data dan mungkin ada "message" di dalamnya
		if m, ok := msg["message"].(string); ok {
			successResponse.Message = m
			delete(msg, "message") // Hapus message dari data agar tidak duplikat
		}
		successResponse.Data = msg
	default: // Jika struct atau tipe data lain
		successResponse.Data = msg
		// Coba infer message dari data jika ada field "message"
		val := reflect.ValueOf(msg)
		if val.Kind() == reflect.Struct {
			if field := val.FieldByName("Message"); field.IsValid() && field.Kind() == reflect.String {
				successResponse.Message = field.String()
			}
		}
	}

	c.IndentedJSON(statusCode, successResponse)
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " wajib diisi."
	case "email":
		return fe.Field() + " harus berupa alamat email yang valid."
	case "min":
		return fe.Field() + " minimal " + fe.Param() + " karakter."
	case "max":
		return fe.Field() + " maksimal " + fe.Param() + " karakter."
	case "gte":
		return fe.Field() + " harus lebih besar atau sama dengan " + fe.Param() + "."
	case "lte":
		return fe.Field() + " harus lebih kecil atau sama dengan " + fe.Param() + "."
	case "gt":
		return fe.Field() + " harus lebih besar dari " + fe.Param() + "."
	case "url":
		return fe.Field() + " harus berupa URL yang valid."
	case "alpha":
		return fe.Field() + " hanya boleh mengandung huruf."
	case "alphanum":
		return fe.Field() + " hanya boleh mengandung huruf dan angka."
	case "numeric":
		return fe.Field() + " hanya boleh mengandung angka."
	case "oneof":
		return fe.Field() + " harus salah satu dari: " + strings.ReplaceAll(fe.Param(), " ", ", ") + "."
	case "uuid":
		return fe.Field() + " harus berupa UUID yang valid."
	case "len":
		return fe.Field() + " harus memiliki panjang " + fe.Param() + " karakter."
	case "eqfield":
		return fe.Field() + " harus sama dengan " + fe.Param() + "."
	case "required_if":
		return fe.Field() + " wajib diisi jika " + fe.Param() + "."
	case "required_without":
		return fe.Field() + " wajib diisi jika " + fe.Param() + " tidak ada."
	default:
		return fe.Field() + " tidak valid."
	}
}

func parseUnmarshalError(errMsg string) (expected string, received string) {
	parts := strings.Split(errMsg, "cannot unmarshal ")
	if len(parts) < 2 {
		return "", ""
	}
	rest := parts[1]
	subParts := strings.Split(rest, " into Go value of type ")
	if len(subParts) == 2 {
		received = strings.Trim(subParts[0], `"`)
		expected = subParts[1]
	}
	return expected, "" // Return empty for expected if it's a complex type
}

func generateExample(t reflect.Type) any {
	// Cek apakah tipe adalah pointer, jika ya, ambil elemennya
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		// Cek di modelExamples berdasarkan nama tipe
		if example, ok := modelExamples[t.Name()]; ok {
			return example
		}

		example := make(map[string]any)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]
			if jsonTag == "-" || jsonTag == "" {
				continue
			}
			example[jsonTag] = defaultValueForType(field.Type)
		}
		return example
	}

	switch t.Kind() {
	case reflect.Slice:
		elem := generateExample(t.Elem())
		return []any{elem}
	default:
		return defaultValueForType(t)
	}
}

func defaultValueForType(t reflect.Type) any {
	// Cek apakah tipe adalah pointer, jika ya, ambil elemennya
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		return 0
	case reflect.Float32, reflect.Float64:
		return 0.0
	case reflect.Bool:
		return true
	case reflect.Struct:
		return generateExample(t)
	case reflect.Slice:
		return []any{defaultValueForType(t.Elem())}
	case reflect.Map:
		return map[string]any{"key": defaultValueForType(t.Elem())}
	default:
		return nil
	}
}