package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"douyin-danmu-tool/danmu"

	"github.com/gorilla/websocket"
)

// DanmuClient 弹幕客户端接口
type DanmuClient interface {
	Connect() error
	Disconnect() error
	IsConnected() bool
	AddHandler(handler danmu.MessageHandler)
}

// ClientWrapper 客户端包装器
type ClientWrapper struct {
	real *danmu.DouyinClient
	mock *danmu.MockClient
}

func (w *ClientWrapper) Connect() error {
	if w.real != nil {
		return w.real.Connect()
	}
	if w.mock != nil {
		return w.mock.Connect()
	}
	return fmt.Errorf("没有可用的客户端")
}

func (w *ClientWrapper) Disconnect() error {
	if w.real != nil {
		return w.real.Disconnect()
	}
	if w.mock != nil {
		return w.mock.Disconnect()
	}
	return nil
}

func (w *ClientWrapper) IsConnected() bool {
	if w.real != nil {
		return w.real.IsConnected()
	}
	if w.mock != nil {
		return w.mock.IsConnected()
	}
	return false
}

func (w *ClientWrapper) AddHandler(handler danmu.MessageHandler) {
	if w.real != nil {
		w.real.AddHandler(handler)
	}
	if w.mock != nil {
		w.mock.AddHandler(handler)
	}
}

// Server Web服务器
type Server struct {
	clients    map[*websocket.Conn]bool
	clientsMu  sync.RWMutex
	danmuClient DanmuClient
	upgrader   websocket.Upgrader
}

// ConnectRequest 连接请求
type ConnectRequest struct {
	RoomID string `json:"room_id"`
}

// Response 响应结构
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewServer 创建新的Web服务器
func NewServer() *Server {
	return &Server{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许跨域
			},
		},
	}
}

