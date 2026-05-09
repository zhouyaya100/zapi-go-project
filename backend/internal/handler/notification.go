package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zapi/zapi-go/internal/core"
	"github.com/zapi/zapi-go/internal/middleware"
	"github.com/zapi/zapi-go/internal/model"
)

func HandleListNotifications(c *gin.Context) {
	u := getUserOrAdmin(c)
	q := model.DB.Model(&model.Notification{})
	// Normal user sees own notifications; admin sees own + broadcast (receiver_id IS NULL)
	if u.Role == "admin" || u.Role == "operator" || u.ID == model.SuperAdminID {
		q = q.Where("receiver_id = ? OR receiver_id IS NULL", u.ID)
	} else {
		q = q.Where("receiver_id = ?", u.ID)
	}
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	if rd := c.Query("read"); rd == "true" {
		q = q.Where("read = ?", true)
	} else if rd == "false" {
		q = q.Where("read = ?", false)
	}
	var total int64
	q.Count(&total)
	offset, limit := middleware.GetPageParams(c)
	var notifs []model.Notification
	q.Order("id desc").Offset(offset).Limit(limit).Find(&notifs)
	// Batch lookup sender names
	senderIDs := make(map[uint]bool)
	for _, n := range notifs {
		if n.SenderID > 0 {
			senderIDs[n.SenderID] = true
		}
	}
	senderMap := make(map[uint]string)
	if len(senderIDs) > 0 {
		var users []model.User
		model.DB.Select("id, username").Find(&users)
		for _, u2 := range users {
			if senderIDs[u2.ID] {
				senderMap[u2.ID] = u2.Username
			}
		}
	}
	items := make([]gin.H, len(notifs))
	for i, n := range notifs {
		item := gin.H{
			"id": n.ID, "category": n.Category, "title": n.Title, "content": n.Content,
			"sender_id": n.SenderID, "sender_name": senderMap[n.SenderID],
			"receiver_id": n.ReceiverID, "read": n.Read,
			"created_at": core.FmtTimeVal(n.CreatedAt),
		}
		items[i] = item
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func HandleCreateNotification(c *gin.Context) {
	a := middleware.GetAuth(c)
	adminID := uint(0)
	if a != nil && a.UserID > 0 {
		adminID = a.UserID
	} else {
		adminID = model.SuperAdminID
	}
	var req struct {
		Title      string `json:"title"`
		Content    string `json:"content"`
		Category   string `json:"category"`
		ReceiverID *uint  `json:"receiver_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u8bf7\u6c42\u53c2\u6570\u9519\u8bef"}})
		return
	}
	if req.ReceiverID != nil {
		// Send to specific user (exclude self)
		if *req.ReceiverID != adminID {
			n := model.Notification{SenderID: adminID, ReceiverID: req.ReceiverID, Category: req.Category, Title: req.Title, Content: req.Content}
			model.DB.Create(&n)
		}
	} else {
		// Broadcast to all enabled users (exclude self)
		var users []model.User
		model.DB.Where("enabled = ? AND id != ?", true, adminID).Find(&users)
		notifs := make([]model.Notification, 0, len(users))
		for _, u := range users {
			notifs = append(notifs, model.Notification{SenderID: adminID, ReceiverID: &u.ID, Category: req.Category, Title: req.Title, Content: req.Content})
		}
		if len(notifs) > 0 { model.DB.CreateInBatches(notifs, 100) }
	}
	c.JSON(200, gin.H{"success": true})
}

func HandleMarkRead(c *gin.Context) {
	id := c.Param("id")
	u := getUserOrAdmin(c)
	var n model.Notification
	if model.DB.First(&n, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u901a\u77e5\u4e0d\u5b58\u5728"}})
		return
	}
	if n.ReceiverID != nil && *n.ReceiverID != u.ID && u.Role != "admin" {
		c.JSON(403, gin.H{"error": gin.H{"message": "\u4e0d\u662f\u60a8\u7684\u901a\u77e5"}})
		return
	}
	model.DB.Model(&n).Update("read", true)
	c.JSON(200, gin.H{"success": true})
}

func HandleMarkAllRead(c *gin.Context) {
	u := getUserOrAdmin(c)
	model.DB.Model(&model.Notification{}).Where("receiver_id = ? AND read = ?", u.ID, false).Update("read", true)
	c.JSON(200, gin.H{"success": true})
}

func HandleUnreadCount(c *gin.Context) {
	u := getUserOrAdmin(c)
	var count int64
	model.DB.Model(&model.Notification{}).Where("receiver_id = ? AND read = ?", u.ID, false).Count(&count)
	c.JSON(200, gin.H{"count": count})
}

func HandleDeleteNotification(c *gin.Context) {
	id := c.Param("id")
	u := getUserOrAdmin(c)
	var n model.Notification
	if model.DB.First(&n, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u901a\u77e5\u4e0d\u5b58\u5728"}})
		return
	}
	if u.Role != "admin" && (n.ReceiverID == nil || *n.ReceiverID != u.ID) {
		c.JSON(403, gin.H{"error": gin.H{"message": "\u4e0d\u662f\u60a8\u7684\u901a\u77e5"}})
		return
	}
	model.DB.Delete(&n)
	c.JSON(200, gin.H{"success": true})
}

func HandleBatchSendNotification(c *gin.Context) {
	a := middleware.GetAuth(c)
	adminID := uint(0)
	if a != nil && a.UserID > 0 {
		adminID = a.UserID
	} else {
		adminID = model.SuperAdminID
	}
	var req struct {
		ReceiverIDs []uint `json:"receiver_ids"`
		Category    string `json:"category"`
		Title       string `json:"title"`
		Content     string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": gin.H{"message": "\u8bf7\u6c42\u53c2\u6570\u9519\u8bef"}})
		return
	}
	count := 0
	notifs := make([]model.Notification, 0, len(req.ReceiverIDs))
	for _, uid := range req.ReceiverIDs {
		if uid == adminID { continue }
		notifs = append(notifs, model.Notification{SenderID: adminID, ReceiverID: &uid, Category: req.Category, Title: req.Title, Content: req.Content})
		count++
	}
	if len(notifs) > 0 { model.DB.CreateInBatches(notifs, 100) }
	c.JSON(200, gin.H{"success": true, "count": count})
}

func HandleListSentNotifications(c *gin.Context) {
	a := middleware.GetAuth(c)
	adminID := uint(0)
	if a != nil && a.UserID > 0 {
		adminID = a.UserID
	} else {
		adminID = model.SuperAdminID
	}
	q := model.DB.Model(&model.Notification{}).Where("sender_id = ?", adminID)
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	var total int64
	q.Count(&total)
	offset, limit := middleware.GetPageParams(c)
	var notifs []model.Notification
	q.Order("id desc").Offset(offset).Limit(limit).Find(&notifs)
	// Batch count recipients using GROUP BY
	type batchCount struct {
		SenderID       uint
		Title          string
		Category       string
		DateStr        string
		RecipientCount int64
	}
	var batchCounts []batchCount
	if len(notifs) > 0 {
		model.DB.Model(&model.Notification{}).
			Select("sender_id, title, category, DATE(created_at) as date_str, count(*) as recipient_count").
			Where("sender_id = ?", adminID).
			Group("sender_id, title, category, DATE(created_at)").
			Scan(&batchCounts)
	}
	countMap := make(map[string]int64)
	for _, bc := range batchCounts {
		key := fmt.Sprintf("%d:%s:%s:%s", bc.SenderID, bc.Title, bc.Category, bc.DateStr)
		countMap[key] = bc.RecipientCount
	}
	senderMap := make(map[uint]string)
	var users []model.User
	model.DB.Select("id, username").Find(&users)
	for _, u := range users {
		senderMap[u.ID] = u.Username
	}
	items := make([]gin.H, len(notifs))
	for i, n := range notifs {
		key := fmt.Sprintf("%d:%s:%s:%s", n.SenderID, n.Title, n.Category, n.CreatedAt.Format("2006-01-02"))
		rc := countMap[key]
		items[i] = gin.H{
			"id": n.ID, "category": n.Category, "title": n.Title, "content": n.Content,
			"sender_id": n.SenderID, "sender_name": senderMap[n.SenderID],
			"recipient_count": rc, "created_at": core.FmtTimeVal(n.CreatedAt),
		}
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func HandleDeleteSentNotification(c *gin.Context) {
	id := c.Param("id")
	var n model.Notification
	if model.DB.First(&n, id).Error != nil {
		c.JSON(404, gin.H{"error": gin.H{"message": "\u901a\u77e5\u4e0d\u5b58\u5728"}})
		return
	}
	// Delete all notifications with same sender+title+content+category+date
	model.DB.Where("sender_id = ? AND title = ? AND content = ? AND category = ? AND DATE(created_at) = ?",
		n.SenderID, n.Title, n.Content, n.Category, n.CreatedAt.Format("2006-01-02")).Delete(&model.Notification{})
	c.JSON(200, gin.H{"success": true})
}

func HandleDeleteOldSentNotifications(c *gin.Context) {
	cutoff := time.Now().UTC().AddDate(0, 0, -30)
	model.DB.Where("sender_id IS NOT NULL AND sender_id > 0").Where("created_at < ?", cutoff).Delete(&model.Notification{})
	c.JSON(200, gin.H{"success": true})
}

// Suppress unused import
var _ = strconv.Itoa
