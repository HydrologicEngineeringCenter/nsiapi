package main

import (
	"log"

	"github.com/hydrologicengineeringcenter/nsiapi/internal/config"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/handlers"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/stores"
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

	api := handlers.ApiHandler{
		TempStore: tempStore,
		DataStore: dataStore,
		Config:    config,
	}

	e.GET(apiprefix+"/home", api.ApiHome)

	// Default to primary dataset if not specified
	e.GET(apiprefix+"/structures", api.GetStructures)
	e.GET(apiprefix+"/structure/:structureId", api.GetStructure)
	// e.POST(apiprefix+"/structures", api.StructuresFromUpload)

	// dataset/version/quality can be from pathing or query after ?
	e.GET(apiprefix+"/structures/dataset/:dataset", api.GetStructures)
	e.GET(apiprefix+"/structure/:structureId/dataset/:dataset", api.GetStructure)
	// upload needs to be dataset specific / maybe with auth
    // GEOJSON / GEOPACKAGE?
	e.POST(apiprefix+"/structures/dataset/:dataset", api.StructuresFromUpload)

	// need an endpoint to query available dataset for user / maybe with auth
	// add endpoint for info about schema and associated fields

	e.GET(apiprefix+"/hexbins/:dataset", api.GetHexbins)
	e.GET(apiprefix+"/export", api.CreateExport)
	e.GET(apiprefix+"/export/:uuid", api.GetExport)
	e.GET(apiprefix+"/export/:uuid/status", api.GetStatus)
	e.POST(apiprefix+"/export", api.ExportFromUpload)

	// deprecate stats
	e.GET(apiprefix+"/stats", api.GetStats)
	e.POST(apiprefix+"/stats", api.StatsFromUpload)

	e.GET(apiprefix+"/export/state/:file", api.DownloadFileDataset)

	e.Debug = config.Debug

	e.Logger.Fatal(e.Start(":" + config.Port))
}
