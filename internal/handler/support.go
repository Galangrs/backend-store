package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/db"
	"portolio-backend/internal/model/dto"
	"portolio-backend/internal/util"
)

type SupportHandler struct {
	db *gorm.DB
}

func NewSupportHandler(db *gorm.DB) *SupportHandler {
	return &SupportHandler{db: db}
}

func (h *SupportHandler) CreateSupportTicket(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	var req dto.RequestCreateTicket
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var maxQueue uint
		tx.Model(&db.SupportTicket{}).Where("status = ?", constants.TicketStatusOpen).Select("COALESCE(MAX(queue_pos), 0)").Row().Scan(&maxQueue)
		newQueuePos := maxQueue + 1

		ticket := db.SupportTicket{
			UserID:    userID,
			Subject:   req.Subject,
			Status:    constants.TicketStatusOpen,
			QueuePos:  newQueuePos,
		}
		if err := tx.Create(&ticket).Error; err != nil {
			return fmt.Errorf("failed to create ticket: %v", err)
		}

		initialMessage := db.SupportMessage{
			TicketID:    ticket.ID,
			SenderID:    userID,
			MessageType: constants.ChatTypeText,
			Content:     req.Message,
		}
		if err := tx.Create(&initialMessage).Error; err != nil {
			return fmt.Errorf("failed to create initial message: %v", err)
		}

		var admins []db.User
		tx.Where("role = ?", constants.RoleAdmin).Find(&admins)
		for _, admin := range admins {
			notification := db.Notification{
				UserID:    admin.ID,
				Type:      constants.NotifTypeSupport,
				Message:   fmt.Sprintf("Tiket dukungan baru dari user ID %d: '%s'. Antrean: %d.", ticket.UserID, ticket.Subject, ticket.QueuePos),
				RelatedID: &ticket.ID,
			}
			util.SendNotificationToUser(admin.ID, dto.NotificationResponse{
				ID:        notification.ID,
				Type:      notification.Type,
				Message:   notification.Message,
				RelatedID: notification.RelatedID,
				CreatedAt: notification.CreatedAt,
				IsRead:    notification.IsRead,
			})
			tx.Create(&notification)
		}

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusCreated, constants.MsgSuccessTicketCreated)
}

func (h *SupportHandler) GetUserSupportTickets(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	status := c.Query("status")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var tickets []db.SupportTicket
	query := h.db.Preload("AssignedAdmin").Where("user_id = ?", userID).Order("created_at DESC")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&db.SupportTicket{}).Count(&total)

	if err := query.Limit(limit).Offset(offset).Find(&tickets).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	var responseTickets []dto.SupportTicketResponse
	for _, ticket := range tickets {
		responseTickets = append(responseTickets, dto.SupportTicketResponse{
			ID:                ticket.ID,
			UserID:            ticket.UserID,
			Subject:           ticket.Subject,
			Status:            ticket.Status,
			AssignedAdminID:   ticket.AssignedAdminID,
			AssignedAdminName: ticket.AssignedAdmin.FullName,
			QueueNumber:       ticket.QueuePos,
			CreatedAt:         ticket.CreatedAt,
			UpdatedAt:         ticket.UpdatedAt,
		})
	}

	util.RespondJSON(c, http.StatusOK, dto.GetSupportTicketsResponse{
		TotalRecords: total,
		Page:         page,
		Limit:        limit,
		Tickets:      responseTickets,
	})
}

func (h *SupportHandler) GetUserSupportTicketMessages(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var ticket db.SupportTicket
	if err := h.db.First(&ticket, ticketID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgTicketNotFound)
		return
	}

	if ticket.UserID != userID {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
		return
	}

	var messages []db.SupportMessage
	if err := h.db.Preload("Sender").Where("ticket_id = ?", ticketID).Order("created_at ASC").Find(&messages).Error; err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, constants.ErrMsgInternalServerError)
		return
	}

	var responseMessages []dto.SupportMessageResponse
	for _, msg := range messages {
		senderName := "Unknown"
		if msg.Sender.ID != 0 {
			senderName = msg.Sender.FullName
		}
		responseMessages = append(responseMessages, dto.SupportMessageResponse{
			ID:          msg.ID,
			TicketID:    msg.TicketID,
			SenderID:    msg.SenderID,
			SenderName:  senderName,
			IsAdmin:     msg.Sender.Role == constants.RoleAdmin,
			Content:     msg.Content,
			MessageType: msg.MessageType,
			FileURL:     msg.FileURL,
			CreatedAt:   msg.CreatedAt,
		})
	}

	util.RespondJSON(c, http.StatusOK, dto.GetSupportTicketMessagesResponse{
		TicketID: ticket.ID,
		Subject:  ticket.Subject,
		Status:   ticket.Status,
		Messages: responseMessages,
	})
}

