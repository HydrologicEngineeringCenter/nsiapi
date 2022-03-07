package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"

	"di2e.net/cwbi/nsiv2-api/config"
	. "di2e.net/cwbi/nsiv2-api/stores"
	"github.com/google/uuid"
	"github.com/labstack/echo"

	//"github.com/paulmach/orb"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/paulmach/orb/encoding/wkb"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/project"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	s "strings"
)

var featureSeparator []byte = []byte(",")
var featureEnd []byte = []byte("}\n")
var objectEnd []byte = []byte("}")
var arrayStart []byte = []byte("[")
var arrayEnd []byte = []byte("]")
var featureCollectionStart []byte = []byte(`{"type": "FeatureCollection","features":`)
var validFipsLengths []int = []int{2, 5, 11, 12, 15}
var proptag string = "prop"

const featureTemplate = `{"type": "Feature","geometry": {"type": "Point","coordinates": [%f, %f]},"properties":`

/*
   Valid API structure output Formats:
	 fc = feature collection (default)
	 fa = feature array
	 fs = feature stream
*/

type ApiHandler struct {
	tempStore *TempStore
	dataStore *DbStore
	config    config.AppConfig
}

func (api *ApiHandler) ApiHome(c echo.Context) error {
	return c.String(http.StatusOK, "National Structures Inventory APIv2")
}

func (api *ApiHandler) GetStatus(c echo.Context) error {
	id := c.Param("uuid")
	status, err := api.tempStore.GetStatus(id)
	if err != nil {
		return err
	} else {
		return c.String(http.StatusOK, "{\"status\":\""+status+"\"}")
	}
}

func (api *ApiHandler) DownloadFileDataset(c echo.Context) error {
	file := c.Param("file")

	sess, awsSessErr := session.NewSession()
	if awsSessErr != nil {
		return awsSessErr
	}
	object := api.config.AwsPrefix + file
	log.Printf("Download request for %s\n", object)

	result, err := s3.New(sess).GetObject(&s3.GetObjectInput{
		Bucket: &api.config.AwsBucket,
		Key:    &object,
	})
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file))
	c.Response().Header().Set("Cache-Control", "no-store")

	_, copyErr := io.Copy(c.Response().Writer, result.Body)
	if copyErr != nil {
		return copyErr
	}
	return nil
}

func (api *ApiHandler) GetStructure(c echo.Context) error {
	fdId := c.Param("structureId")
	criteria := " where fd_id=$1"
	nsi := Nsi{}
	err := api.dataStore.Db.Get(&nsi, fmt.Sprintf("%s %s", NsiSelect, criteria), fdId)
	if err != nil {
		return err
	}
	feature := fmt.Sprintf(featureTemplate, nsi.X, nsi.Y)
	props, err := json.Marshal(nsi)
	return c.String(http.StatusOK, fmt.Sprintf("%s %s}", feature, props))
}

