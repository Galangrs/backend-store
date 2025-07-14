package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"portolio-backend/configs/constants"
	"portolio-backend/internal/model/db"
	"portolio-backend/internal/model/dto"
	"portolio-backend/internal/util"
)

type WebsocketHandler struct {
	db  *gorm.DB
	hub *util.Hub
}

func NewWebsocketHandler(db *gorm.DB, hub *util.Hub) *WebsocketHandler {
	return &WebsocketHandler{db: db, hub: hub}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const (
	writeWait = 10 * time.Second
	pongWait = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMessageSize = 512
)

func (h *WebsocketHandler) ChatHandler(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	roleRaw, exists := c.Get("ROLE")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	role := roleRaw.(constants.UserRole)

	if role == constants.RoleGuest {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
		return
	}

	transactionIDStr := c.Param("transaction_id")
	transactionID, err := strconv.ParseUint(transactionIDStr, 10, 64)
	if err != nil {
		util.RespondJSON(c, http.StatusBadRequest, "ID transaksi tidak valid.")
		return
	}

	var transaction db.TransactionHistory
	if err := h.db.Preload("Product").First(&transaction, transactionID).Error; err != nil {
		util.RespondJSON(c, http.StatusNotFound, constants.ErrMsgTransactionNotFound)
		return
	}

	if transaction.UserID != userID && transaction.Product.UserID != userID {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
		return
	}

	if transaction.Status == constants.TrxStatusSuccess || transaction.Status == constants.TrxStatusCancel {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgChatNotAllowed)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Gagal mengupgrade ke websocket: %v", err)
		return 
	}

	client := &util.Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256), 
		Stop:   make(chan struct{}),    
	}
	h.hub.RegisterClient(client)

	go h.readPumpChat(client, uint(transactionID))
	go writePump(client)
}

func (h *WebsocketHandler) readPumpChat(client *util.Client, transactionID uint) {
	defer func() {
		h.hub.UnregisterClient(client)
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(appData string) error {
		log.Printf("Received pong from UserID %d", client.UserID)
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-client.Stop: 
			log.Printf("Stopping readPump for UserID %d (transaction %d) via stop signal.", client.UserID, transactionID)
			return
		default:
		}

		mt, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error for UserID %d (transaction %d): %v", client.UserID, transactionID, err)
			} else {
				log.Printf("WebSocket connection closed for UserID %d (transaction %d): %v", client.UserID, transactionID, err)
			}
			break 
		}

		if mt != websocket.TextMessage {
			log.Printf("Received non-text message type %d from UserID %d, skipping.", mt, client.UserID)
			continue
		}

		var req dto.RequestSendMessage
		if err := json.Unmarshal(message, &req); err != nil {
			log.Printf("Gagal mengurai pesan chat dari UserID %d: %v. Pesan asli: %s", client.UserID, err, string(message))
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": constants.ErrMsgBadRequest})
			client.Send <- errMsg
			continue
		}

		req.TransactionID = transactionID

		if req.MessageType == "" || (req.Content == "" && req.FileURL == "") {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "Tipe pesan atau konten/URL file tidak ada."})
			client.Send <- errMsg
			continue
		}
		if (req.MessageType == constants.ChatTypeImage || req.MessageType == constants.ChatTypeFile) && req.FileURL == "" {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "FileURL diperlukan untuk pesan gambar/file."})
			client.Send <- errMsg
			continue
		}

		var transaction db.TransactionHistory
		if err := h.db.Preload("Product").First(&transaction, req.TransactionID).Error; err != nil {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": constants.ErrMsgTransactionNotFound})
			client.Send <- errMsg
			continue
		}

		if transaction.UserID != client.UserID && transaction.Product.UserID != client.UserID {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "Anda bukan bagian dari transaksi ini."})
			client.Send <- errMsg
			continue
		}

		if transaction.Status == constants.TrxStatusSuccess || transaction.Status == constants.TrxStatusCancel {
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": constants.ErrMsgChatNotAllowed})
			client.Send <- errMsg
			continue
		}

		var receiverID uint
		if transaction.UserID == client.UserID {
			receiverID = transaction.Product.UserID
		} else {
			receiverID = transaction.UserID
		}

		chatMessage := db.ChatMessage{
			TransactionID: req.TransactionID,
			SenderID:      client.UserID,
			ReceiverID:    receiverID,
			MessageType:   req.MessageType,
			Content:       req.Content,
			FileURL:       req.FileURL,
			IsRead:        false,
		}

		if err := h.db.Create(&chatMessage).Error; err != nil {
			log.Printf("Gagal menyimpan pesan chat ke DB: %v", err)
			errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": constants.ErrMsgInternalServerError})
			client.Send <- errMsg
			continue
		}

		var senderUser db.User
		if err := h.db.First(&senderUser, client.UserID).Error; err != nil {
			log.Printf("Failed to get sender name for UserID %d: %v", client.UserID, err)
			senderUser.FullName = "Unknown"
		}

		notification := db.Notification{
			UserID:    receiverID,
			Type:      constants.NotificationType(constants.NotifTypeChat),
			Message:   fmt.Sprintf("Anda memiliki pesan baru dari %s terkait transaksi ID %d.", senderUser.FullName, req.TransactionID),
			RelatedID: &chatMessage.ID,
		}

		if err := h.db.Create(&notification).Error; err != nil { 
			log.Printf("Gagal menyimpan notifikasi chat ke DB: %v", err)
		}
		util.SendNotificationToUser(receiverID, dto.NotificationResponse{
			ID:        notification.ID,
			Type:      notification.Type,
			Message:   notification.Message,
			RelatedID: notification.RelatedID,
			CreatedAt: notification.CreatedAt,
			IsRead:    notification.IsRead,
		})

		chatMsgResp := dto.ChatMessageResponse{
			ID:            chatMessage.ID,
			TransactionID: chatMessage.TransactionID,
			SenderID:      chatMessage.SenderID,
			SenderName:    senderUser.FullName, 
			MessageType:   chatMessage.MessageType,
			Content:       chatMessage.Content,
			FileURL:       chatMessage.FileURL,
			CreatedAt:     chatMessage.CreatedAt,
		}

		chatWebSocketResp := dto.ChatMessageWebSocketResponse{
			Type:    "chat_message",
			Message: chatMsgResp,
		}
		jsonMsgToReceiver, _ := json.Marshal(chatWebSocketResp)
		util.SendToUser(receiverID, jsonMsgToReceiver)

		successWebSocketResp := dto.SuccessMessageWebSocketResponse{
			Status:      "sent",
			Message:     "Pesan berhasil dikirim dan disimpan.",
			ChatMessage: chatMsgResp,
		}
		successMsgToSender, _ := json.Marshal(successWebSocketResp)
		client.Send <- successMsgToSender
	}
}

