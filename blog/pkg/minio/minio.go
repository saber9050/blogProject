package minio

import (
	"blog/pkg/config"
	"context"

	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
