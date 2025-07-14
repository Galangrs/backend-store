package db

import (
	"time"

	"gorm.io/gorm"
	"database/sql/driver"
	"encoding/json"
	"errors"

	"portolio-backend/configs/constants"
)

type User struct {
	gorm.Model
	FullName        string `gorm:"type:varchar(255);not null" json:"full_name"`
	Email           string `gorm:"type:varchar(255);unique;not null" json:"email"`
	Password        string `gorm:"type:varchar(255);not null" json:"-"`
	Role            constants.UserRole `gorm:"type:varchar(50);default:'user'" json:"role"`
	Balance         uint   `gorm:"default:0" json:"balance"`
	Status          constants.UserStatus `gorm:"type:varchar(50);default:'active'" json:"status"`
	BanUntil        *int64 `gorm:"type:bigint" json:"ban_until,omitempty"`
	BanReason       string `gorm:"type:text" json:"ban_reason,omitempty"`
	PenaltyWarnings uint   `gorm:"default:0" json:"penalty_warnings"`

	Products           []Product           `gorm:"foreignKey:UserID" json:"products,omitempty"`
	TransactionHistories []TransactionHistory `gorm:"foreignKey:UserID" json:"transaction_histories,omitempty"`
	BalanceHistories   []BalanceHistory   `gorm:"foreignKey:UserID" json:"balance_histories,omitempty"`
	Reviews            []Review            `gorm:"foreignKey:UserID" json:"reviews,omitempty"`
	Notifications      []Notification      `gorm:"foreignKey:UserID" json:"notifications,omitempty"`
	ChatMessages       []ChatMessage       `gorm:"foreignKey:SenderID" json:"chat_messages,omitempty"`
	SupportTickets     []SupportTicket     `gorm:"foreignKey:UserID" json:"support_tickets,omitempty"`
	AdminSessions      []AdminSession      `gorm:"foreignKey:UserID" json:"admin_sessions,omitempty"`
	AdminLogs          []AdminLog          `gorm:"foreignKey:AdminID" json:"admin_logs,omitempty"`
}

type Product struct {
	gorm.Model
	UserID     uint   `gorm:"not null" json:"user_id"`
	Title      string `gorm:"type:varchar(255);not null" json:"title"`
	Price      uint   `gorm:"not null" json:"price"`
	Stock      uint   `gorm:"not null" json:"stock"`
	Visibility constants.ProductVisibility `gorm:"type:varchar(50);default:'all'" json:"visibility"`
	Categories string `gorm:"type:text" json:"categories"`
	IsActive   bool   `gorm:"default:true" json:"is_active"`

	User                 User                  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Images               []ProductImage        `gorm:"foreignKey:ProductID" json:"images,omitempty"`
	Reviews              []Review              `gorm:"foreignKey:ProductID" json:"reviews,omitempty"`
	TransactionHistories []TransactionHistory `gorm:"foreignKey:ProductID" json:"transaction_histories,omitempty"`
}

type ProductImage struct {
	gorm.Model
	ProductID uint   `json:"product_id"`
	ImageURL  string `json:"image_url"`

	Product Product `gorm:"foreignKey:ProductID" json:"-"`
}

