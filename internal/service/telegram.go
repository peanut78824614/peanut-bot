package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type ITelegram interface {
	SendMessage(ctx context.Context, chatID, text string) error
	SendMessageWithMarkdown(ctx context.Context, chatID, text string) error
	GetUpdates(ctx context.Context) ([]Update, error)
	GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error)
}

// Update 表示Telegram更新
type Update struct {
	UpdateID int64 `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// Message 表示Telegram消息
type Message struct {
	MessageID int64 `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`
	Chat      *Chat `json:"chat"`
	Text      string `json:"text,omitempty"`
	Date      int64 `json:"date"`
}

// TelegramUser 表示Telegram用户
type TelegramUser struct {
	ID        int64 `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat 表示Telegram聊天
type Chat struct {
	ID        int64 `json:"id"`
	Type      string `json:"type"` // "private", "group", "supergroup", "channel"
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// ChatInfo 表示聊天信息
type ChatInfo struct {
	ID        int64 `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type telegramImpl struct {
	botToken string
}

var telegramService *telegramImpl

// Telegram 获取 Telegram 服务实例
func Telegram() ITelegram {
	if telegramService == nil {
		telegramService = &telegramImpl{
			botToken: g.Cfg().MustGet(context.Background(), "telegram.botToken", "").String(),
		}
	}
	return telegramService
}

// SendMessage 发送普通消息
func (s *telegramImpl) SendMessage(ctx context.Context, chatID, text string) error {
	return s.sendMessage(ctx, chatID, text, false)
}

// SendMessageWithMarkdown 发送 Markdown 格式消息
func (s *telegramImpl) SendMessageWithMarkdown(ctx context.Context, chatID, text string) error {
	return s.sendMessage(ctx, chatID, text, true)
}

// sendMessage 发送消息的内部实现
func (s *telegramImpl) sendMessage(ctx context.Context, chatID, text string, parseMode bool) error {
	if s.botToken == "" {
		return fmt.Errorf("Telegram bot token 未配置")
	}
	
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.botToken)
	
	params := url.Values{}
	params.Set("chat_id", chatID)
	params.Set("text", text)
	if parseMode {
		params.Set("parse_mode", "Markdown")
	}
	params.Set("disable_web_page_preview", "false")
	
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Description string `json:"description"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return fmt.Errorf("Telegram API 错误: %s", errorResp.Description)
		}
		return fmt.Errorf("Telegram API 错误: HTTP %d", resp.StatusCode)
	}
	
	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	
	if !result.OK {
		return fmt.Errorf("Telegram API 返回失败")
	}
	
	return nil
}

// GetUpdates 获取 Telegram 更新
func (s *telegramImpl) GetUpdates(ctx context.Context) ([]Update, error) {
	if s.botToken == "" {
		return nil, fmt.Errorf("Telegram bot token 未配置")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", s.botToken)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Telegram API 错误: HTTP %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(body))
	}

	if !result.OK {
		return nil, fmt.Errorf("Telegram API 返回失败")
	}

	return result.Result, nil
}

// GetChatInfo 获取聊天信息
func (s *telegramImpl) GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error) {
	if s.botToken == "" {
		return nil, fmt.Errorf("Telegram bot token 未配置")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getChat", s.botToken)

	params := url.Values{}
	params.Set("chat_id", chatID)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Telegram API 错误: HTTP %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result struct {
		OK     bool     `json:"ok"`
		Result ChatInfo `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(body))
	}

	if !result.OK {
		return nil, fmt.Errorf("Telegram API 返回失败")
	}

	return &result.Result, nil
}
