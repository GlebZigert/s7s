package configuration

import (
	"io"
    "crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
    "encoding/hex"
)

var (
    // TODO: update salt && passwd
    msg_salt = []byte{12, 48, 16, 2, 19, 14, 99, 170, 33, 17, 81, 140, 0 , 115, 211, 38, 3, 46, 223, 224, 29, 134, 122, 40, 101}
    passwd = []byte{75, 7, 99, 19, 33, 136, 123, 187, 131, 52, 249, 9, 173, 52, 97, 238, 53, 112, 0, 175, 221, 34, 50, 13, 90, 208, 78}
)

func createHash(key []byte) []byte{
	hasher := md5.New()
	hasher.Write(key)
	return hasher.Sum(nil)
}

func _encrypt(data []byte, pass []byte) []byte {
	block, _ := aes.NewCipher(createHash(pass))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func _decrypt(data []byte, pass []byte) []byte {
	key := createHash(pass)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}
	return plaintext
}

func encrypt(text string) string{
    payload := append(msg_salt, text...)
    return hex.EncodeToString(_encrypt(payload, passwd))
}

func decrypt(cipher string) string {
    b, _ := hex.DecodeString(cipher)
    text := _decrypt(b, passwd)
	return string(text[len(msg_salt):])
}