func (api *ApiHandler) GetStructures(c echo.Context) error {
	fips := c.QueryParam("fips")
	bbox := c.QueryParam("bbox")
	apifmt := c.QueryParam("fmt")
	if apifmt == "" {
		apifmt = "fc"
	}
	var params []interface{}
	fipsCriteria, params, err := getFipsCriteria(fips, params)
	if err != nil {
		return err
	}
	bboxCriteria, err := getBboxCriteria(bbox, 4326)
	if err != nil {
		return err
	}
	criteria := buildCritieria(bboxCriteria, fipsCriteria, api.config.FeatureLimit)
	fmt.Println(criteria)
	rows, err := api.dataStore.Db.Queryx(fmt.Sprintf("%s %s", NsiSelect, criteria), params...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if apifmt == "fs" {
		err = rowsToGeojsonStream(c, rows)
	} else {
		err = rowsToGeojson(c, apifmt, rows)
	}
	return err
}

func (api *ApiHandler) StructuresFromUpload(c echo.Context) error {
	geodataPost := GeodataPost{
		echoContext:     c,
		tempStoragePath: api.config.TempStoragePath,
	}

	apifmt := c.QueryParam("fmt")
	if apifmt == "" {
		apifmt = "fc"
	}

	if hasFile, err := geodataPost.HasFile(); hasFile {
		if err != nil {
			return err
		}
		err := geodataPost.ExtractFile()
		if err != nil {
			return err
		}
		err = geodataPost.Open()
		if err != nil {
			return err
		}
	} else {
		geodataPost.OpenFromBody()
	}

	defer geodataPost.Close()

	gdalwkb, err := geodataPost.GetGeometryAsWkb()
	if err != nil {
		return err
	}
	rows, err := api.dataStore.Db.Queryx(fmt.Sprintf("%s %s", NsiSelect, "where st_intersects(shape,st_geomfromwkb($1,4326))"), gdalwkb)
	if err != nil {
		return err
	}
	defer rows.Close()
	if apifmt == "fs" {
		err = rowsToGeojsonStream(c, rows)
	} else {
		err = rowsToGeojson(c, apifmt, rows)
	}
	if err != nil {
		return err
	}
	return nil
}

func (api *ApiHandler) StructuresFromPost(c echo.Context) error {
	geodataPost := GeodataPost{
		echoContext:     c,
		tempStoragePath: api.config.TempStoragePath,
	}

	apifmt := c.QueryParam("fmt")
	if apifmt == "" {
		apifmt = "fc"
	}

	err := geodataPost.OpenFromBody()
	if err != nil {
		return err
	}
	defer geodataPost.Close()
	gdalwkb, err := geodataPost.GetGeometryAsWkb()
	if err != nil {
		return err
	}
	rows, err := api.dataStore.Db.Queryx(fmt.Sprintf("%s %s", NsiSelect, "where st_intersects(shape,st_geomfromwkb($1,4326))"), gdalwkb)
	if err != nil {
		return err
	}
	defer rows.Close()
	if apifmt == "fs" {
		err = rowsToGeojsonStream(c, rows)
	} else {
		err = rowsToGeojson(c, apifmt, rows)
	}
	if err != nil {
		return err
	}
	return nil
}

func (api *ApiHandler) CreateExport(c echo.Context) error {
	bbox := c.QueryParam("bbox")
	bboxCriteria, err := getBboxCriteria(bbox, 4326)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf("select * from nsi where %s", bboxCriteria)
	uuid, _ := uuid.NewUUID()
	name := uuid.String()
	api.tempStore.PutStatus(name, "Initialized")
	etl := Db2FileEtl{
		DbDriver:     "PostgreSQL",
		UrlTemplate:  "PG: host=%s dbname=%s user=%s password=%s",
		DbDialect:    "POSTGRESQL",
		DbOptions:    []string{"GEOMETRY_NAME=shape"},
		User:         api.config.Dbuser,
		Pass:         api.config.Dbpass,
		Host:         api.config.Dbhost,
		Db:           api.config.Dbname,
		Sql:          sql,
		FileDriver:   "GPKG",
		NewLayerName: "nsi_export",
		FileOut:      name + ".gpkg",
		Guid:         name,
	}
	go RunDb2FileEtl(&etl, api.tempStore, api.config.TempStoragePath, &ConsoleReporter{})
	c.String(http.StatusOK, name)
	return nil
}

func (api *ApiHandler) ExportFromUpload(c echo.Context) error {
	geodataPost := GeodataPost{
		echoContext:     c,
		tempStoragePath: api.config.TempStoragePath,
	}
	err := geodataPost.ExtractFile()
	if err != nil {
		return err
	}
	err = geodataPost.Open()
	if err != nil {
		return err
	}
	defer geodataPost.Close()
	filterGeom, err := geodataPost.GetGeometry()
	if err != nil {
		return err
	}
	uuidVal, _ := uuid.NewUUID()
	name := uuidVal.String()
	etl := Db2FileEtl{
		DbDriver:     "PostgreSQL",
		UrlTemplate:  "PG: host=%s dbname=%s user=%s password=%s",
		DbDialect:    "POSTGRESQL",
		DbOptions:    []string{"GEOMETRY_NAME=shape"},
		User:         api.config.Dbuser,
		Pass:         api.config.Dbpass,
		Host:         api.config.Dbhost,
		Db:           api.config.Dbname,
		Sql:          "select * from nsi",
		GeomFilter:   filterGeom,
		FileDriver:   "GPKG",
		NewLayerName: "nsi_export",
		FileOut:      name + ".gpkg",
		Guid:         name,
	}
	go RunDb2FileEtl(&etl, api.tempStore, api.config.TempStoragePath, &ConsoleReporter{})
	c.String(http.StatusOK, name)

	return nil
}

func (api *ApiHandler) GetStats(c echo.Context) error {
	bbox := c.QueryParam("bbox")
	bboxCriteria, err := getBboxCriteria(bbox, 4326)
	if err != nil {
		return err
	}
	criteria := buildCritieria(bboxCriteria, "", api.config.FeatureLimit)
	var nsiSummary NsiSummary
	err = api.dataStore.Db.Get(&nsiSummary, fmt.Sprintf("%s %s", NsiStatsSelect, criteria))
	if err == nil {
		c.JSON(http.StatusOK, &nsiSummary)
	}
	return err
}

func (api *ApiHandler) StatsFromUpload(c echo.Context) error {
	geodataPost := GeodataPost{
		echoContext:     c,
		tempStoragePath: api.config.TempStoragePath,
	}

	if hasFile, err := geodataPost.HasFile(); hasFile {
		if err != nil {
			return err
		}
		err := geodataPost.ExtractFile()
		if err != nil {
			return err
		}
		err = geodataPost.Open()
		if err != nil {
			return err
		}
	} else {
		geodataPost.OpenFromBody()
	}

	defer geodataPost.Close()
	gdalwkb, err := geodataPost.GetGeometryAsWkb()
	if err != nil {
		return err
	}

	var nsiSummary NsiSummary
	err = api.dataStore.Db.Get(&nsiSummary, fmt.Sprintf("%s %s", NsiStatsSelect, "where st_intersects(shape,st_geomfromwkb($1,4326))"), gdalwkb)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, &nsiSummary)

	return nil
}

