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

func _encrypt(data []byte, pass []byte) (bbb []byte, err error) {
	block, _ := aes.NewCipher(createHash(pass))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func _decrypt(data []byte, pass []byte) (bbb []byte, err error) {
	key := createHash(pass)
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return
	}
	return plaintext, nil
}

func encrypt(text string) (string, error) {
    payload := append(msg_salt, text...)
    cipher, err := _encrypt(payload, passwd)
    if nil != err {
        return "", err
    }
    return hex.EncodeToString(cipher), err
}

func decrypt(cipher string) (string, error) {
    b, _ := hex.DecodeString(cipher)
    text, err := _decrypt(b, passwd)
    if nil != err {
        return "", err
    }
	return string(text[len(msg_salt):]), err
}
