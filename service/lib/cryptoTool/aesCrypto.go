package cryptoTool

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
)

// GenerateKeyFromPassword 用 SHA256 将任意用户密码派生为 32 字节 AES-256 Key
func GenerateKeyFromPassword(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

// EncryptData 使用 AES-256-GCM 算法加密数据
func EncryptData(data []byte, password string) ([]byte, error) {
	if len(password) == 0 {
		return nil, errors.New("加密密码不能为空")
	}
	key := GenerateKeyFromPassword(password)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// 拼接: nonce + 密文
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// DecryptData 使用 AES-256-GCM 算法解密数据，密码错误时返回错误
func DecryptData(ciphertext []byte, password string) ([]byte, error) {
	if len(password) == 0 {
		return nil, errors.New("解密密码不能为空")
	}
	key := GenerateKeyFromPassword(password)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("密文数据长度不合法或已损坏")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, errors.New("解密失败：密码错误或数据已被篡改")
	}

	return plaintext, nil
}

// EncryptAES256GCM 便捷别名
func EncryptAES256GCM(data []byte, password string) ([]byte, error) {
	return EncryptData(data, password)
}

// DecryptAES256GCM 便捷别名
func DecryptAES256GCM(ciphertext []byte, password string) ([]byte, error) {
	return DecryptData(ciphertext, password)
}