// IndexHandler 首页处理器
func (s *Server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>抖音弹幕获取工具</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Microsoft YaHei', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            padding: 40px;
            width: 90%;
            max-width: 1200px;
            min-height: 600px;
        }
        
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        
        .header h1 {
            color: #333;
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        
        .header p {
            color: #666;
            font-size: 1.1em;
        }
        
        .main-content {
            display: grid;
            grid-template-columns: 1fr 300px;
            gap: 30px;
            height: 500px;
        }
        
        .danmu-area {
            background: #f8f9fa;
            border-radius: 15px;
            padding: 20px;
            overflow-y: auto;
            border: 2px solid #e9ecef;
        }
        
        .control-panel {
            background: #f8f9fa;
            border-radius: 15px;
            padding: 20px;
            border: 2px solid #e9ecef;
        }
        
        .form-group {
            margin-bottom: 20px;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 8px;
            font-weight: bold;
            color: #333;
        }
        
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 2px solid #ddd;
            border-radius: 8px;
            font-size: 14px;
            transition: border-color 0.3s;
        }
        
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }
        
        .btn {
            width: 100%;
            padding: 12px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: bold;
            cursor: pointer;
            transition: all 0.3s;
            margin-bottom: 10px;
        }
        
        .btn-primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        
        .btn-danger {
            background: linear-gradient(135deg, #ff6b6b 0%, #ee5a52 100%);
            color: white;
        }
        
        .btn-danger:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(255, 107, 107, 0.4);
        }
        
        .status {
            padding: 10px;
            border-radius: 8px;
            margin-bottom: 20px;
            font-weight: bold;
            text-align: center;
        }
        
        .status.disconnected {
            background: #ffe6e6;
            color: #d63031;
            border: 2px solid #fab1a0;
        }
        
        .status.connected {
            background: #e6ffe6;
            color: #00b894;
            border: 2px solid #81ecec;
        }
        
        .danmu-item {
            background: white;
            padding: 12px;
            margin-bottom: 10px;
            border-radius: 8px;
            border-left: 4px solid #667eea;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
            animation: slideIn 0.3s ease-out;
        }
        
        @keyframes slideIn {
            from {
                opacity: 0;
                transform: translateX(-20px);
            }
            to {
                opacity: 1;
                transform: translateX(0);
            }
        }
        
        .danmu-user {
            font-weight: bold;
            color: #667eea;
            margin-bottom: 5px;
        }
        
        .danmu-content {
            color: #333;
            line-height: 1.4;
        }
        
        .danmu-time {
            font-size: 12px;
            color: #999;
            margin-top: 5px;
        }
        
        .stats {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 10px;
            margin-bottom: 20px;
        }
        
        .stat-item {
            background: white;
            padding: 15px;
            border-radius: 8px;
            text-align: center;
            border: 2px solid #e9ecef;
        }
        
        .stat-number {
            font-size: 24px;
            font-weight: bold;
            color: #667eea;
        }
        
        .stat-label {
            font-size: 12px;
            color: #666;
            margin-top: 5px;
        }
        
        @media (max-width: 768px) {
            .main-content {
                grid-template-columns: 1fr;
                grid-template-rows: auto 1fr;
            }
            
            .container {
                padding: 20px;
                margin: 20px;
            }
            
            .header h1 {
                font-size: 2em;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎭 抖音弹幕获取工具</h1>
            <p>实时获取抖音直播间弹幕，支持数据导出和分析</p>
        </div>
        
        <div class="main-content">
            <div class="danmu-area">
                <h3 style="margin-bottom: 15px; color: #333;">📝 实时弹幕</h3>
                <div id="danmu-list"></div>
            </div>
            
            <div class="control-panel">
                <div id="status" class="status disconnected">
                    🔴 未连接
                </div>
                
                <div class="stats">
                    <div class="stat-item">
                        <div class="stat-number" id="danmu-count">0</div>
                        <div class="stat-label">弹幕总数</div>
                    </div>
                    <div class="stat-item">
                        <div class="stat-number" id="user-count">0</div>
                        <div class="stat-label">用户数</div>
                    </div>
                </div>
                
                <div class="form-group">
                    <label for="room-id">🏠 直播间ID或URL</label>
                    <input type="text" id="room-id" placeholder="请输入抖音直播间ID或完整URL">
                </div>
                
                <button id="connect-btn" class="btn btn-primary">
                    🚀 连接直播间
                </button>
                
                <button id="disconnect-btn" class="btn btn-danger" style="display: none;">
                    ⏹️ 断开连接
                </button>
                
                <button id="clear-btn" class="btn" style="background: #74b9ff; color: white;">
                    🗑️ 清空弹幕
                </button>
                
                <button id="export-btn" class="btn" style="background: #00b894; color: white;">
                    📥 导出数据
                </button>
            </div>
        </div>
    </div>

    <script>
        let ws = null;
        let danmuCount = 0;
        let userSet = new Set();
        let danmuData = [];
        
        const statusEl = document.getElementById('status');
        const danmuListEl = document.getElementById('danmu-list');
        const connectBtn = document.getElementById('connect-btn');
        const disconnectBtn = document.getElementById('disconnect-btn');
        const roomIdInput = document.getElementById('room-id');
        const danmuCountEl = document.getElementById('danmu-count');
        const userCountEl = document.getElementById('user-count');
        
        // 连接WebSocket
        function connectWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = protocol + '//' + window.location.host + '/ws';
            
            ws = new WebSocket(wsUrl);
            
            ws.onopen = function() {
                console.log('WebSocket连接已建立');
            };
            
            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);
                if (data.type === 'danmu') {
                    addDanmu(data);
                }
            };
            
            ws.onclose = function() {
                console.log('WebSocket连接已关闭');
            };
            
            ws.onerror = function(error) {
                console.error('WebSocket错误:', error);
            };
        }
        
        // 添加弹幕到界面
        function addDanmu(data) {
            danmuCount++;
            userSet.add(data.username);
            danmuData.push(data);
            
            const danmuItem = document.createElement('div');
            danmuItem.className = 'danmu-item';
            danmuItem.innerHTML = 
                '<div class="danmu-user">' + escapeHtml(data.username) + '</div>' +
                '<div class="danmu-content">' + escapeHtml(data.content) + '</div>' +
                '<div class="danmu-time">' + new Date(data.timestamp).toLocaleTimeString() + '</div>';
            
            danmuListEl.insertBefore(danmuItem, danmuListEl.firstChild);
            
            // 限制显示的弹幕数量
            if (danmuListEl.children.length > 100) {
                danmuListEl.removeChild(danmuListEl.lastChild);
            }
            
            // 更新统计
            danmuCountEl.textContent = danmuCount;
            userCountEl.textContent = userSet.size;
        }
        
        // HTML转义
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        
        // 连接直播间
        connectBtn.addEventListener('click', function() {
            const roomId = roomIdInput.value.trim();
            if (!roomId) {
                alert('请输入直播间ID或URL');
                return;
            }
            
            fetch('/api/connect', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ room_id: roomId })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    statusEl.textContent = '🟢 已连接';
                    statusEl.className = 'status connected';
                    connectBtn.style.display = 'none';
                    disconnectBtn.style.display = 'block';
                    roomIdInput.disabled = true;
                } else {
                    alert('连接失败: ' + data.message);
                }
            })
            .catch(error => {
                console.error('连接错误:', error);
                alert('连接失败，请检查网络连接');
            });
        });
        
        // 断开连接
        disconnectBtn.addEventListener('click', function() {
            fetch('/api/disconnect', {
                method: 'POST'
            })
            .then(response => response.json())
            .then(data => {
                statusEl.textContent = '🔴 未连接';
                statusEl.className = 'status disconnected';
                connectBtn.style.display = 'block';
                disconnectBtn.style.display = 'none';
                roomIdInput.disabled = false;
            });
        });
        
        // 清空弹幕
        document.getElementById('clear-btn').addEventListener('click', function() {
            danmuListEl.innerHTML = '';
            danmuCount = 0;
            userSet.clear();
            danmuData = [];
            danmuCountEl.textContent = '0';
            userCountEl.textContent = '0';
        });
        
        // 导出数据
        document.getElementById('export-btn').addEventListener('click', function() {
            if (danmuData.length === 0) {
                alert('暂无数据可导出');
                return;
            }
            
            const csvContent = "data:text/csv;charset=utf-8," 
                + "时间,用户名,弹幕内容\n"
                + danmuData.map(item => 
                    '"' + new Date(item.timestamp).toLocaleString() + '",'
                    + '"' + item.username + '",'
                    + '"' + item.content + '"'
                ).join('\n');
            
            const encodedUri = encodeURI(csvContent);
            const link = document.createElement('a');
            link.setAttribute('href', encodedUri);
            link.setAttribute('download', '抖音弹幕数据_' + new Date().toISOString().slice(0,10) + '.csv');
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
        });
        
        // 页面加载时连接WebSocket
        window.addEventListener('load', function() {
            connectWebSocket();
        });
        
        // 回车键连接
        roomIdInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                connectBtn.click();
            }
        });
    </script>
