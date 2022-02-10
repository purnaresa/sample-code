package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

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
	flag.String("text1", "", "input your text")
	flag.String("text2", "", "input your text")
	flag.String("ciphertext", "", "input your text")
	flag.String("id", "", "input your text")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	log.WithField("status", "success").Debug("initialize")
}

func main() {
	log.WithField("region", viper.GetString("region")).Info("region")
	if viper.Get("mode") == "enc" {
		dataKey := generateDataKey()
		ciphertextOne, _ := encrypt(
			viper.GetString("text1"),
			dataKey.Plaintext)
		ciphertextTwo, _ := encrypt(
			viper.GetString("text2"),
			dataKey.Plaintext)

		createOutput(
			viper.GetString("id"),
			ciphertextOne,
			ciphertextTwo,
			base64.StdEncoding.EncodeToString(dataKey.CiphertextBlob),
		)

		log.Info("encrypt complete")
	} else if viper.Get("mode") == "dec" {
		t := time.Now()
		obj := readObject(viper.GetString("ciphertext"))

		datakeyByte, _ := base64.StdEncoding.DecodeString(obj.DataKey)
		dataKeyPlain := decryptDataKey(datakeyByte)

		plaintextOne, _ := decrypt(obj.FieldOne, dataKeyPlain)
		plaintextTwo, _ := decrypt(obj.FieldOne, dataKeyPlain)

		log.WithFields(log.Fields{
			"Field One": plaintextOne,
			"Field Two": plaintextTwo,
		}).Info("decrypt complete")

		log.WithField("time(ms)", time.Since(t).Milliseconds()).Info("decrypt total time")
	}
}

func generateDataKey() *kms.GenerateDataKeyOutput {
	t := time.Now()
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
	lapse := time.Since(t).Milliseconds()
	log.WithField("time(ms)", lapse).Debug("generate data key success")
	return result
}

func decryptDataKey(datakey []byte) (dataKeyPlain []byte) {
	t := time.Now()
	region := viper.GetString("REGION")
	sess, _ := session.NewSession()
	svc := kms.New(sess,
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
	lapse := time.Since(t).Milliseconds()
	log.WithField("time(ms)", lapse).Debug("decrypt data key success")
	return
}

func encrypt(plaintext string, key []byte) (ciphertext string, err error) {
	t := time.Now()
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
	lapse := time.Since(t).Microseconds()
	log.WithField("time(us)", lapse).Debug("encrypt field success")
	return
}

func decrypt(cipherText string, key []byte) (plainText string, err error) {
	t := time.Now()
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

	lapse := time.Since(t).Microseconds()
	log.WithField("time(us)", lapse).Debug("decrypt field success")
	return
}

type SecureObject struct {
	ID       string `json:"id"`
	FieldOne string `json:"field-one"`
	FieldTwo string `json:"field-two"`
	DataKey  string `json:"datakey"`
}

func readObject(source string) (obj SecureObject) {
	objSource, err := os.ReadFile(source)
	if err != nil {
		log.Fatalln(err)
	}

	errParse := json.Unmarshal(objSource, &obj)
	if errParse != nil {
		log.Fatalln(errParse)
	}

	return
}

func createOutput(id, ciphertextOne, cipherTextTwo, datakey string) {
	secObject := &SecureObject{
		ID:       id,
		FieldOne: ciphertextOne,
		FieldTwo: cipherTextTwo,
		DataKey:  datakey}

	secObjectString, err := json.Marshal(secObject)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(fmt.Sprintf("%s-encrypted.json", id))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err2 := f.WriteString(string(secObjectString))
	if err2 != nil {
		log.Fatal(err2)
	}
}