func (api *ApiHandler) GetExport(c echo.Context) error {
	id := c.Param("uuid")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("%s%s.%s", api.config.TempStoragePath, uuid.String(), "gpkg")
	path = sanitizePath(path)
	return c.Attachment(path, "nsi_export.gpkg")
}

func (api *ApiHandler) GetHexbins(c echo.Context) error {
	hbds := c.Param("dataset")
	bbox := c.QueryParam("bbox")
	if hbds == "" || bbox == "" {
		return errors.New("Hexbin dataset and bounding box are required")
	}
	dataset := HexbinDatasets[hbds]
	if dataset == "" {
		return errors.New("Invalid hexbin dataset")
	}
	bboxCriteria, err := getBboxCriteria(bbox, 3857)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf(HexbinSelect, dataset)
	//fmt.Printf(fmt.Sprintf("%s %s\n", sql, bboxCriteria))
	rows, err := api.dataStore.Db.Queryx(fmt.Sprintf("%s where %s", sql, bboxCriteria))
	if err != nil {
		return err
	}
	defer rows.Close()
	err = rowsToGeojsonHb(c, rows)
	return err

}

func sanitizePath(path string) string {
	path = filepath.Clean(path)
	return strings.ReplaceAll(path, "..", "")
}

func rowsToGeojsonHb(c echo.Context, rows *sqlx.Rows) error {
	hb := Hexbin{}
	writer := c.Response().Writer
	writer.Write([]byte(`{"type": "FeatureCollection",`))
	writer.Write([]byte(`"features":`))
	writer.Write(arrayStart)
	for i := 0; rows.Next(); i++ {
		err := rows.StructScan(&hb)
		if err != nil {
			log.Printf("Unable to scan hexbin:%s\n", err)
			return err
		}
		geom, err := wkb.Unmarshal(hb.Shape)
		if err != nil {
			log.Printf("Unable to unmarshall hexbin geometry:%s\n", err)
			return err
		}
		geom4326 := project.Geometry(geom, project.Mercator.ToWGS84)
		ggeom := geojson.NewGeometry(geom4326)
		jsonb, err := ggeom.MarshalJSON()
		if err != nil {
			log.Printf("Unable to marshall hexbin geometry to geojson:%s\n", err)
			return err
		}
		if i > 0 {
			writer.Write(featureSeparator)
		}
		writer.Write([]byte(`{"type": "Feature","geometry":`))
		writer.Write(jsonb)
		writer.Write([]byte(`,"properties":`))
		writer.Write([]byte(hexbinRecToProps(&hb)))
		writer.Write(featureEnd)
	}
	writer.Write(arrayEnd)
	writer.Write(objectEnd)
	c.Response().Flush()
	return nil
}

