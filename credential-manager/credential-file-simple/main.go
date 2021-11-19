package main

import (
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
	log.WithField("user", viper.GetString("DB-USER")).Info("db-credentials")
	log.WithField("host", viper.GetString("DB-HOST")).Info("db-credentials")
	log.WithField("api-key", viper.GetString("DB-HOST")).Info("weather-api")
}
