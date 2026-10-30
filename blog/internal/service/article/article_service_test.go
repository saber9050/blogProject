package article

import (
	"reflect"
	"testing"
)

// TestExtractImageURLs 图片地址提取：Markdown 与 HTML 两套语法都要认。
//
// 背景：正文自 2026-10-29 起存 Markdown，图片写成 `![](url)`。旧实现只匹配
// HTML 的 `<img src>`，导致正文图片不被记进 article_images、删文章时 MinIO 里
// 的文件成了孤儿。两套语法必须并存——Markdown 里内嵌 HTML 是合法写法。
func TestExtractImageURLs(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "Markdown 基本",
			content: "![示意图](https://cdn.example.com/a/1.png)",
			want:    []string{"https://cdn.example.com/a/1.png"},
		},
		{
			name:    "Markdown alt 为空",
			content: "![](https://cdn.example.com/a/2.png)",
			want:    []string{"https://cdn.example.com/a/2.png"},
		},
		{
			name:    "Markdown 带 title",
			content: `![示意图](https://cdn.example.com/a/3.png "这张图的标题")`,
			want:    []string{"https://cdn.example.com/a/3.png"},
		},
		{
			name:    "Markdown 尖括号包裹",
			content: "![图](<https://cdn.example.com/a/4.png>)",
			want:    []string{"https://cdn.example.com/a/4.png"},
		},
		{
			name:    "HTML img 仍然兼容",
			content: `<p>文字</p><img src="https://cdn.example.com/a/5.png" alt="x">`,
			want:    []string{"https://cdn.example.com/a/5.png"},
		},
		{
			name:    "两套语法混用",
			content: "开头\n\n![a](https://cdn.example.com/a/6.png)\n\n<img src=\"https://cdn.example.com/a/7.png\">\n\n结尾",
			want:    []string{"https://cdn.example.com/a/6.png", "https://cdn.example.com/a/7.png"},
		},
		{
			name:    "同一张图引用两次只算一次",
			content: "![a](https://cdn.example.com/a/8.png)\n\n再贴一遍：![b](https://cdn.example.com/a/8.png)",
			want:    []string{"https://cdn.example.com/a/8.png"},
		},
		{
			name:    "普通超链接不是图片",
			content: "详见 [官方文档](https://cdn.example.com/page.html) 和 [另一篇](/articles/3)",
			want:    nil,
		},
		{
			name:    "没有图片",
			content: "## 一、正文\n\n纯文字，一张图都没有。",
			want:    nil,
		},
		{
			name:    "图片夹在正文中间",
			content: "## 一、前言\n\n这段话后面跟一张图：\n\n![流程图](https://cdn.example.com/a/9.png)\n\n然后是结尾。",
			want:    []string{"https://cdn.example.com/a/9.png"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractImageURLs(c.content)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("extractImageURLs(%q)\n期望：%v\n实际：%v", c.content, c.want, got)
			}
		})
	}
}
