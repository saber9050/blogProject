package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

// 生成一对临时 RSA 密钥，并返回 PEM 编码的公钥
func generateTestPublicKeyPEM(t *testing.T) (string, *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成 RSA 密钥失败: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("序列化公钥失败: %v", err)
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	return pemStr, priv
}

func TestGenerateRandomHex(t *testing.T) {
	tests := []struct {
		name      string
		byteLen   int
		wantChars int
	}{
		{"16字节", 16, 32},
		{"32字节", 32, 64},
		{"0字节", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRandomHex(tt.byteLen)
			if err != nil {
				t.Fatalf("GenerateRandomHex(%d) 失败: %v", tt.byteLen, err)
			}
			if len(got) != tt.wantChars {
				t.Errorf("长度 = %d，期望 %d", len(got), tt.wantChars)
			}
			// 十六进制字符串只应包含 0-9a-f
			if strings.TrimLeft(got, "0123456789abcdef") != "" {
				t.Errorf("结果不是合法十六进制: %s", got)
			}
		})
	}
}

func TestLoadRSAPublicKey(t *testing.T) {
	pemStr, _ := generateTestPublicKeyPEM(t)

	pub, err := LoadRSAPublicKey(pemStr)
	if err != nil {
		t.Fatalf("加载合法公钥失败: %v", err)
	}
	if pub.Size() != 256 { // 2048 bit = 256 byte
		t.Errorf("密钥长度 = %d 字节，期望 256", pub.Size())
	}

	t.Run("非 PEM 格式", func(t *testing.T) {
		if _, err := LoadRSAPublicKey("not-a-pem-string"); err == nil {
			t.Error("非法 PEM 应返回错误")
		}
	})

	t.Run("PEM 格式正确但内容非法", func(t *testing.T) {
		bad := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("garbage")}))
		if _, err := LoadRSAPublicKey(bad); err == nil {
			t.Error("垃圾内容应返回错误")
		}
	})
}

func TestEncryptWithPublicKeyRoundTrip(t *testing.T) {
	pemStr, priv := generateTestPublicKeyPEM(t)

	pub, err := LoadRSAPublicKey(pemStr)
	if err != nil {
		t.Fatalf("加载公钥失败: %v", err)
	}

	plaintext := []byte(`{"password":"MyP@ssw0rd"}`)
	ciphertext, err := EncryptWithPublicKey(pub, plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if string(ciphertext) == string(plaintext) {
		t.Fatal("密文不应等于明文")
	}

	decrypted, err := priv.Decrypt(nil, ciphertext, nil)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("解密结果 = %q，期望 %q", decrypted, plaintext)
	}
}

func TestEncryptWithPublicKeyDifferentCiphertext(t *testing.T) {
	// PKCS1v15 带随机填充，同一明文两次加密结果必须不同
	pemStr, _ := generateTestPublicKeyPEM(t)
	pub, _ := LoadRSAPublicKey(pemStr)

	first, err := EncryptWithPublicKey(pub, []byte("same-plaintext"))
	if err != nil {
		t.Fatalf("第一次加密失败: %v", err)
	}
	second, err := EncryptWithPublicKey(pub, []byte("same-plaintext"))
	if err != nil {
		t.Fatalf("第二次加密失败: %v", err)
	}
	if string(first) == string(second) {
		t.Error("两次加密结果相同，说明未使用随机填充")
	}
}
