package configuration

import (
	"testing"
    //"../configuration"
    "fmt"
    //"encoding/hex"
//    "time"
)

func TestMain(m *testing.M) {
	//msg := "Hello, world!"
    //msg := "ABC"
	//ciphertext := encrypt(msg)
    ciphertext := "3dd8c12c2d249ccee9f99101afae0ec0bf9f77200c4920911722fef398707afa5efcf82d07aed3bae9059041ff1bb2fef4bfd27d9e9d"
	//fmt.Printf("Encrypted: %x\n", ciphertext)
    fmt.Println("Encrypted:", ciphertext)
	plaintext := decrypt(ciphertext)
	fmt.Println(plaintext)
}
