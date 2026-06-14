package minio

import (
	"blog/pkg/config"
	"bufio"
	"context"

	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	mp3dec "github.com/tcolgate/mp3"
)

// Client MinIO 客户端封装
type Client struct {
	client     *minio.Client
	bucketName string
	baseURL    string
}

// NewClient 初始化 MinIO 客户端，并确保 Bucket 存在
func NewClient(cfg config.MinioConfig) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.AccessKeySecret, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 MinIO 失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("检查 Bucket 失败: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("创建 Bucket 失败: %w", err)
		}
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		baseURL = scheme + "://" + cfg.Endpoint
	}

	return &Client{
		client:     client,
		bucketName: cfg.BucketName,
		baseURL:    baseURL,
	}, nil
}

// Upload 上传文件到 MinIO
// objectName: 存储路径，如 "images/20240101/abc123.jpg"
// reader: 文件内容读取器
// size: 文件大小（字节），-1 表示未知
// contentType: MIME 类型，如 "image/jpeg"
func (c *Client) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	// 归一化对象名称
	objectName = c.NormalizeObjectName(objectName)

	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	if size >= 0 {
		opts.PartSize = 0 // 小于 5MB 的文件不分片
	}

	_, err := c.client.PutObject(ctx, c.bucketName, objectName, reader, size, opts)
	if err != nil {
		return "", fmt.Errorf("上传文件到 MinIO 失败: %w", err)
	}

	// 返回 file_key（相对路径），供数据库存储
	return c.GetPath(objectName), nil
}

// UploadMusic 上传音频文件到 MinIO，返回 file_key(相对路径)、时长(如"3:45")和错误
func (c *Client) UploadMusic(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, string, error) {
	// 归一化对象名称
	objectName = c.NormalizeObjectName(objectName)

	// 1. 创建临时文件，用于存储上传内容
	// GetAudioDuration 目前是“按扩展名”选择解析器（mp3/mp4/m4a），
	// 所以临时文件必须带上正确的扩展名，否则会被识别为“不支持的音频格式”。
	ext := strings.ToLower(filepath.Ext(objectName))
	if ext == "" {
		ext = ExtByContentType(contentType)
	}
	if ext == "" {
		// 兜底：即使无法判断，也要有扩展名，避免 filepath.Ext 为空
		ext = ".tmp"
	}

	tmpFile, err := os.CreateTemp("", "minio-music-*"+ext)
	if err != nil {
		return "", "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	// 确保最终删除临时文件
	defer func() { _ = os.Remove(tmpPath) }()

	// 2. 将上传内容写入临时文件
	written, err := io.Copy(tmpFile, reader)
	if err != nil {
		_ = tmpFile.Close()
		return "", "", fmt.Errorf("写入临时文件失败: %w", err)
	}
	// 关闭文件，确保数据落盘
	if err := tmpFile.Close(); err != nil {
		return "", "", fmt.Errorf("关闭临时文件失败: %w", err)
	}

	// 3. 获取音频时长
	duration, err := GetAudioDuration(tmpPath)
	if err != nil {
		return "", "", fmt.Errorf("获取音频时长失败: %w", err)
	}

	// 4. 重新打开临时文件，用于上传
	fileReader, err := os.Open(tmpPath)
	if err != nil {
		return "", "", fmt.Errorf("重新打开临时文件失败: %w", err)
	}
	defer func() { _ = fileReader.Close() }()

	// 5. 上传到 MinIO，使用实际写入的字节数作为 size（当原 size 为 -1 时）
	uploadSize := size
	if size < 0 {
		uploadSize = written
	}
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	// 小于 5MB 的文件不分片
	if uploadSize < 5*1024*1024 {
		opts.PartSize = 0
	}

	_, err = c.client.PutObject(ctx, c.bucketName, objectName, fileReader, uploadSize, opts)
	if err != nil {
		return "", "", fmt.Errorf("上传文件到 MinIO 失败: %w", err)
	}

	// 6. 生成并返回 file_key 和文件名
	fileKey := c.GetPath(objectName)
	return fileKey, duration, nil
}

// GetPath 生成对象的相对路径
func (c *Client) GetPath(objectName string) string {
	return fmt.Sprintf("/%s/%s", c.bucketName, objectName)
}

// GetFileURL 将 file_key 转换为完整的访问 URL
// 输入: /core-coach/images/20260524/abc.png
// 输出: http://118.31.10.161:9000/core-coach/images/20260524/abc.png
func (c *Client) GetFileURL(fileKey string) string {
	return c.baseURL + fileKey
}

// GetBucketName 返回 Bucket 名称
func (c *Client) GetBucketName() string {
	return c.bucketName
}

// PresignedGetURL 生成预签名下载链接
func (c *Client) PresignedGetURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = 1 * time.Hour
	}
	// 归一化对象名称，去除可能的桶名和前导斜杠
	objectName = c.NormalizeObjectName(objectName)

	presignedURL, err := c.client.PresignedGetObject(ctx, c.bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("生成预签名 URL 失败: %w", err)
	}
	return presignedURL.String(), nil
}

