package main

import (
	"log"
	"time"

	"nexus/internal/config"
	"nexus/internal/database"
	"nexus/internal/handler"
	"nexus/internal/llm"
	"nexus/internal/router"
	"nexus/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置: %v", err)
	}

	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("init database: %v", err)
	}

	// mock 内置回显 Provider；provider=openai 的 Agent 按自身 base_url/api_key/model
	// 动态构造真实 Provider，config.yml 的 llm 段仅作为页面留空时的回退默认值。
	llmDefaults := service.LLMDefaults{
		BaseURL:   cfg.LLM.BaseURL,
		APIKey:    cfg.LLM.APIKey,
		Model:     cfg.LLM.Model,
		MaxTokens: cfg.LLM.MaxTokens,
		Timeout:   time.Duration(cfg.LLM.TimeoutSeconds) * time.Second,
	}

	userSvc := service.NewUserService(db, time.Duration(cfg.Auth.SessionTTLHours)*time.Hour)
	agentSvc := service.NewAgentService(db)
	convSvc := service.NewConversationService(db)
	chatSvc := service.NewChatService(db, llm.NewMockProvider(), llmDefaults)

	authH := handler.NewAuthHandler(userSvc, cfg.Auth)
	agentH := handler.NewAgentHandler(agentSvc)
	convH := handler.NewConversationHandler(convSvc)
	msgH := handler.NewMessageHandler(chatSvc)

	r := router.New(router.Handlers{
		Auth:         authH,
		Agent:        agentH,
		Conversation: convH,
		Message:      msgH,
	}, userSvc)

	addr := ":" + cfg.Server.Port
	log.Printf("Nexus server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
