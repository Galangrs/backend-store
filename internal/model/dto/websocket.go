package dto

import (
	"time"

	"portolio-backend/configs/constants"
)

type RequestSendMessage struct {
	TransactionID uint                      `json:"transaction_id"`
	MessageType   constants.ChatMessageType `json:"message_type" binding:"required,oneof=text image file"`
	Content       string                    `json:"content,omitempty" binding:"required_without=FileURL,max=1000"`
	FileURL       string                    `json:"file_url,omitempty" binding:"required_if=MessageType image required_if=MessageType file,url"`
}

type ChatMessageResponse struct {
	ID            uint                      `json:"id"`
	TransactionID uint                      `json:"transaction_id"`
	SenderID      uint                      `json:"sender_id"`
	SenderName    string                    `json:"sender_name"`
	MessageType   constants.ChatMessageType `json:"message_type"`
	Content       string                    `json:"content"`
	FileURL       string                    `json:"file_url,omitempty"`
	CreatedAt     time.Time                 `json:"created_at"`
}

type NotificationResponse struct {
	ID        uint                        `json:"id"`
	UserID    uint                        `json:"user_id"`
	Type      constants.NotificationType  `json:"type"`
	Message   string                      `json:"message"`
	RelatedID *uint                       `json:"related_id,omitempty"`
	CreatedAt time.Time                   `json:"created_at"`
	IsRead    bool                        `json:"is_read"`
}

type ChatMessageWebSocketResponse struct {
	Type    string              `json:"type"` // e.g., "chat_message"
	Message ChatMessageResponse `json:"message"`
}

type SuccessMessageWebSocketResponse struct {
	Status      string              `json:"status"`
	Message     string              `json:"message"`
	ChatMessage ChatMessageResponse `json:"chat_message"`
}

type NotificationWebSocketResponse struct {
	Type    string               `json:"type"` // e.g., "notification"
	Message NotificationResponse `json:"message"`
}