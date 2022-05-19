package gis

import (
	"fmt"
	"log"
	"strconv"
	s "strings"

	"github.com/hydrologicengineeringcenter/nsiapi/internal/stores"

	ogr "github.com/lukeroth/gdal"
	"github.com/paulmach/orb"
)

var featureReportNumber int = 10000

type Db2FileEtl struct {
	DbDriver     string
	UrlTemplate  string
	DbDialect    string
	DbOptions    []string
	User         string
	Pass         string
	Host         string
	Db           string
	Sql          string
	GeomFilter   *ogr.Geometry
	FileDriver   string
	FileOut      string
	NewLayerName string
	Guid         string
}

type ProgressReporter interface {
	Message(msg string, count int)
}

type ConsoleReporter struct{}

func (cr ConsoleReporter) Message(msg string, count int) {
	if count > 0 {
		if count%featureReportNumber == 0 {
			log.Printf("%s number %d\n", msg, count)
		}
	} else {
		log.Println(msg)
	}
}

func RunDb2FileEtl(etl *Db2FileEtl, tempStore *stores.TempStore, tempStoragePath string, reporter ProgressReporter) {
	defer func() {
		if etl.GeomFilter != nil {
			etl.GeomFilter.Destroy()
		}
	}()
	tempStore.PutStatus(etl.Guid, "Processing")
	driverIn := ogr.OGRDriverByName(etl.DbDriver)
	dburl := fmt.Sprintf(etl.UrlTemplate, etl.Host, etl.Db, etl.User, etl.Pass)
	dsIn, okIn := driverIn.Open(dburl, 0)
	defer func() {
		dsIn.Destroy()
		if r := recover(); r != nil {
			reporter.Message(fmt.Sprintf("Recovered from %s\n", r), 0)
		}
	}()
	if !okIn {
		reporter.Message("Unable to open DB datasource", 0)
	} else {
		reporter.Message("Opened DB datasource", 0)
		driverOut := ogr.OGRDriverByName(etl.FileDriver)
		dsOut, okOut := driverOut.Create(tempStoragePath+etl.FileOut, []string{})
		defer dsOut.Destroy()

		if !okOut {
			reporter.Message(fmt.Sprintf("Unable to open ouput datasource:%s", tempStoragePath+etl.FileOut), 0)
		} else {
			var layer ogr.Layer
			if etl.GeomFilter == nil {
				filter := ogr.Create(ogr.GT_Null)
				layer = dsIn.ExecuteSQL(etl.Sql, filter, etl.DbDialect)
			} else {
				layer = dsIn.ExecuteSQL(etl.Sql, *etl.GeomFilter, etl.DbDialect)
			}

			if !layer.IsNull() {
				defer dsIn.ReleaseResultSet(layer)
				copyFeatures(layer, dsOut, etl, reporter)
			} else {
				reporter.Message("Unable to Retrieve Layer", 0)
			}
		}
	}
	tempStore.PutStatus(etl.Guid, "Completed")
}

func copyFeatures(layer ogr.Layer, dsOut ogr.DataSource, etl *Db2FileEtl, reporter ProgressReporter) {
	sr := layer.SpatialReference()
	newLayer := dsOut.CreateLayer(etl.NewLayerName, sr, ogr.GT_Point, etl.DbOptions) //forcing point data type.  source type (using lyaer.type()) from postgis was a generic geometry
	if !newLayer.IsNull() {
		layerDef := layer.Definition()
		for i := 0; i < layerDef.FieldCount(); i++ {
			newLayer.CreateField(layerDef.FieldDefinition(i), false)
		}
		isReading := true
		var c int = 0
		for c = 1; isReading; c++ {
			func() {
				feature := layer.NextFeature()
				if feature != nil {
					defer feature.Destroy()
					newLayer.Create(*feature)
					reporter.Message(etl.FileOut+": Copying feature ", c)
				} else {
					isReading = false
				}
			}()
		}
		reporter.Message(fmt.Sprintf("%s: Completed Export of %d features", etl.FileOut, c), 0)
	} else {
		reporter.Message("Unable to create output layer", 0)
	}
}

func StringToCoords(bboxParam string) (*[]float64, error) {
	bboxParamArray := s.Split(bboxParam, ",")
	coords := []float64{}
	for i, _ := range bboxParamArray {
		f, err := strconv.ParseFloat(bboxParamArray[i], 64)
		if err != nil {
			return nil, err
		}
		coords = append(coords, f)
	}
	return &coords, nil
}

//@TODO is there a way to avoid the dereference copy of coords???
func CoordsToLineString(coords *[]float64) orb.LineString {
	lineString := orb.LineString{}
	c := *coords
	for i := 0; i < len(c); i += 2 {
		lineString = append(lineString, orb.Point{c[i], c[i+1]})
	}
	return lineString
}

func LineStringToPoly(lineString orb.LineString) *orb.Polygon {
	ring := orb.Ring(lineString)
	p := make(orb.Polygon, 0, 1)
	p = append(p, ring)
	return &p
}

/////////////////////
////////////////////

const (
	WKB      = iota
	GDALGEOM = iota
)
