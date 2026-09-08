package utils

import (
	"testing"
	"time"
)

func TestGetCurrentTimestamp(t *testing.T) {
	before := time.Now().Unix()
	got := GetCurrentTimestamp()
	after := time.Now().Unix()

	if got < before || got > after {
		t.Errorf("时间戳 %d 不在合理区间 [%d, %d]", got, before, after)
	}
}

func TestGetCurrentMilliTimestamp(t *testing.T) {
	before := time.Now().UnixMilli()
	got := GetCurrentMilliTimestamp()
	after := time.Now().UnixMilli()

	if got < before || got > after {
		t.Errorf("毫秒时间戳 %d 不在合理区间 [%d, %d]", got, before, after)
	}
	// 毫秒级时间戳应比秒级大三个数量级
	if got < GetCurrentTimestamp()*1000 {
		t.Errorf("毫秒时间戳 %d 异常偏小", got)
	}
}

func TestFormatAndParseTime(t *testing.T) {
	// 注意：ParseTime 基于 time.Parse，解析结果是 UTC 时间，与本地时区无关。
	// 与该实现保持一致，测试使用 UTC 基准，避免 CI 上因 TZ 不同而失败。
	ref := time.Date(2026, 9, 8, 10, 30, 0, 0, time.UTC)

	formatted := FormatTime(ref)
	if formatted != "2026-09-08 10:30:00" {
		t.Errorf("FormatTime = %q，期望 %q", formatted, "2026-09-08 10:30:00")
	}

	parsed, err := ParseTime(formatted)
	if err != nil {
		t.Fatalf("ParseTime 失败: %v", err)
	}
	if !parsed.Equal(ref) {
		t.Errorf("往返转换不一致：got %v, want %v", parsed, ref)
	}
}

// 锁定时区语义：保证任何 TZ 环境下的 CI 行为一致
func TestParseTimeReturnsUTC(t *testing.T) {
	parsed, err := ParseTime("2026-09-08 10:30:00")
	if err != nil {
		t.Fatalf("ParseTime 失败: %v", err)
	}
	if parsed.Location() != time.UTC {
		t.Errorf("解析结果时区 = %v，期望 UTC", parsed.Location())
	}
	if parsed.Format("2006-01-02 15:04:05") != "2026-09-08 10:30:00" {
		t.Error("格式化后应与输入字符串一致")
	}
}

func TestParseTimeInvalid(t *testing.T) {
	if _, err := ParseTime("not-a-time"); err == nil {
		t.Error("非法时间字符串应返回错误")
	}
}