// Delete 删除 MinIO 中的文件
func (c *Client) Delete(ctx context.Context, objectName string) error {
	if objectName == "" {
		return fmt.Errorf("对象名称不能为空")
	}

	// 兼容传入完整 URL 的情况，统一提取 path 再删除。
	if u, err := url.Parse(objectName); err == nil && u.Scheme != "" && u.Host != "" {
		objectName = u.Path
	}

	objectName = c.NormalizeObjectName(objectName)
	if objectName == "" {
		return fmt.Errorf("对象名称不能为空")
	}

	if err := c.client.RemoveObject(ctx, c.bucketName, objectName, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("删除 MinIO 文件失败: %w", err)
	}

	_, err := c.client.StatObject(ctx, c.bucketName, objectName, minio.StatObjectOptions{})
	if err == nil {
		return fmt.Errorf("删除 MinIO 文件失败: 对象仍然存在")
	}

	resp := minio.ToErrorResponse(err)
	if resp.Code == "NoSuchKey" || resp.Code == "NoSuchObject" || resp.Code == "NoSuchBucket" {
		return nil
	}
	return fmt.Errorf("校验 MinIO 文件删除结果失败: %w", err)
}

// NormalizeObjectName 归一化对象名称，去除前导斜杠和可能的桶名
func (c *Client) NormalizeObjectName(objectName string) string {
	// 1. 去除前导斜杠
	objectName = strings.TrimPrefix(objectName, "/")

	// 2. 如果以桶名开头，则去除桶名
	bucketPrefix := c.bucketName + "/"
	if strings.HasPrefix(objectName, bucketPrefix) {
		objectName = strings.TrimPrefix(objectName, bucketPrefix)
	}

	return objectName
}

// ParseFileKey 从预签名 URL 中提取 file_key（相对路径）
// 输入：http://127.0.0.1:9000/core-coach/common/20260519/xxx.png?X-Amz-...
// 输出：/core-coach/common/20260519/xxx.png
func (c *Client) ParseFileKey(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("解析 URL 失败: %w", err)
	}
	// 获取路径部分并归一化
	path := strings.TrimLeft(u.Path, "/")
	if path == "" {
		return "", fmt.Errorf("URL 中未找到 file_key")
	}
	return "/" + path, nil
}

// ExtByContentType extByContentType 获取文件扩展名
func ExtByContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/wav", "audio/x-wav", "audio/wave":
		return ".wav"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

// GetAudioDuration 获取音频时长（专门解析 mp3 / mp4 / m4a）
//
// 说明：
// - 纯 Go 解析：不依赖服务器安装 ffprobe/ffmpeg，便于部署
// - mp3：扫描帧头累计采样数计算时长（对 VBR/CBR 均可用，但需要遍历文件）
// - mp4/m4a：读取 mp4 box（mvhd）计算时长
func GetAudioDuration(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	var d time.Duration
	var err error
	switch ext {
	case ".mp3":
		d, err = getMP3Duration(filePath)
	case ".mp4", ".m4a":
		d, err = getMP4Duration(filePath)
	default:
		return "", fmt.Errorf("不支持的音频格式: %s（仅支持 mp3/mp4/m4a）", ext)
	}
	if err != nil {
		return "", err
	}

	totalSeconds := int(d.Round(time.Second).Seconds())
	if totalSeconds <= 0 {
		return "", fmt.Errorf("解析到的时长无效: %v", d)
	}
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%d:%02d", minutes, seconds), nil
}

// getMP3Duration 获取mp3文件时长
func getMP3Duration(filePath string) (time.Duration, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	dec := mp3dec.NewDecoder(bufio.NewReaderSize(f, 64*1024))
	var frame mp3dec.Frame
	var skipped int

	var total time.Duration
	for {
		if err := dec.Decode(&frame, &skipped); err != nil {
			if err == io.EOF {
				break
			}
			return 0, fmt.Errorf("解析 mp3 帧失败: %w", err)
		}

		// 直接使用库内计算好的每帧时长（避免依赖 SampleRate()/SamplesPerFrame() 等版本差异）
		d := frame.Duration()
		if d <= 0 {
			// 极端情况下 Duration 可能为 0（非法帧等），这里跳过该帧
			continue
		}
		total += d
	}

	if total <= 0 {
		return 0, fmt.Errorf("无法解析 mp3 时长")
	}
	return total, nil
}

func getMP4Duration(filePath string) (time.Duration, error) {
	f, err := mp4.ReadMP4File(filePath)
	if err != nil {
		return 0, fmt.Errorf("解析 mp4 失败: %w", err)
	}
	if f == nil || f.Moov == nil || f.Moov.Mvhd == nil {
		return 0, fmt.Errorf("mp4 缺少 moov/mvhd box")
	}

	ts := f.Moov.Mvhd.Timescale
	dur := f.Moov.Mvhd.Duration
	if ts == 0 || dur == 0 {
		return 0, fmt.Errorf("mp4 mvhd 时长信息为空")
	}

	sec := float64(dur) / float64(ts)
	return time.Duration(sec * float64(time.Second)), nil
}