type TransactionHistory struct {
	gorm.Model
	ProductID     uint   `gorm:"not null" json:"product_id"`
	UserID        uint   `gorm:"not null" json:"user_id"`
	Quantity      uint   `gorm:"not null" json:"quantity"`
	TotalPrice    uint   `gorm:"not null" json:"total_price"`
	GovtTax       uint   `gorm:"default:0" json:"govt_tax"`
	EcommerceTax  uint   `gorm:"default:0" json:"ecommerce_tax"`
	Status        constants.TransactionStatus `gorm:"type:varchar(50);not null" json:"status"`
	IsSolved      bool   `gorm:"default:false" json:"is_solved"`
	ReceiptStatus constants.ReceiptStatus `gorm:"type:varchar(50);default:'PENDING_PROCESS'" json:"receipt_status"`

	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type BalanceHistory struct {
	gorm.Model
	UserID       uint        `json:"user_id"`
	Description  string      `json:"description" validate:"required"`
	Amount       int         `json:"amount"`
	LastBalance  uint        `json:"last_balance"`
	FinalBalance uint        `json:"final_balance"`
	Status       constants.BalanceStatus `json:"status"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

type Review struct {
	gorm.Model
	UserID    uint   `json:"user_id"`
	ProductID uint   `json:"product_id"`
	Rating    uint   `json:"rating" validate:"required,gte=1,lte=5"`
	Comment   string `json:"comment" validate:"max=500"`

	User    User    `gorm:"foreignKey:UserID" json:"-"`
	Product Product `gorm:"foreignKey:ProductID" json:"-"`
}

type Notification struct {
	gorm.Model
	UserID    uint             `gorm:"not null" json:"user_id"`
	Type      constants.NotificationType `gorm:"type:varchar(50);not null" json:"type"`
	Message   string           `gorm:"type:text;not null" json:"message"`
	RelatedID *uint            `json:"related_id,omitempty"`
	IsRead    bool             `gorm:"default:false" json:"is_read"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type ChatMessage struct {
	gorm.Model
	TransactionID uint            `gorm:"not null" json:"transaction_id"`
	SenderID      uint            `gorm:"not null" json:"sender_id"`
	ReceiverID    uint            `gorm:"not null" json:"receiver_id"`
	MessageType   constants.ChatMessageType `gorm:"type:varchar(50);not null" json:"message_type"`
	Content       string          `gorm:"type:text" json:"content"`
	FileURL       string          `gorm:"type:varchar(255)" json:"file_url,omitempty"`
	IsRead        bool            `gorm:"default:false" json:"is_read"`

	Sender      User                `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Transaction TransactionHistory `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

type SupportTicket struct {
	gorm.Model
	UserID          uint                `gorm:"not null" json:"user_id"`
	AssignedAdminID *uint               `json:"assigned_admin_id,omitempty"`
	Subject         string              `gorm:"type:varchar(255);not null" json:"subject"`
	Status          constants.SupportTicketStatus `gorm:"type:varchar(50);default:'open'" json:"status"`
	QueuePos        uint                `gorm:"default:0" json:"queue_pos"`

	User           User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	AssignedAdmin  User            `gorm:"foreignKey:AssignedAdminID" json:"assigned_admin,omitempty"`
	SupportMessages []SupportMessage `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
}

type SupportMessage struct {
	gorm.Model
	TicketID    uint            `gorm:"not null" json:"ticket_id"`
	SenderID    uint            `gorm:"not null" json:"sender_id"`
	MessageType constants.ChatMessageType `gorm:"type:varchar(50);not null" json:"message_type"`
	Content     string          `gorm:"type:text" json:"content"`
	FileURL     string          `gorm:"type:varchar(255)" json:"file_url,omitempty"`

	Sender User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Ticket SupportTicket `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
}

type AdminSession struct {
	gorm.Model
	UserID      uint      `gorm:"not null;" json:"user_id"`
	TokenHash   string    `gorm:"type:varchar(255);not null" json:"-"`
	ExpiresAt   time.Time `gorm:"not null" json:"expires_at"`
	IPAddress   string    `gorm:"type:varchar(50)" json:"ip_address"`
	UserAgent   string    `gorm:"type:text" json:"user_agent"`
	LastUsedAt  time.Time `json:"last_used_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}
	return json.Unmarshal(bytes, &j)
}

type AdminLog struct {
	gorm.Model
	AdminID    uint   `gorm:"not null" json:"admin_id"`
	Action     string `gorm:"type:varchar(255);not null" json:"action"`
	TargetType string `gorm:"type:varchar(50)" json:"target_type"`
	TargetID   *uint  `json:"target_id,omitempty"`
	Details    JSONB  `gorm:"type:jsonb" json:"details"`
	IPAddress  string `gorm:"type:varchar(50)" json:"ip_address"`

	Admin User `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}