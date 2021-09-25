package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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

	getSecret()

	log.WithField("status", "success").Debug("initialize")
}

func main() {

	data := fetchWeather(viper.GetString("WEATHER-API"))
	db := connectRDS()
	createData(db, data)
	readData(db)
}

type openWeatherResp struct {
	Main struct {
		Temperature float64 `json:"temp"`
		Humidity    int64   `json:"humidity"`
	} `json:"main"`
}

func fetchWeather(token string) (output openWeatherResp) {
	log.WithField("status", "starting").Info("fetchWeather")
	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?units=metric&q=%s&appid=%s",
		"Tangerang Selatan",
		token)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return
	}

	err = json.Unmarshal(body, &output)
	if err != nil {
		log.Fatalln(err)
		return
	}
	log.WithField("status", "success").Info("fetchWeather")
	return
}

func connectRDS() (db *sql.DB) {
	log.WithField("status", "starting").Info("connectRDS")

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s",
		viper.GetString("DB-USER"),
		viper.GetString("DB-PASSWORD"),
		viper.GetString("DB-HOST"),
		viper.GetString("DB-DEFAULT"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	log.WithField("status", "success").Info("connectRDS")
	return
}

func createData(db *sql.DB, data openWeatherResp) {
	insertStmt := fmt.Sprintf(`INSERT INTO weather(time, temp, humidity) VALUES ('%s', %v, %d)`,
		time.Now().Format("2006-01-02 15:04:05"),
		data.Main.Temperature,
		data.Main.Humidity)

	_, err := db.Exec(insertStmt)
	if err != nil {
		log.Fatalln(err)
	}

	return
}

func readData(db *sql.DB) {
	type data struct {
		ID       int
		Time     string
		Temp     float64
		Humidity int64
	}

	rows, err := db.Query("select * from weather")
	if err != nil {
		log.Fatalln(err)
	}

	defer rows.Close()
	for rows.Next() {
		m := data{}
		err := rows.Scan(&m.ID, &m.Time, &m.Temp, &m.Humidity)
		if err != nil {
			log.Fatalln(err)
		}
		log.WithFields(log.Fields{
			"temp":     m.Temp,
			"humadity": m.Humidity,
			"time":     m.Time,
		}).Info("weather data")
	}
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

	secretDBName := "test/AppDemo/mysql"
	secretWeatherName := "test/AppDemo/weather"

	region := "ap-southeast-1"

	svc := secretsmanager.New(session.New(),
		aws.NewConfig().WithRegion(region))

	//get db
	inputDb := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretDBName),
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
		SecretId: aws.String(secretWeatherName),
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
