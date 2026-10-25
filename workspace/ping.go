package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/ping", pingHandler)
	
	fmt.Println("HTTP 服务器启动，监听端口 :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("服务器启动失败: %s\n", err)
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	
	// 返回 JSON 响应
	response := map[string]interface{}{
		"code":    200,
		"message": "pong",
	}
	
	// 编码为 JSON 并写入响应
	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorResponse := map[string]interface{}{
			"code":    500,
			"message": "响应生成失败",
		}
		json.NewEncoder(w).Encode(errorResponse)
	}
}