package main

import (
	"fmt"
	"log"
	"time"

	"douyin-danmu-tool/danmu"
)

// TestHandler 测试消息处理器
type TestHandler struct{}

func (h *TestHandler) HandleMessage(msg *danmu.DanmuMessage) {
	fmt.Printf("[%s] %s: %s\n", 
		msg.Timestamp.Format("15:04:05"), 
		msg.UserName, 
		msg.Content)
}

func main() {
	// 创建测试客户端
	client := danmu.NewDouyinClient("123456789")
	
	// 添加测试处理器
	handler := &TestHandler{}
	client.AddHandler(handler)
	
	fmt.Println("正在连接到抖音直播间...")
	
	// 尝试连接
	if err := client.Connect(); err != nil {
		log.Printf("连接失败: %v", err)
		
		// 模拟一些测试数据
		fmt.Println("连接失败，生成模拟弹幕数据进行演示...")
		
		testMessages := []*danmu.DanmuMessage{
			{
				Type:      "chat",
				UserName:  "测试用户1",
				Content:   "这是一条测试弹幕",
				Timestamp: time.Now(),
				RoomID:    "123456789",
			},
			{
				Type:      "chat", 
				UserName:  "测试用户2",
				Content:   "666666",
				Timestamp: time.Now(),
				RoomID:    "123456789",
			},
			{
				Type:      "chat",
				UserName:  "测试用户3", 
				Content:   "主播好厉害！",
				Timestamp: time.Now(),
				RoomID:    "123456789",
			},
		}
		
		for _, msg := range testMessages {
			handler.HandleMessage(msg)
			time.Sleep(1 * time.Second)
		}
		
		return
	}
	
	fmt.Println("连接成功！正在接收弹幕...")
	
	// 保持连接30秒
	time.Sleep(30 * time.Second)
	
	// 断开连接
	client.Disconnect()
	fmt.Println("测试完成")
}