package danmu

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DouyinClient 抖音弹幕客户端
type DouyinClient struct {
	roomID      string
	conn        *websocket.Conn
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	isConnected bool
	handlers    []MessageHandler
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(msg *DanmuMessage)
}

// DanmuMessage 弹幕消息结构
type DanmuMessage struct {
	Type      string    `json:"type"`
	UserName  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id"`
	RoomID    string    `json:"room_id"`
}

// NewDouyinClient 创建新的抖音客户端
func NewDouyinClient(roomID string) *DouyinClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &DouyinClient{
		roomID:   roomID,
		ctx:      ctx,
		cancel:   cancel,
		handlers: make([]MessageHandler, 0),
	}
}

// AddHandler 添加消息处理器
func (c *DouyinClient) AddHandler(handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, handler)
}

// Connect 连接到抖音直播间
func (c *DouyinClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return fmt.Errorf("已经连接到直播间")
	}

	// 获取WebSocket URL
	wsURL, err := c.getWebSocketURL()
	if err != nil {
		return fmt.Errorf("获取WebSocket URL失败: %v", err)
	}

	// 建立WebSocket连接
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	headers := http.Header{
		"User-Agent": []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"},
		"Origin":     []string{"https://live.douyin.com"},
	}

	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("WebSocket连接失败: %v", err)
	}

	c.conn = conn
	c.isConnected = true

	// 启动消息接收协程
	go c.messageLoop()

	// 启动心跳协程
	go c.heartbeat()

	log.Printf("成功连接到抖音直播间: %s", c.roomID)
	return nil
}

// Disconnect 断开连接
func (c *DouyinClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected {
		return nil
	}

	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
	c.isConnected = false

	log.Printf("已断开与直播间 %s 的连接", c.roomID)
	return nil
}

// IsConnected 检查是否已连接
func (c *DouyinClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// getWebSocketURL 获取WebSocket连接URL
func (c *DouyinClient) getWebSocketURL() (string, error) {
	// 构建WebSocket URL
	// 这里使用简化的URL构建，实际项目中可能需要更复杂的签名算法
	baseURL := "wss://webcast3-ws-web-hl.douyin.com/webcast/im/push/v2/"
	
	params := url.Values{}
	params.Set("app_name", "douyin_web")
	params.Set("version_code", "180800")
	params.Set("webcast_sdk_version", "1.3.0")
	params.Set("update_version_code", "1.3.0")
	params.Set("compress", "gzip")
	params.Set("device_platform", "web")
	params.Set("cookie_enabled", "true")
	params.Set("screen_width", "1920")
	params.Set("screen_height", "1080")
	params.Set("browser_language", "zh-CN")
	params.Set("browser_platform", "Win32")
	params.Set("browser_name", "Mozilla")
	params.Set("browser_version", "5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	params.Set("browser_online", "true")
	params.Set("tz_name", "Asia/Shanghai")
	params.Set("identity", "audience")
	params.Set("room_id", c.roomID)
	params.Set("heartbeatDuration", "0")
	params.Set("signature", c.generateSignature())

	return baseURL + "?" + params.Encode(), nil
}

// generateSignature 生成签名（简化版本）
func (c *DouyinClient) generateSignature() string {
	// 这里是简化的签名生成，实际项目中需要更复杂的算法
	timestamp := time.Now().Unix()
	return fmt.Sprintf("00000000000000000000000000000000_%d", timestamp)
}

// messageLoop 消息接收循环
func (c *DouyinClient) messageLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("消息循环异常: %v", r)
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if !c.isConnected {
				return
			}

			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Printf("读取消息失败: %v", err)
				c.handleDisconnect()
				return
			}

			// 处理接收到的消息
			c.processMessage(message)
		}
	}
}

// processMessage 处理接收到的消息
func (c *DouyinClient) processMessage(data []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("处理消息异常: %v", r)
		}
	}()

	// 解析消息头
	if len(data) < 4 {
		return
	}

	// 读取消息长度
	msgLen := binary.BigEndian.Uint32(data[:4])
	if len(data) < int(msgLen) {
		return
	}

	// 提取payload
	payload := data[4:msgLen]

	// 尝试解压缩
	if len(payload) > 0 {
		if decompressed, err := c.decompressGzip(payload); err == nil {
			payload = decompressed
		}
	}

	// 解析弹幕消息（简化版本）
	msg := c.parseMessage(payload)
	if msg != nil {
		// 通知所有处理器
		c.mu.RLock()
		handlers := make([]MessageHandler, len(c.handlers))
		copy(handlers, c.handlers)
		c.mu.RUnlock()

		for _, handler := range handlers {
			go handler.HandleMessage(msg)
		}
	}
}

// decompressGzip 解压缩gzip数据
func (c *DouyinClient) decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// parseMessage 解析消息（简化版本）
func (c *DouyinClient) parseMessage(data []byte) *DanmuMessage {
	// 这里是简化的消息解析，实际项目中需要使用protobuf
	content := string(data)
	
	// 简单的文本匹配来提取弹幕信息
	if strings.Contains(content, "chat") || strings.Contains(content, "message") {
		// 尝试提取用户名和内容
		username := c.extractUsername(content)
		text := c.extractText(content)
		
		if text != "" {
			return &DanmuMessage{
				Type:      "chat",
				UserName:  username,
				Content:   text,
				Timestamp: time.Now(),
				UserID:    "",
				RoomID:    c.roomID,
			}
		}
	}

	return nil
}

// extractUsername 提取用户名（简化版本）
func (c *DouyinClient) extractUsername(content string) string {
	// 使用正则表达式尝试提取用户名
	re := regexp.MustCompile(`"nickname":"([^"]+)"`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return "匿名用户"
}

// extractText 提取弹幕文本（简化版本）
func (c *DouyinClient) extractText(content string) string {
	// 使用正则表达式尝试提取弹幕内容
	re := regexp.MustCompile(`"content":"([^"]+)"`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// heartbeat 心跳保持连接
func (c *DouyinClient) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if !c.isConnected {
				return
			}

			// 发送心跳消息
			heartbeatMsg := []byte("ping")
			if err := c.conn.WriteMessage(websocket.TextMessage, heartbeatMsg); err != nil {
				log.Printf("发送心跳失败: %v", err)
				c.handleDisconnect()
				return
			}
		}
	}
}

// handleDisconnect 处理断开连接
func (c *DouyinClient) handleDisconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		c.isConnected = false
		if c.conn != nil {
			c.conn.Close()
		}
		log.Printf("与直播间 %s 的连接已断开", c.roomID)
	}
}

// GetRoomIDFromURL 从抖音直播间URL提取房间ID
func GetRoomIDFromURL(liveURL string) (string, error) {
	// 简化的房间ID提取逻辑
	re := regexp.MustCompile(`/(\d+)`)
	matches := re.FindStringSubmatch(liveURL)
	if len(matches) > 1 {
		return matches[1], nil
	}

	// 如果直接是数字，则认为是房间ID
	if _, err := strconv.Atoi(liveURL); err == nil {
		return liveURL, nil
	}

	return "", fmt.Errorf("无法从URL中提取房间ID: %s", liveURL)
}