package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var dummyMasterKey = "04076d64bdb6fcf31706eea85ec98431"

func init() {
	log.SetLevel(log.DebugLevel)
	log.WithField("status", "success").
		Debug("initialize")

	flag.String("mode", "", "input your text")
	flag.String("text", "", "input your text")
	flag.String("ciphertext", "", "input your text")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

}

func main() {
	if viper.Get("mode") == "encrypt" {
		log.WithField("status", "start").
			Debug("encrypt")
		cipherText, _ := Encrypt(
			viper.GetString("text"),
			dummyMasterKey,
		)

		log.WithField("ciphertext", cipherText).
			Info("encrypt")
		log.WithField("status", "success").
			Debug("encrypt")

	} else if viper.Get("mode") == "decrypt" {
		log.WithField("decrypt", "start").
			Debug("encrypt")
		plaintext, _ := Decrypt(
			viper.GetString("ciphertext"),
			dummyMasterKey,
		)
		log.WithField("plaintext", plaintext).
			Info("decrypt")
		log.WithField("decrypt", "success").
			Debug("encrypt")
	}
}

func Encrypt(plaintext, key string) (ciphertext string, err error) {
	keyByte := []byte(key)
	block, _ := aes.NewCipher(keyByte)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return
	}

	ciphertextByte := gcm.Seal(
		nonce,
		nonce,
		[]byte(plaintext),
		nil)
	ciphertext = base64.StdEncoding.EncodeToString(ciphertextByte)

	return
}

func Decrypt(cipherText, key string) (plainText string, err error) {
	// prepare cipher
	keyByte := []byte(key)
	block, err := aes.NewCipher(keyByte)
	if err != nil {
		return
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}
	nonceSize := gcm.NonceSize()
	//

	// process ciphertext
	ciphertextByte, _ := base64.StdEncoding.DecodeString(cipherText)
	nonce, ciphertextByteClean := ciphertextByte[:nonceSize], ciphertextByte[nonceSize:]
	plaintextByte, err := gcm.Open(
		nil,
		nonce,
		ciphertextByteClean,
		nil)
	if err != nil {
		log.Println(err)
		return
	}
	plainText = string(plaintextByte)
	//
	return
}