func (h *SupportHandler) ReplySupportTicket(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var req dto.RequestReplyTicket 
	if err := c.ShouldBindJSON(&req); err != nil {
		util.RespondJSON(c, http.StatusBadRequest, err)
		return
	}

	var ticket db.SupportTicket
	if err := h.db.First(&ticket, ticketID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgTicketNotFound)
		return
	}

	if ticket.UserID != userID {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
		return
	}

	if ticket.Status == constants.TicketStatusClosed || ticket.Status == constants.TicketStatusCanceled {
		util.RespondJSON(c, http.StatusBadRequest, "Tidak dapat membalas tiket yang sudah ditutup atau dibatalkan.")
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		message := db.SupportMessage{
			TicketID:    ticket.ID,
			SenderID:    userID,
			MessageType: req.MessageType,
			Content:     req.Content,
			FileURL:     req.FileURL,
		}
		if err := tx.Create(&message).Error; err != nil {
			return fmt.Errorf("failed to create support message: %v", err)
		}

		if ticket.Status == constants.TicketStatusPendingUser {
			ticket.Status = constants.TicketStatusPendingAdmin
			if err := tx.Save(&ticket).Error; err != nil {
				return fmt.Errorf("failed to update ticket status: %v", err)
			}
		}

		if ticket.AssignedAdminID != nil && *ticket.AssignedAdminID != 0 {
			notificationAdmin := db.Notification{
				UserID:    *ticket.AssignedAdminID,
				Type:      constants.NotifTypeSupport,
				Message:   fmt.Sprintf("Ada balasan baru dari pengguna untuk tiket dukungan #%d ('%s').", ticket.ID, ticket.Subject),
				RelatedID: &ticket.ID,
			}
			util.SendNotificationToUser(*ticket.AssignedAdminID, dto.NotificationResponse{
				ID:        notificationAdmin.ID,
				Type:      notificationAdmin.Type,
				Message:   notificationAdmin.Message,
				RelatedID: notificationAdmin.RelatedID,
				CreatedAt: notificationAdmin.CreatedAt,
				IsRead:    notificationAdmin.IsRead,
			})
			tx.Create(&notificationAdmin)
		} else {
			var admins []db.User
			tx.Where("role = ?", constants.RoleAdmin).Find(&admins)
			for _, admin := range admins {
				notification := db.Notification{
					UserID:    admin.ID,
					Type:      constants.NotifTypeSupport,
					Message:   fmt.Sprintf("Ada balasan baru dari pengguna untuk tiket dukungan #%d ('%s').", ticket.ID, ticket.Subject),
					RelatedID: &ticket.ID,
				}
				util.SendNotificationToUser(admin.ID, dto.NotificationResponse{
					ID:        notification.ID,
					Type:      notification.Type,
					Message:   notification.Message,
					RelatedID: notification.RelatedID,
					CreatedAt: notification.CreatedAt,
					IsRead:    notification.IsRead,
				})
				tx.Create(&notification)
			}
		}

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessMessageSent)
}

func (h *SupportHandler) CancelSupportTicket(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	ticketIDStr := c.Param("id")
	ticketID, err := strconv.ParseUint(ticketIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgBadRequest)
		return
	}

	var ticket db.SupportTicket
	if err := h.db.First(&ticket, ticketID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgTicketNotFound)
		return
	}

	if ticket.UserID != userID {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
		return
	}

	if ticket.Status == constants.TicketStatusClosed || ticket.Status == constants.TicketStatusCanceled {
		util.RespondJSON(c, http.StatusBadRequest, constants.ErrMsgTicketNotCancellable)
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		ticket.Status = constants.TicketStatusCanceled
		if err := tx.Save(&ticket).Error; err != nil {
			return fmt.Errorf("failed to cancel ticket: %v", err)
		}

		if ticket.AssignedAdminID != nil && *ticket.AssignedAdminID != 0 {
			notificationAdmin := db.Notification{
				UserID:    *ticket.AssignedAdminID,
				Type:      constants.NotifTypeSupport,
				Message:   fmt.Sprintf("Tiket dukungan #%d ('%s') telah dibatalkan oleh pengguna.", ticket.ID, ticket.Subject),
				RelatedID: &ticket.ID,
			}
			util.SendNotificationToUser(*ticket.AssignedAdminID, dto.NotificationResponse{
				ID:        notificationAdmin.ID,
				Type:      notificationAdmin.Type,
				Message:   notificationAdmin.Message,
				RelatedID: notificationAdmin.RelatedID,
				CreatedAt: notificationAdmin.CreatedAt,
				IsRead:    notificationAdmin.IsRead,
			})
			tx.Create(&notificationAdmin)
		}

		return nil
	})

	if err != nil {
		util.RespondJSON(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.RespondJSON(c, http.StatusOK, constants.MsgSuccessTicketCanceled)
}