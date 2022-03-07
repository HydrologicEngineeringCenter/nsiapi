package main

import (
	"log"

	"di2e.net/cwbi/nsiv2-api/config"
	"di2e.net/cwbi/nsiv2-api/stores"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

const apiprefix = "/nsiapi"

func main() {
	config := config.GetConfig()

	dataStore, err := stores.InitDbStore(config)
	if err != nil {
		log.Printf("Error initializing data store: %s. Continuing with startup.", err)
	}

	tempStore, err := stores.InitTempStore(config)
	if err != nil {
		log.Fatalf("Error initializing local temparary data store: %s. Shutting down.", err)
	}

	e := echo.New()
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10, // 1 KB
	}))
	e.Use(middleware.Logger())

	api := ApiHandler{
		tempStore,
		dataStore,
		config,
	}

	e.GET(apiprefix+"/home", api.ApiHome)
	e.GET(apiprefix+"/structures", api.GetStructures)
	e.GET(apiprefix+"/structure/:structureId", api.GetStructure)
	e.POST(apiprefix+"/structures", api.StructuresFromUpload)
	e.GET(apiprefix+"/hexbins/:dataset", api.GetHexbins)
	e.GET(apiprefix+"/export", api.CreateExport)
	e.GET(apiprefix+"/export/:uuid", api.GetExport)
	e.GET(apiprefix+"/export/:uuid/status", api.GetStatus)
	e.POST(apiprefix+"/export", api.ExportFromUpload)
	e.GET(apiprefix+"/stats", api.GetStats)
	e.POST(apiprefix+"/stats", api.StatsFromUpload)
	e.GET(apiprefix+"/export/state/:file", api.DownloadFileDataset)

	e.Debug = config.Debug

	e.Logger.Fatal(e.Start(":" + config.Port))
}