</body>
</html>
`

	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		http.Error(w, "模板解析错误", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, nil)
}

// WebSocketHandler WebSocket处理器
func (s *Server) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 添加客户端
	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	// 移除客户端
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, conn)
		s.clientsMu.Unlock()
	}()

	// 保持连接
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// ConnectHandler 连接处理器
func (s *Server) ConnectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendResponse(w, false, "请求格式错误", nil)
		return
	}

	// 提取房间ID
	roomID, err := danmu.GetRoomIDFromURL(req.RoomID)
	if err != nil {
		s.sendResponse(w, false, fmt.Sprintf("房间ID格式错误: %v", err), nil)
		return
	}

	// 如果已有连接，先断开
	if s.danmuClient != nil {
		s.danmuClient.Disconnect()
	}

	// 创建新的弹幕客户端
	realClient := danmu.NewDouyinClient(roomID)
	realClient.AddHandler(s)

	// 尝试连接到真实直播间
	if err := realClient.Connect(); err != nil {
		log.Printf("真实连接失败，使用模拟模式: %v", err)
		
		// 使用模拟客户端
		mockClient := danmu.NewMockClient(roomID)
		mockClient.AddHandler(s)
		
		if err := mockClient.Connect(); err != nil {
			s.sendResponse(w, false, fmt.Sprintf("模拟连接也失败: %v", err), nil)
			return
		}
		
		// 将模拟客户端转换为通用接口
		s.danmuClient = &ClientWrapper{mock: mockClient}
		s.sendResponse(w, true, "连接成功（演示模式）", map[string]string{"room_id": roomID, "mode": "demo"})
		return
	}

	s.danmuClient = &ClientWrapper{real: realClient}

	s.sendResponse(w, true, "连接成功", map[string]string{"room_id": roomID})
}

// DisconnectHandler 断开连接处理器
func (s *Server) DisconnectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	if s.danmuClient != nil {
		s.danmuClient.Disconnect()
		s.danmuClient = nil
	}

	s.sendResponse(w, true, "已断开连接", nil)
}

// HandleMessage 实现MessageHandler接口
func (s *Server) HandleMessage(msg *danmu.DanmuMessage) {
	// 广播消息到所有WebSocket客户端
	data := map[string]interface{}{
		"type":      "danmu",
		"username":  msg.UserName,
		"content":   msg.Content,
		"timestamp": msg.Timestamp,
		"room_id":   msg.RoomID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("序列化消息失败: %v", err)
		return
	}

	s.clientsMu.RLock()
	clients := make([]*websocket.Conn, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.clientsMu.RUnlock()

	for _, client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			log.Printf("发送消息到客户端失败: %v", err)
			// 移除失效的客户端
			s.clientsMu.Lock()
			delete(s.clients, client)
			s.clientsMu.Unlock()
		}
	}
}

// sendResponse 发送JSON响应
func (s *Server) sendResponse(w http.ResponseWriter, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	response := Response{
		Success: success,
		Message: message,
		Data:    data,
	}
	json.NewEncoder(w).Encode(response)
}

// Close 关闭服务器
func (s *Server) Close() {
	if s.danmuClient != nil {
		s.danmuClient.Disconnect()
	}

	s.clientsMu.Lock()
	for client := range s.clients {
		client.Close()
	}
	s.clientsMu.Unlock()
}