func hexbinRecToProps(hb *Hexbin) string {
	var builder strings.Builder
	builder.WriteString("{")
	builder.WriteString(fmt.Sprintf(`"OBJECTID":%d,`, hb.ID))
	val := reflect.ValueOf(hb).Elem()
	fv := val.FieldByName("NsiSummary")
	for i := 0; i < fv.NumField(); i++ {
		p := fv.Field(i)
		t := fv.Type().Field(i)
		if i > 0 {
			builder.WriteString(",")
		}
		if tagval, ok := t.Tag.Lookup("json"); ok {
			switch p.Kind() {
			case reflect.Int32, reflect.Int64:
				builder.WriteString(fmt.Sprintf(`"%s":%d`, tagval, p.Interface()))
			case reflect.Float64:
				builder.WriteString(fmt.Sprintf(`"%s":%.2f`, tagval, p.Interface()))
			}
		}
	}
	builder.WriteString("}")
	return builder.String()
}

//@TODO this has potential to return mangled json on error
//need to decide best approch.  mangle or skip...
func rowsToGeojson(c echo.Context, apifmt string, rows *sqlx.Rows) error {
	nsi := Nsi{}

	if apifmt == "fc" {
		c.Response().Write(featureCollectionStart)
	}

	c.Response().Write(arrayStart)
	for i := 0; rows.Next(); i++ {
		err := rows.StructScan(&nsi)
		if err != nil {
			log.Printf("Unable to map query to NSI Struct. Msg: %s\n", err)
			return err
		}
		props, err := json.Marshal(nsi)
		if err != nil {
			log.Printf("Unable to encode nsi record to JSON. Msg: %s\n", err)
			return err
		}
		if i > 0 {
			c.Response().Write(featureSeparator)
		}
		c.Response().Write([]byte(fmt.Sprintf(featureTemplate, nsi.X, nsi.Y)))
		c.Response().Write(props)
		c.Response().Write(featureEnd)
	}
	c.Response().Write(arrayEnd)
	if apifmt == "fc" {
		c.Response().Write(featureEnd) //actually closing out the feature collection object.
	}
	c.Response().Flush()
	return nil
}

func rowsToGeojsonStream(c echo.Context, rows *sqlx.Rows) error {
	nsi := Nsi{}
	for i := 0; rows.Next(); i++ {
		err := rows.StructScan(&nsi)
		if err != nil {
			log.Printf("Unable to map query to NSI Struct. Msg: %s\n", err)
			return err
		}
		props, err := json.Marshal(nsi)
		if err != nil {
			log.Printf("Unable to encode nsi record to JSON. Msg: %s\n", err)
			return err
		}
		c.Response().Write([]byte(fmt.Sprintf(featureTemplate, nsi.X, nsi.Y)))
		c.Response().Write(props)
		c.Response().Write(featureEnd)
	}
	c.Response().Flush()
	return nil
}

func buildCritieria(bboxCriteria string, fipsCritiera string, featureLimit string) string {
	var builder s.Builder
	builder.WriteString("where ")
	if bboxCriteria != "" {
		builder.WriteString(bboxCriteria)
	}
	if fipsCritiera != "" {
		if bboxCriteria != "" {
			builder.WriteString(" and ")
		}
		builder.WriteString(fipsCritiera)
	}
	//builder.WriteString(fmt.Sprintf(" limit %s", featureLimit))
	return builder.String()
}

func getFipsCriteria(fips string, params []interface{}) (string, []interface{}, error) {
	var fipsCriteria string
	if fips != "" {
		if !contains(validFipsLengths, len(fips)) {
			return "", nil, errors.New("Invalid FIPS query")
		}
		params = append(params, fips)
		paramsCount := len(params)
		fipsLen := len(fips)
		if fipsLen == 15 {
			fipsCriteria = fmt.Sprintf("cbfips=$%d", paramsCount)
		} else {
			fipsCriteria = fmt.Sprintf("substr(cbfips,1,%d)=$%d", fipsLen, paramsCount)
		}
	} else {
		fipsCriteria = ""
	}
	return fipsCriteria, params, nil
}

func getBboxCriteria(bbox string, crs int) (string, error) {
	bboxCriteria := ""
	if bbox != "" {
		coords, err := StringToCoords(bbox)
		if err != nil {
			log.Printf("Unable to convert bbox coordinates: %s; Error was %s", bbox, err.Error())
			return "", err
		} else {
			ls := CoordsToLineString(coords)
			poly := LineStringToPoly(ls)
			switch crs {
			case 3857:
				merc := project.Polygon(*poly, project.WGS84.ToMercator)
				bboxCriteria = fmt.Sprintf("st_intersects(shape,'SRID=3857;%s')", wkt.MarshalString(merc))
			default:
				bboxCriteria = fmt.Sprintf("st_intersects(shape,'SRID=4326;%s')", wkt.MarshalString(*poly))
			}
		}
	}
	return bboxCriteria, nil
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
