package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.WithField("status", "starting").Debug("initialize")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	errConfig := viper.ReadInConfig()
	if errConfig != nil {
		log.Fatalln(errConfig)
	}

	flag.String("mode", "", "input your text")
	flag.String("text", "", "input your text")
	flag.String("ciphertext", "", "input your text")
	flag.String("datakey", "", "input your text")
	flag.String("output", "", "input your text")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	log.WithField("status", "success").Debug("initialize")
}

func main() {
	log.WithField("region", viper.GetString("region")).Info("region")
	if viper.Get("mode") == "encrypt" {
		dataKey := generateDataKey()
		ciphertext, _ := encrypt(
			viper.GetString("text"),
			dataKey.Plaintext)

		writeOutput(ciphertext,
			base64.StdEncoding.EncodeToString(dataKey.CiphertextBlob),
			viper.GetString("output"))

		log.Info("encrypt success")
	} else if viper.Get("mode") == "decrypt" {
		ciphertext, datakey := readData()
		datakeyByte, _ := base64.StdEncoding.DecodeString(datakey)
		dataKeyPlain := decryptDataKey(datakeyByte)

		plaintext, _ := decrypt(ciphertext, dataKeyPlain)

		log.Println(plaintext)
	}
}

func generateDataKey() *kms.GenerateDataKeyOutput {
	region := viper.GetString("REGION")
	svc := kms.New(session.New(),
		aws.NewConfig().WithRegion(region))

	input := &kms.GenerateDataKeyInput{
		KeyId:   aws.String(viper.GetString("USER-MASTER-KEY")),
		KeySpec: aws.String("AES_256"),
	}

	result, err := svc.GenerateDataKey(input)
	if err != nil {
		log.Fatalln(err)

	}

	return result

}

func decryptDataKey(datakey []byte) (dataKeyPlain []byte) {
	region := viper.GetString("REGION")
	svc := kms.New(session.New(),
		aws.NewConfig().WithRegion(region))
	input := &kms.DecryptInput{
		CiphertextBlob: datakey,
		KeyId:          aws.String(viper.GetString("USER-MASTER-KEY")),
	}
	result, err := svc.Decrypt(input)
	if err != nil {
		log.Fatalln(err)

	}
	dataKeyPlain = result.Plaintext
	return

}

func encrypt(plaintext string, key []byte) (ciphertext string, err error) {

	block, _ := aes.NewCipher(key)
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

func decrypt(cipherText string, key []byte) (plainText string, err error) {
	// prepare cipher
	block, err := aes.NewCipher(key)
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

func writeOutput(ciphertext, datakey, prefix string) {

	f, err := os.Create(fmt.Sprintf("%s-ciphertext.txt", prefix))

	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err2 := f.WriteString(ciphertext)

	if err2 != nil {
		log.Fatal(err2)
	}

	fkey, err := os.Create(fmt.Sprintf("%s-datakey.txt", prefix))

	if err != nil {
		log.Fatal(err)
	}

	defer fkey.Close()

	_, err4 := fkey.WriteString(datakey)

	if err4 != nil {
		log.Fatal(err4)
	}

}

func readData() (ciphertext, datakey string) {
	ciphertextData, err := os.ReadFile(viper.GetString("ciphertext"))
	if err != nil {
		log.Fatalln(err)
	}
	ciphertext = string(ciphertextData)

	datakeyData, err := os.ReadFile(viper.GetString("datakey"))
	if err != nil {
		log.Fatalln(err)
	}
	datakey = string(datakeyData)
	return
}
