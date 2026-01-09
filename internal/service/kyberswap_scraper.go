package service

import (
	"context"
	"data/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

// fetchPoolsFromPage 从网页获取池子数据（备用方法）
func (s *kyberSwapImpl) fetchPoolsFromPage(ctx context.Context, page int) ([]model.Pool, error) {
	url := fmt.Sprintf("https://kyberswap.com/earn/pools?tag=high_apr&chainIds=56%%2C8453&page=%d", page)
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	htmlContent := string(body)
	
	// 尝试从 HTML 中提取 JSON 数据
	// 很多现代网站会在 script 标签中嵌入 JSON 数据
	pools := s.extractPoolsFromHTML(htmlContent)
	
	return pools, nil
}

// extractPoolsFromHTML 从 HTML 中提取池子数据
func (s *kyberSwapImpl) extractPoolsFromHTML(html string) []model.Pool {
	pools := make([]model.Pool, 0)
	
	// 尝试查找 JSON 数据
	// 方法1: 查找 window.__INITIAL_STATE__ 或类似变量
	patterns := []string{
		`window\.__INITIAL_STATE__\s*=\s*({.+?});`,
		`window\.__NEXT_DATA__\s*=\s*({.+?})`,
		`"pools":\s*(\[.+?\])`,
		`pools:\s*(\[.+?\])`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			var data interface{}
			if err := json.Unmarshal([]byte(matches[1]), &data); err == nil {
				// 尝试解析为池子数据
				if parsed := s.parsePoolsFromJSON(data); len(parsed) > 0 {
					return parsed
				}
			}
		}
	}
	
	// 方法2: 查找所有可能的 JSON 对象
	jsonPattern := regexp.MustCompile(`\{[^{}]*"apr"[^{}]*\}`)
	matches := jsonPattern.FindAllString(html, -1)
	
	for _, match := range matches {
		var pool model.Pool
		if err := json.Unmarshal([]byte(match), &pool); err == nil && pool.ID != "" {
			pools = append(pools, pool)
		}
	}
	
	return pools
}

// parsePoolsFromJSON 从 JSON 数据中解析池子
func (s *kyberSwapImpl) parsePoolsFromJSON(data interface{}) []model.Pool {
	pools := make([]model.Pool, 0)
	
	// 尝试不同的 JSON 结构
	// 这里需要根据实际 API 响应格式调整
	// 暂时返回空列表，等待实际 API 响应后再调整
	
	return pools
}
