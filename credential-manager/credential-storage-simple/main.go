package main

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
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

	log.WithField("status", "success").Debug("initialize")
}

func main() {
	getSecret()
	log.WithField("user", viper.GetString("DB-USER")).Info("db-credentials")
	log.WithField("host", viper.GetString("DB-HOST")).Info("db-credentials")
	log.WithField("api-key", viper.GetString("WEATHER-API")).Info("weather-api")
}

func getSecret() {
	type dbCredential struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		DBDefault string `json:"db-default"`
	}
	type weather struct {
		Api string `json:"api"`
	}

	secretDBPath := viper.GetString("DB-CREDENTIAL")
	secretWeatherPath := viper.GetString("WEATHER-CREDENTIAL")

	region := viper.GetString("REGION")

	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(region))

	//get db
	inputDb := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretDBPath),
	}

	resultDb, err := svc.GetSecretValue(inputDb)
	if err != nil {
		log.Fatalln(err)
	}
	db := dbCredential{}
	err = json.Unmarshal([]byte(*resultDb.SecretString), &db)
	if err != nil {
		log.Fatalln(err)
	}
	//

	// get weather
	inputWeather := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretWeatherPath),
	}

	resultWeather, err := svc.GetSecretValue(inputWeather)
	if err != nil {
		log.Fatalln(err)
	}
	w := weather{}
	err = json.Unmarshal([]byte(*resultWeather.SecretString), &w)
	if err != nil {
		log.Fatalln(err)
	}
	//

	viper.Set("WEATHER-API", w.Api)
	viper.Set("DB-USER", db.Username)
	viper.Set("DB-PASSWORD", db.Password)
	viper.Set("DB-HOST", fmt.Sprintf("%s:%d", db.Host, db.Port))
	viper.Set("DB-DEFAULT", db.DBDefault)

}
