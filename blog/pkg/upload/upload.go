package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxImageSize = 5 << 20 // 5 MB
	baseDir      = "uploads"
)

var allowedExt = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".webp": {},
	".gif":  {},
}

type Result struct {
	URL string
}

func SaveImage(c *gin.Context, field, category string) (*Result, error) {
	fileHeader, err := c.FormFile(field)
	if err != nil {
		return nil, fmt.Errorf("请选择要上传的图片")
	}
	if fileHeader.Size <= 0 {
		return nil, fmt.Errorf("上传文件为空")
	}
	if fileHeader.Size > maxImageSize {
		return nil, fmt.Errorf("图片大小不能超过 5MB")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败")
	}
	defer src.Close()

	head := make([]byte, 512)
	n, err := src.Read(head)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("读取上传文件失败")
	}
	contentType := http.DetectContentType(head[:n])
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("仅支持图片文件上传")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if _, ok := allowedExt[ext]; !ok {
		ext = extByContentType(contentType)
		if _, ok := allowedExt[ext]; !ok {
			return nil, fmt.Errorf("仅支持 jpg、jpeg、png、webp、gif 格式图片")
		}
	}

	relDir := filepath.ToSlash(filepath.Join(baseDir, category, time.Now().Format("20060102")))
	absDir := filepath.Join(".", filepath.FromSlash(relDir))
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败")
	}

	filename := randomName() + ext
	absPath := filepath.Join(absDir, filename)

	dst, err := os.Create(absPath)
	if err != nil {
		return nil, fmt.Errorf("创建目标文件失败")
	}
	defer dst.Close()

	if _, err := dst.Write(head[:n]); err != nil {
		return nil, fmt.Errorf("保存图片失败")
	}
	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("保存图片失败")
	}

	return &Result{
		URL: "/" + strings.TrimPrefix(filepath.ToSlash(filepath.Join(relDir, filename)), "/"),
	}, nil
}

func extByContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

func randomName() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%d_%s", time.Now().Unix(), hex.EncodeToString(buf))
}
