package config

import (
	"log"
	"os"
	"strconv"

	dq "github.com/usace/goquery"
)

type AppConfig struct {
	Dbport           string
	Dbuser           string
	Dbpass           string
	Dbhost           string
	Dbname           string
	DbMaxConnections int
	FeatureLimit     string
	TempStoragePath  string
	Port             string
	Debug            bool
	AwsBucket        string
	AwsPrefix        string
}

func GetConfig() AppConfig {
	appConfig := AppConfig{}
	appConfig.AwsBucket = os.Getenv("AWS_BUCKET")
	appConfig.AwsPrefix = os.Getenv("AWS_PREFIX")
	appConfig.Dbuser = os.Getenv("DBUSER")
	appConfig.Dbpass = os.Getenv("DBPASS")
	appConfig.Dbhost = os.Getenv("DBHOST")
	appConfig.Dbname = os.Getenv("DBNAME")
	appConfig.Dbport = os.Getenv("DBPORT")
	maxConnections, err := strconv.Atoi(os.Getenv("DBMAXCONNECTIONS"))
	log.Println(maxConnections)
	if err != nil || maxConnections == 0 {
		maxConnections = 10
	}
	appConfig.DbMaxConnections = maxConnections
	appConfig.FeatureLimit = os.Getenv("FEATURELIMIT")
	appConfig.TempStoragePath = os.Getenv("TEMPSTORAGEPATH")
	appConfig.Port = os.Getenv("PORT")
	debug := os.Getenv("DEBUG")
	if debug == "TRUE" {
		appConfig.Debug = true
	}
	return appConfig
}

func (c *AppConfig) Rdbmsconfig() dq.RdbmsConfig {
	return dq.RdbmsConfig{
		Dbuser:   c.Dbuser,
		Dbpass:   c.Dbpass,
		Dbhost:   c.Dbhost,
		Dbport:   c.Dbport,
		Dbname:   c.Dbname,
		DbDriver: "postgres",
		DbStore:  "pgx",
	}
}
