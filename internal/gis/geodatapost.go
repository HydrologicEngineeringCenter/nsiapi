package gis

import (
	"errors"
	"io/ioutil"

	"github.com/google/uuid"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/utils"
	"github.com/labstack/echo"
	ogr "github.com/lukeroth/gdal"
)

type GeodataPost struct {
	GisFileName     string
	GDALDriverName  string
	Guid            string
	ds              ogr.DataSource
	EchoContext     echo.Context
	TempStoragePath string
}

func (gd *GeodataPost) Close() {
	gd.ds.Destroy()
}

func (gd *GeodataPost) GetGeometryAsWkb() (*[]uint8, error) {
	layer := gd.ds.LayerByIndex(0)
	feature := layer.NextFeature()
	defer feature.Destroy()
	fg := feature.Geometry()
	gdalwkb, err := fg.ToWKB()
	if err != nil {
		return nil, err
	}
	return &gdalwkb, nil
}

func (gd *GeodataPost) GetGeometry() (*ogr.Geometry, error) {
	layer := gd.ds.LayerByIndex(0)
	feature := layer.NextFeature()
	defer feature.Destroy()
	fg := feature.Geometry()
	filterGeom := fg.Clone()
	return &filterGeom, nil
}

func (gd *GeodataPost) Open() error {
	driver := ogr.OGRDriverByName(gd.GDALDriverName)
	ds, ok := driver.Open(gd.GisFileName, 0)
	if ok {
		gd.ds = ds
		return nil
	} else {
		return errors.New("Unable to open gis file data source")
	}
}

func (gd *GeodataPost) OpenFromBody() error {
	bodyBytes, err := ioutil.ReadAll(gd.EchoContext.Request().Body)
	if err != nil {
		return err
	}
	geojson := string(bodyBytes)
	if ds, ok := ogr.OpenDataSource(geojson, 0); ok {
		gd.ds = ds
		return nil
	} else {
		return errors.New("Unable to open gis payload in post body")
	}
}

func (gd *GeodataPost) HasFile() (bool, error) {
	file, err := gd.EchoContext.FormFile("file")
	if err != nil {
		return false, err
	}
	return file != nil, err
}

func (gd *GeodataPost) ExtractFile() error {
	file, err := gd.EchoContext.FormFile("file")
	if err != nil {
		return err
	}
	uuid, _ := uuid.NewUUID()
	tempname := uuid.String()
	gd.Guid = tempname
	newfile, err := utils.CopyPostFileToTemp(gd.TempStoragePath, tempname, file)
	if err != nil {
		return err
	}
	files, err := utils.Unzip(newfile, gd.TempStoragePath+tempname)
	if err != nil {
		return err
	}
	gisFiletype, gisFileName, err := utils.GetGisFileType(files)
	gd.GisFileName = gisFileName
	if err != nil {
		return err
	}
	ogrDriverName, err := utils.GetGdalDriverName(gisFiletype)
	gd.GDALDriverName = ogrDriverName
	if err != nil {
		return err
	}
	return nil
}

func geometryProcessor(fg *ogr.Geometry) {
	fg.Buffer(0, 1)
	sr := ogr.CreateSpatialReference("")
	sr.FromEPSG(4326)
	fg.TransformTo(sr)
}
