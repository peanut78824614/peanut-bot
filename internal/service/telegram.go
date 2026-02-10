package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type ITelegram interface {
	SendMessage(ctx context.Context, chatID, text string) error
	SendMessageWithMarkdown(ctx context.Context, chatID, text string) error
	SendMessageWithMarkdownAndButton(ctx context.Context, chatID, text, buttonText, buttonURL string) error
	SendPhoto(ctx context.Context, chatID string, photoPath string, caption string) error
	SendPhotoByURL(ctx context.Context, chatID string, photoURL string, caption string) error
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
	return s.sendMessage(ctx, chatID, text, false, "", "")
}

// SendMessageWithMarkdown 发送 Markdown 格式消息
func (s *telegramImpl) SendMessageWithMarkdown(ctx context.Context, chatID, text string) error {
	return s.sendMessage(ctx, chatID, text, true, "", "")
}

// SendMessageWithMarkdownAndButton 发送 Markdown 消息并在底部带一条宽链接按钮
func (s *telegramImpl) SendMessageWithMarkdownAndButton(ctx context.Context, chatID, text, buttonText, buttonURL string) error {
	return s.sendMessage(ctx, chatID, text, true, buttonText, buttonURL)
}

// telegramSendBody 用于 JSON 方式发送带按钮的消息，保证 reply_markup 正确传递
type telegramSendBody struct {
	ChatID                string                 `json:"chat_id"`
	Text                  string                 `json:"text"`
	ParseMode             string                 `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool                   `json:"disable_web_page_preview,omitempty"`
	ReplyMarkup           *telegramInlineKeyboard `json:"reply_markup,omitempty"`
}
type telegramInlineKeyboard struct {
	InlineKeyboard [][]telegramInlineButton `json:"inline_keyboard"`
}
type telegramInlineButton struct {
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
}

// sendMessage 发送消息的内部实现；buttonText/buttonURL 非空时附加底部 InlineKeyboard URL 按钮
// 带按钮时使用 JSON 请求体，避免 form 编码导致 reply_markup 被丢弃或解析失败
// 遇 connection reset / timeout 等网络瞬时错误时自动重试最多 3 次
func (s *telegramImpl) sendMessage(ctx context.Context, chatID, text string, parseMode bool, buttonText, buttonURL string) error {
	if s.botToken == "" {
		return fmt.Errorf("Telegram bot token 未配置")
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.botToken)
	var bodyBytes []byte
	if buttonText != "" && buttonURL != "" {
		body := telegramSendBody{
			ChatID:                chatID,
			Text:                  text,
			ParseMode:             "Markdown",
			DisableWebPagePreview: false,
			ReplyMarkup: &telegramInlineKeyboard{
				InlineKeyboard: [][]telegramInlineButton{
					{{Text: buttonText, URL: buttonURL}},
				},
			},
		}
		if !parseMode {
			body.ParseMode = ""
		}
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return err
		}
	} else {
		params := url.Values{}
		params.Set("chat_id", chatID)
		params.Set("text", text)
		if parseMode {
			params.Set("parse_mode", "Markdown")
		}
		params.Set("disable_web_page_preview", "false")
		bodyBytes = []byte(params.Encode())
	}
	client := &http.Client{Timeout: 45 * time.Second}
	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		contentType := "application/x-www-form-urlencoded"
		if buttonText != "" && buttonURL != "" {
			contentType = "application/json"
		}
		req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", contentType)
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if isRetriableNetworkError(err) && attempt < maxRetries-1 {
				g.Log().Warning(ctx, fmt.Sprintf("Telegram 发送网络错误，第 %d 次重试: %v", attempt+1, err))
				continue
			}
			return err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			var errorResp struct {
				Description string `json:"description"`
			}
			if json.Unmarshal(body, &errorResp) == nil {
				lastErr = fmt.Errorf("Telegram API 错误: %s", errorResp.Description)
			} else {
				lastErr = fmt.Errorf("Telegram API 错误: HTTP %d", resp.StatusCode)
			}
			continue
		}
		var result struct {
			OK bool `json:"ok"`
		}
		if json.Unmarshal(body, &result) != nil || !result.OK {
			lastErr = fmt.Errorf("Telegram API 返回失败")
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("Telegram 发送失败，已重试 %d 次", maxRetries)
}

// isRetriableNetworkError 判断是否为可重试的网络错误（连接被重置、超时等）
func isRetriableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "connection reset") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "EOF") ||
		strings.Contains(s, "TLS handshake")
}

// escapeJSONString 转义 JSON 字符串中的 \ 和 "
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
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

// SendPhotoByURL 通过图片 URL 发送图片到 Telegram（Telegram 会从该 URL 拉取图片）
func (s *telegramImpl) SendPhotoByURL(ctx context.Context, chatID string, photoURL string, caption string) error {
	if s.botToken == "" {
		return fmt.Errorf("Telegram bot token 未配置")
	}
	if photoURL == "" {
		return fmt.Errorf("图片 URL 不能为空")
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", s.botToken)
	params := url.Values{}
	params.Set("chat_id", chatID)
	params.Set("photo", photoURL)
	if caption != "" {
		params.Set("caption", caption)
		params.Set("parse_mode", "Markdown")
	}
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 30 * time.Second}
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
		if json.Unmarshal(body, &errorResp) == nil {
			return fmt.Errorf("Telegram API 错误: %s", errorResp.Description)
		}
		return fmt.Errorf("Telegram API 错误: HTTP %d", resp.StatusCode)
	}
	var result struct {
		OK bool `json:"ok"`
	}
	if json.Unmarshal(body, &result) != nil || !result.OK {
		return fmt.Errorf("Telegram API 返回失败")
	}
	return nil
}

// SendPhoto 发送图片到Telegram（本地文件路径）
func (s *telegramImpl) SendPhoto(ctx context.Context, chatID string, photoPath string, caption string) error {
	if s.botToken == "" {
		return fmt.Errorf("Telegram bot token 未配置")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", s.botToken)

	// 读取图片文件（二进制数据）
	photoData, err := os.ReadFile(photoPath)
	if err != nil {
		return fmt.Errorf("读取图片文件失败: %v", err)
	}

	// 创建multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 添加chat_id
	writer.WriteField("chat_id", chatID)
	
	// 添加caption（如果有）
	if caption != "" {
		writer.WriteField("caption", caption)
		writer.WriteField("parse_mode", "Markdown")
	}

	// 添加图片文件
	part, err := writer.CreateFormFile("photo", "pool_info.png")
	if err != nil {
		return fmt.Errorf("创建表单字段失败: %v", err)
	}
	_, err = part.Write([]byte(photoData))
	if err != nil {
		return fmt.Errorf("写入图片数据失败: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("关闭writer失败: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, &requestBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

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
		return fmt.Errorf("Telegram API 错误: HTTP %d, 响应: %s", resp.StatusCode, string(body))
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
