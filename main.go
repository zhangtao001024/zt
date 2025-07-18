package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"douyin-danmu-tool/web"
)

func main() {
	var (
		port = flag.String("port", "12000", "服务端口")
		host = flag.String("host", "0.0.0.0", "服务地址")
	)
	flag.Parse()

	// 创建Web服务器
	server := web.NewServer()

	// 设置路由
	http.HandleFunc("/", server.IndexHandler)
	http.HandleFunc("/ws", server.WebSocketHandler)
	http.HandleFunc("/api/connect", server.ConnectHandler)
	http.HandleFunc("/api/disconnect", server.DisconnectHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 启动服务器
	addr := fmt.Sprintf("%s:%s", *host, *port)
	fmt.Printf("抖音弹幕获取工具启动成功！\n")
	fmt.Printf("访问地址: http://localhost:%s\n", *port)
	fmt.Printf("WebSocket地址: ws://localhost:%s/ws\n", *port)

	// 优雅关闭
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal("服务器启动失败:", err)
		}
	}()

	// 等待中断信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n正在关闭服务器...")
	server.Close()
	fmt.Println("服务器已关闭")
}