package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
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
	authToken := getRDSToken()
	db := connectRDS(authToken)
	testQuery(db)
}

func getRDSToken() (authenticationToken string) {
	log.WithField("status", "starting").Info("getRDSToken")
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.WithField("status", "error").Panicln(err)
	}

	authenticationToken, err = auth.BuildAuthToken(
		context.TODO(),
		viper.GetString("RDSHOST"),
		viper.GetString("AWSREGION"),
		viper.GetString("RDSUSER"),
		cfg.Credentials,
	)
	if err != nil {
		log.WithField("status", "error").Panicln(err)
	}
	log.WithField("status", "success").Info("getRDSToken")
	return
}

func connectRDS(authToken string) (db *sql.DB) {
	log.WithField("status", "starting").Info("connectRDS")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?tls=rds&allowCleartextPasswords=true",
		viper.GetString("RDSUSER"),
		authToken,
		viper.GetString("RDSHOST"),
		viper.GetString("DBDEFAULT"),
	)

	tlsConfig := registerRDSMysqlCerts()
	mysql.RegisterTLSConfig("rds", &tlsConfig)
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

func registerRDSMysqlCerts() (tlsConfig tls.Config) {
	log.WithField("status", "starting").Info("connectRDS.registerRDSMysqlCerts")
	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile("global-bundle.pem") //from https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem
	if err != nil {
		log.Fatalln(err)
	}
	ok := rootCertPool.AppendCertsFromPEM(pem)
	if !ok {
		log.Fatalln("couldn't append certs from pem")
	}
	log.WithField("status", "success").Info("connectRDS.registerRDSMysqlCerts")
	tlsConfig = tls.Config{
		RootCAs:            rootCertPool,
		InsecureSkipVerify: true}
	return
}

func testQuery(db *sql.DB) {
	type movieType struct {
		ID     int
		Title  string
		Genre  string
		Rating int
	}

	rows, err := db.Query("select * from movie")
	if err != nil {
		log.Fatalln(err)
	}

	defer rows.Close()
	for rows.Next() {
		m := movieType{}
		err := rows.Scan(&m.ID, &m.Title, &m.Genre, &m.Rating)
		if err != nil {
			log.Fatalln(err)
		}
		log.Debug(m)
	}
}
