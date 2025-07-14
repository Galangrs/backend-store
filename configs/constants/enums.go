package constants

type UserRole string
const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
	RoleGuest UserRole = "guest"
)

type UserStatus string
const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusBanned    UserStatus = "banned"
)


type ProductVisibility string
const (
	ProductVisibilityAll        ProductVisibility = "all"
	ProductVisibilityOwnerAdmin ProductVisibility = "owner_admin"
	ProductVisibilityAdminOnly  ProductVisibility = "admin_only"
)

type TransactionStatus string
const (
	TrxStatusPending      TransactionStatus = "pending"
	TrxStatusWaitingOwner TransactionStatus = "waiting_owner"
	TrxStatusWaitingUser  TransactionStatus = "waiting_users"
	TrxStatusSuccess      TransactionStatus = "success"
	TrxStatusCancel       TransactionStatus = "cancel"
)

type ReceiptStatus string
const (
	ReceiptPendingProcess ReceiptStatus = "PENDING_PROCESS"
	ReceiptCompleted      ReceiptStatus = "COMPLETED"
	ReceiptCanceled       ReceiptStatus = "CANCELED"
)

type BalanceStatus string
const (
	BalanceStatusCredit BalanceStatus = "credit"
	BalanceStatusDebit  BalanceStatus = "debit"
	BalanceStatusRefund BalanceStatus = "refund"
)

type NotificationType string
const (
	NotifTypePurchase   NotificationType = "purchase"
	NotifTypeSale       NotificationType = "sale"
	NotifTypeTopUp      NotificationType = "topup"
	NotifTypeWithdraw   NotificationType = "withdraw"
	NotifTypeChat       NotificationType = "chat"
	NotifTypeSupport    NotificationType = "support"
	NotifTypeAccount    NotificationType = "account_status"
)

type ChatMessageType string
const (
	ChatTypeText  ChatMessageType = "text"
	ChatTypeImage ChatMessageType = "image"
	ChatTypeFile  ChatMessageType = "file"
)

type SupportTicketStatus string
const (
	TicketStatusOpen        SupportTicketStatus = "open"
	TicketStatusPendingAdmin SupportTicketStatus = "pending_admin"
	TicketStatusPendingUser SupportTicketStatus = "pending_user"
	TicketStatusClosed      SupportTicketStatus = "closed"
	TicketStatusTransferred SupportTicketStatus = "transferred"
	TicketStatusCanceled    SupportTicketStatus = "canceled"
)

const (
	DefaultGovtTaxPercent      = 0.05
	DefaultEcommerceTaxPercent = 0.02
)

const (
	MaxImageSizeMB    = 5
	MaxDocumentSizeMB = 10
)

const (
	DefaultPenaltyWarningLimit = 3
)