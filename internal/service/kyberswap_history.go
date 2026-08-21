package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gogf/gf/v2/frame/g"
)

const defaultEarnFeeHistoryFilePath = "data/pool_earn_fee_history.json"

var (
	earnFeeHistoryMu       sync.Mutex
	earnFeeHistoryFilePath = defaultEarnFeeHistoryFilePath
)

// parseEarnFeeHistoryContent 解析 earnFee 历史文件。
// 关键：json.Unmarshal 遇到「完整 JSON 后面还有残留字节」会报
// invalid character '"' after top-level value；json.Decoder 只读第一个值，
// 能从并发写入残留的 "timestamp": ... 中恢复出前面那份完整记录。
func parseEarnFeeHistoryContent(raw []byte) (history map[string]EarnFeeHistory, trailing bool, err error) {
	return parseEarnFeeHistoryContentLimited(raw, true)
}

func parseEarnFeeHistoryContentLimited(raw []byte, allowRecover bool) (map[string]EarnFeeHistory, bool, error) {
	raw = bytes.TrimSpace(raw)
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || bytes.Equal(raw, []byte("{}")) {
		return map[string]EarnFeeHistory{}, false, nil
	}

	var newer map[string]EarnFeeHistory
	if t, e := decodeFirstJSON(raw, &newer); e == nil && newer != nil {
		return newer, t, nil
	}

	var old map[string]float64
	if t, e := decodeFirstJSON(raw, &old); e == nil && old != nil {
		converted := make(map[string]EarnFeeHistory, len(old))
		now := time.Now()
		for id, value := range old {
			converted[id] = EarnFeeHistory{Value: value, Timestamp: now}
		}
		return converted, t, nil
	}

	if allowRecover {
		if recovered, ok := recoverJSONObject(raw); ok && !bytes.Equal(recovered, raw) {
			if h, _, e := parseEarnFeeHistoryContentLimited(recovered, false); e == nil {
				return h, true, nil
			}
		}
	}

	return nil, false, fmt.Errorf("earnFee 历史文件无法解析")
}

func decodeFirstJSON(raw []byte, dest interface{}) (trailing bool, err error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err = dec.Decode(dest); err != nil {
		return false, err
	}
	var extra json.RawMessage
	switch e := dec.Decode(&extra); e {
	case nil:
		return true, nil
	case io.EOF:
		return false, nil
	default:
		return true, nil
	}
}

func recoverJSONObject(raw []byte) ([]byte, bool) {
	start := bytes.IndexByte(raw, '{')
	if start < 0 {
		return nil, false
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(raw); {
		c := raw[i]
		if inString {
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == '"' {
				inString = false
			}
			i++
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return raw[start : i+1], true
			}
		}
		i++
	}
	return nil, false
}

func loadEarnFeeHistoryUnlocked(ctx context.Context) (map[string]EarnFeeHistory, error) {
	path := earnFeeHistoryFilePath
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]EarnFeeHistory{}, nil
		}
		return nil, err
	}

	history, trailing, parseErr := parseEarnFeeHistoryContent(content)
	if parseErr != nil {
		backupCorruptHistoryFile(ctx, path, content, parseErr)
		empty := map[string]EarnFeeHistory{}
		if err := saveEarnFeeHistoryUnlocked(empty); err != nil {
			g.Log().Warning(ctx, "重建空的 earnFee 历史文件失败:", err)
		}
		return empty, nil
	}
	if history == nil {
		history = map[string]EarnFeeHistory{}
	}
	if trailing {
		g.Log().Warning(ctx, "earnFee 历史文件存在尾部残留（多为并发写入导致），已恢复第一个完整 JSON 并重写干净文件")
		if err := saveEarnFeeHistoryUnlocked(history); err != nil {
			g.Log().Warning(ctx, "重写干净的 earnFee 历史文件失败:", err)
		}
	}
	return history, nil
}

func backupCorruptHistoryFile(ctx context.Context, path string, content []byte, parseErr error) {
	previewLen := 200
	if len(content) < previewLen {
		previewLen = len(content)
	}
	g.Log().Warning(ctx, fmt.Sprintf("earnFee 历史文件损坏，已备份后重建空记录: %v; 文件前 %d 字节: %q", parseErr, previewLen, sanitizePreview(content[:previewLen])))

	backup := path + ".corrupt." + time.Now().Format("20060102-150405")
	if err := os.WriteFile(backup, content, 0644); err != nil {
		g.Log().Warning(ctx, "备份损坏的 earnFee 历史文件失败:", err)
	}
}

func sanitizePreview(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	return string(bytes.ToValidUTF8(b, []byte("?")))
}

func saveEarnFeeHistoryUnlocked(history map[string]EarnFeeHistory) error {
	if history == nil {
		history = map[string]EarnFeeHistory{}
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(earnFeeHistoryFilePath, data)
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".pool_earn_fee_history-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	return nil
}

// GetPoolEarnFeeHistory 获取所有池子的 earnFee 历史值（兼容旧版本）
func (s *kyberSwapImpl) GetPoolEarnFeeHistory(ctx context.Context) (map[string]float64, error) {
	historyWithTime, err := s.GetPoolEarnFeeHistoryWithTime(ctx)
	if err != nil {
		g.Log().Warning(ctx, "解析 earnFee 历史（含时间戳）失败，将使用空记录继续:", err)
		return make(map[string]float64), nil
	}
	history := make(map[string]float64, len(historyWithTime))
	for id, h := range historyWithTime {
		history[id] = h.Value
	}
	return history, nil
}

// GetPoolEarnFeeHistoryWithTime 获取所有池子的 earnFee 历史值和时间戳。
// 解析失败时返回空记录，不把 JSON 错误抛给任务层，避免监控中断。
func (s *kyberSwapImpl) GetPoolEarnFeeHistoryWithTime(ctx context.Context) (map[string]EarnFeeHistory, error) {
	earnFeeHistoryMu.Lock()
	defer earnFeeHistoryMu.Unlock()
	history, err := loadEarnFeeHistoryUnlocked(ctx)
	if err != nil {
		g.Log().Warning(ctx, "读取 earnFee 历史失败，将使用空记录继续:", err)
		return map[string]EarnFeeHistory{}, nil
	}
	return history, nil
}

// UpdatePoolEarnFeeHistory 更新指定池子的 earnFee 历史值
func (s *kyberSwapImpl) UpdatePoolEarnFeeHistory(ctx context.Context, poolID string, earnFee float64) error {
	return s.UpdatePoolEarnFeeHistories(ctx, map[string]float64{poolID: earnFee})
}

// UpdatePoolEarnFeeHistories 在一次加锁 + 一次原子写中批量更新历史值，避免每池写一次文件。
func (s *kyberSwapImpl) UpdatePoolEarnFeeHistories(ctx context.Context, updates map[string]float64) error {
	if len(updates) == 0 {
		return nil
	}
	earnFeeHistoryMu.Lock()
	defer earnFeeHistoryMu.Unlock()

	history, err := loadEarnFeeHistoryUnlocked(ctx)
	if err != nil {
		g.Log().Warning(ctx, "解析 earnFee 历史失败，使用空记录重新写入:", err)
		history = make(map[string]EarnFeeHistory)
	}
	now := time.Now()
	for poolID, earnFee := range updates {
		history[poolID] = EarnFeeHistory{
			Value:     earnFee,
			Timestamp: now,
		}
	}
	return saveEarnFeeHistoryUnlocked(history)
}
