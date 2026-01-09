package service

import (
	"context"
	"data/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type IWeChat interface {
	SendMessage(ctx context.Context, text string) error
	SendMarkdown(ctx context.Context, content string) error
	SendPoolMessage(ctx context.Context, pools []model.Pool) error
}

type weChatImpl struct {
	serviceType string // "serverchan", "wxpusher", "qywx" (企业微信)
	apiKey      string
	webhookURL  string
	uid         string // WxPusher 的 UID
}

var weChatService *weChatImpl

// WeChat 获取微信推送服务实例
func WeChat() IWeChat {
	if weChatService == nil {
		ctx := context.Background()
		serviceType := g.Cfg().MustGet(ctx, "wechat.serviceType", "serverchan").String()
		apiKey := g.Cfg().MustGet(ctx, "wechat.apiKey", "").String()
		webhookURL := g.Cfg().MustGet(ctx, "wechat.webhookUrl", "").String()
		uid := g.Cfg().MustGet(ctx, "wechat.uid", "").String()

		// 如果是企业微信，支持从 key 构建 URL
		if serviceType == "qywx" && webhookURL == "" {
			key := g.Cfg().MustGet(ctx, "wechat.webhookKey", "").String()
			if key != "" {
				webhookURL = fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=%s", key)
			}
		}

		weChatService = &weChatImpl{
			serviceType: serviceType,
			apiKey:      apiKey,
			webhookURL:  webhookURL,
			uid:         uid,
		}
	}
	return weChatService
}

// SendMessage 发送文本消息
func (s *weChatImpl) SendMessage(ctx context.Context, text string) error {
	switch s.serviceType {
	case "serverchan":
		return s.sendServerChan(ctx, text)
	case "wxpusher":
		return s.sendWxPusher(ctx, text)
	case "qywx":
		return s.sendQyWx(ctx, text)
	default:
		return fmt.Errorf("不支持的微信服务类型: %s", s.serviceType)
	}
}

// SendMarkdown 发送 Markdown 格式消息
func (s *weChatImpl) SendMarkdown(ctx context.Context, content string) error {
	// 大多数微信推送服务不支持 Markdown，转换为文本
	return s.SendMessage(ctx, content)
}

// SendPoolMessage 发送池子消息（格式化为文本）
func (s *weChatImpl) SendPoolMessage(ctx context.Context, pools []model.Pool) error {
	if len(pools) == 0 {
		return nil
	}

	// 格式化消息
	message := FormatPoolsMessageForWeChat(pools)

	// 企业微信 Markdown 支持有限，使用文本格式
	return s.SendMessage(ctx, message)
}

// sendServerChan 发送到 Server酱
func (s *weChatImpl) sendServerChan(ctx context.Context, text string) error {
	if s.apiKey == "" {
		return fmt.Errorf("Server酱 API Key 未配置")
	}

	// Server酱 API (新版本)
	apiURL := fmt.Sprintf("https://sctapi.ftqq.com/%s.send", s.apiKey)

	params := url.Values{}
	params.Set("title", "KyberSwap 新池子通知")
	params.Set("desp", text)

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
		return fmt.Errorf("Server酱 API 错误: HTTP %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("Server酱 API 错误: %d - %s", result.Code, result.Message)
	}

	return nil
}

// sendWxPusher 发送到 WxPusher
func (s *weChatImpl) sendWxPusher(ctx context.Context, text string) error {
	if s.apiKey == "" {
		return fmt.Errorf("WxPusher AppToken 未配置")
	}
	if s.uid == "" {
		return fmt.Errorf("WxPusher UID 未配置")
	}

	apiURL := "https://wxpusher.zjiecode.com/api/send/message"

	payload := map[string]interface{}{
		"appToken":    s.apiKey,
		"content":     text,
		"summary":     "KyberSwap 新池子通知",
		"contentType": 1, // 文本
		"uids":        []string{s.uid},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

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
		return fmt.Errorf("WxPusher API 错误: HTTP %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("WxPusher API 错误: %s", result.Msg)
	}

	return nil
}

// sendQyWx 发送到企业微信（保留原有功能）
func (s *weChatImpl) sendQyWx(ctx context.Context, text string) error {
	if s.webhookURL == "" {
		return fmt.Errorf("企业微信 Webhook URL 未配置")
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": text,
		},
	}

	return s.sendQyWxRequest(ctx, payload)
}

// sendQyWxRequest 发送请求到企业微信
func (s *weChatImpl) sendQyWxRequest(ctx context.Context, payload map[string]interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

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
		return fmt.Errorf("企业微信 API 错误: HTTP %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("企业微信 API 错误: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// FormatPoolsMessageForWeChat 格式化池子消息用于企业微信
func FormatPoolsMessageForWeChat(pools []model.Pool) string {
	if len(pools) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("🎉 发现 %d 个新池子\n\n", len(pools)))

	for i, pool := range pools {
		builder.WriteString(fmt.Sprintf("%d. 🎯 新发现高 APR 池子\n\n", i+1))
		builder.WriteString(fmt.Sprintf("📊 %s\n", pool.Name))
		builder.WriteString(fmt.Sprintf("💰 APR: %s\n", formatAPRForWeChat(pool.APR)))
		builder.WriteString(fmt.Sprintf("💎 TVL: %s\n", formatTVLForWeChat(pool.TVL)))
		builder.WriteString(fmt.Sprintf("🔄 交易对: %s / %s\n", pool.Token0Symbol, pool.Token1Symbol))

		var chainName string
		switch pool.ChainID {
		case 56:
			chainName = "BSC"
		case 8453:
			chainName = "Base"
		default:
			chainName = fmt.Sprintf("Chain %d", pool.ChainID)
		}
		builder.WriteString(fmt.Sprintf("⛓️ 链: %s\n", chainName))

		if pool.Volume24h > 0 {
			builder.WriteString(fmt.Sprintf("📈 24h 交易量: %s\n", formatTVLForWeChat(pool.Volume24h)))
		}
		if pool.Fees24h > 0 {
			builder.WriteString(fmt.Sprintf("💵 24h 手续费: %s\n", formatTVLForWeChat(pool.Fees24h)))
		}
		builder.WriteString(fmt.Sprintf("🔗 查看详情: %s\n", pool.URL))

		if i < len(pools)-1 {
			builder.WriteString("\n---\n\n")
		}
	}

	return builder.String()
}

// formatAPRForWeChat 格式化 APR（企业微信版本）
func formatAPRForWeChat(apr float64) string {
	if apr >= 1000 {
		return fmt.Sprintf("%.2f%%", apr)
	} else if apr >= 100 {
		return fmt.Sprintf("%.1f%%", apr)
	} else {
		return fmt.Sprintf("%.2f%%", apr)
	}
}

// formatTVLForWeChat 格式化 TVL（企业微信版本）
func formatTVLForWeChat(tvl float64) string {
	if tvl >= 1000000 {
		return fmt.Sprintf("$%.2fM", tvl/1000000)
	} else if tvl >= 1000 {
		return fmt.Sprintf("$%.2fK", tvl/1000)
	} else {
		return fmt.Sprintf("$%.2f", tvl)
	}
}