func (h *WebsocketHandler) NotificationHandler(c *gin.Context) {
	userIDRaw, exists := c.Get("ID")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	userID := userIDRaw.(uint)

	roleRaw, exists := c.Get("ROLE")
	if !exists {
		util.RespondJSON(c, http.StatusUnauthorized, constants.ErrMsgUnauthorized)
		return
	}
	role := roleRaw.(constants.UserRole)

	if role == constants.RoleGuest {
		util.RespondJSON(c, http.StatusForbidden, constants.ErrMsgForbidden)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Gagal mengupgrade ke websocket: %v", err)
		return
	}

	client := &util.Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Stop:   make(chan struct{}),
	}
	h.hub.RegisterClient(client)

	go h.readPumpGeneric(client) 
	go writePump(client)

	go h.sendUnreadNotifications(client)
}

func (h *WebsocketHandler) readPumpGeneric(client *util.Client) {
	defer func() {
		h.hub.UnregisterClient(client)
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(appData string) error {
		log.Printf("Received pong from UserID %d (Notification connection)", client.UserID)
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-client.Stop:
			log.Printf("Stopping readPumpGeneric for UserID %d via stop signal.", client.UserID)
			return
		default:
			_, _, err := client.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error for UserID %d (Notification): %v", client.UserID, err)
				} else {
					log.Printf("WebSocket connection closed for UserID %d (Notification): %v", client.UserID, err)
				}
				return
			}
			client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		}
	}
}

func (h *WebsocketHandler) sendUnreadNotifications(client *util.Client) {
	var notifications []db.Notification
	if err := h.db.Where("user_id = ? AND is_read = ?", client.UserID, false).Order("created_at DESC").Find(&notifications).Error; err != nil {
		log.Printf("Gagal mengambil notifikasi belum dibaca untuk user %d: %v", client.UserID, err)
		return
	}

	for _, notif := range notifications {
		resp := dto.NotificationResponse{
			ID:        notif.ID,
			Type:      notif.Type,
			Message:   notif.Message,
			RelatedID: notif.RelatedID,
			CreatedAt: notif.CreatedAt,
			IsRead:    notif.IsRead,
		}
		notificationWebSocketResp := dto.NotificationWebSocketResponse{
			Type:    "notification",
			Message: resp,
		}
		jsonMsg, err := json.Marshal(notificationWebSocketResp)
		if err != nil {
			log.Printf("Gagal mengurai notifikasi ke JSON: %v", err)
			continue
		}
		select {
		case client.Send <- jsonMsg:
		case <-client.Stop:
			log.Printf("Stopping sending unread notifications for UserID %d via stop signal.", client.UserID)
			return
		default:
			log.Printf("Channel kirim notifikasi penuh untuk user %d saat mengirim yang belum dibaca. Pesan ID: %d", client.UserID, notif.ID)
		}
	}

	if len(notifications) > 0 {
		var ids []uint
		for _, n := range notifications {
			ids = append(ids, n.ID)
		}
		if err := h.db.Model(&db.Notification{}).Where("id IN (?)", ids).Update("is_read", true).Error; err != nil {
			log.Printf("Failed to mark notifications as read for user %d: %v", client.UserID, err)
		} else {
			log.Printf("Marked %d notifications as read for UserID %d", len(notifications), client.UserID)
		}
	}
}

func writePump(client *util.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
		log.Printf("Closed connection for UserID %d (writePump ended).", client.UserID)
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				log.Printf("Client.Send channel closed for UserID %d.", client.UserID)
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting next writer for UserID %d: %v", client.UserID, err)
				return
			}
			w.Write(message)
			n := len(client.Send)
			for i := 0; i < n; i++ {
				select {
				case msg := <-client.Send:
					w.Write(msg)
				default:
					break
				}
			}

			if err := w.Close(); err != nil {
				log.Printf("Error closing writer for UserID %d: %v", client.UserID, err)
				return
			}
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping to UserID %d: %v", client.UserID, err)
				return
			}
		case <-client.Stop:
			log.Printf("Stopping writePump for UserID %d via stop signal.", client.UserID)
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
				log.Printf("Error sending close message on stop for UserID %d: %v", client.UserID, err)
			}
			return
		}
	}
}