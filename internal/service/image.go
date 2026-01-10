package service

import (
	"context"
	"data/internal/model"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// GeneratePoolImage 生成池子信息图片（黑色背景）
func GeneratePoolImage(ctx context.Context, pool model.Pool) (string, error) {
	var chainName string
	switch pool.ChainID {
	case 56:
		chainName = "BSC"
	case 8453:
		chainName = "Base"
	default:
		chainName = fmt.Sprintf("Chain %d", pool.ChainID)
	}

	tokenPair := fmt.Sprintf("%s/%s", pool.Token0Symbol, pool.Token1Symbol)
	feeText := ""
	if pool.FeeTier == 1 {
		feeText = "0.01%"
	} else if pool.FeeTier == 3 {
		feeText = "1%"
	} else if pool.FeeTier > 0 {
		feeText = fmt.Sprintf("%.2f%%", float64(pool.FeeTier)/100.0)
	} else {
		feeText = "N/A"
	}

	// 格式化APR（复用kyberswap.go中的formatAPR逻辑）
	var aprText string
	if pool.APR >= 1000 {
		aprText = fmt.Sprintf("%.2f%%", pool.APR)
	} else if pool.APR >= 100 {
		aprText = fmt.Sprintf("%.1f%%", pool.APR)
	} else {
		aprText = fmt.Sprintf("%.2f%%", pool.APR)
	}

	// 创建黑色背景图片
	width := 600
	height := 200
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 填充黑色背景
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 255}) // 黑色
		}
	}

	// 设置字体和颜色
	face := basicfont.Face7x13
	white := color.RGBA{255, 255, 255, 255} // 白色文字
	yellow := color.RGBA{255, 255, 0, 255}   // 黄色（用于APR）

	// 绘制文字
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(white),
		Face: face,
		Dot:  fixed.P(20, 40),
	}

	// 第一行：交易对（大号，使用更大的字体）
	drawer.Dot = fixed.P(20, 50)
	drawer.DrawString(tokenPair)

	// 第二行：费率和链
	drawer.Dot = fixed.P(20, 80)
	drawer.DrawString(fmt.Sprintf("Fee: %s  |  %s", feeText, chainName))

	// 第三行：APR（黄色）
	drawer.Src = image.NewUniform(yellow)
	drawer.Dot = fixed.P(20, 110)
	drawer.DrawString(fmt.Sprintf("APR: %s", aprText))

	// 保存图片到临时文件
	tempDir := "temp"
	if !gfile.Exists(tempDir) {
		if err := gfile.Mkdir(tempDir); err != nil {
			return "", fmt.Errorf("创建临时目录失败: %v", err)
		}
	}

	// 使用池子ID的哈希值作为文件名，避免特殊字符问题
	poolIDHash := strings.ReplaceAll(pool.ID, "/", "_")
	poolIDHash = strings.ReplaceAll(poolIDHash, "\\", "_")
	poolIDHash = strings.ReplaceAll(poolIDHash, ":", "_")
	imagePath := filepath.Join(tempDir, fmt.Sprintf("pool_%s.png", poolIDHash))
	file, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("创建图片文件失败: %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return "", fmt.Errorf("编码图片失败: %v", err)
	}

	return imagePath, nil
}
