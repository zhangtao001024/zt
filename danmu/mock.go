package danmu

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// MockClient 模拟弹幕客户端（用于演示）
type MockClient struct {
	roomID      string
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	isConnected bool
	handlers    []MessageHandler
}

// NewMockClient 创建模拟客户端
func NewMockClient(roomID string) *MockClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &MockClient{
		roomID:   roomID,
		ctx:      ctx,
		cancel:   cancel,
		handlers: make([]MessageHandler, 0),
	}
}

// AddHandler 添加消息处理器
func (c *MockClient) AddHandler(handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, handler)
}

// Connect 连接（模拟）
func (c *MockClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return fmt.Errorf("已经连接到直播间")
	}

	c.isConnected = true

	// 启动模拟消息生成协程
	go c.generateMockMessages()

	return nil
}

// Disconnect 断开连接
func (c *MockClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected {
		return nil
	}

	c.cancel()
	c.isConnected = false
	return nil
}

// IsConnected 检查是否已连接
func (c *MockClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// generateMockMessages 生成模拟弹幕消息
func (c *MockClient) generateMockMessages() {
	mockUsers := []string{
		"小明同学", "爱看直播的小红", "游戏高手", "路过的观众", "粉丝一号",
		"夜猫子", "学生党", "上班族", "宝妈", "大叔",
		"小仙女", "技术宅", "吃货", "旅行者", "音乐爱好者",
	}

	mockMessages := []string{
		"666666", "主播好厉害！", "这个操作太秀了", "学到了学到了",
		"哈哈哈哈", "太搞笑了", "主播加油！", "支持支持",
		"这是什么神仙操作", "我也想学", "教教我吧", "太强了",
		"牛牛牛", "厉害厉害", "点赞点赞", "关注了关注了",
		"第一次看这个主播", "很有意思", "继续继续", "不错不错",
		"哇塞", "太棒了", "学会了", "谢谢主播",
		"好看好看", "有趣有趣", "再来一个", "精彩精彩",
	}

	ticker := time.NewTicker(time.Duration(rand.Intn(3)+1) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if !c.isConnected {
				return
			}

			// 随机生成弹幕
			user := mockUsers[rand.Intn(len(mockUsers))]
			content := mockMessages[rand.Intn(len(mockMessages))]

			msg := &DanmuMessage{
				Type:      "chat",
				UserName:  user,
				Content:   content,
				Timestamp: time.Now(),
				UserID:    fmt.Sprintf("user_%d", rand.Intn(10000)),
				RoomID:    c.roomID,
			}

			// 通知所有处理器
			c.mu.RLock()
			handlers := make([]MessageHandler, len(c.handlers))
			copy(handlers, c.handlers)
			c.mu.RUnlock()

			for _, handler := range handlers {
				go handler.HandleMessage(msg)
			}

			// 随机调整下次发送时间
			ticker.Reset(time.Duration(rand.Intn(4)+1) * time.Second)
		}
	}
}