package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/config"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/gis"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/models"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/models/types"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/sql"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/stores"
	"github.com/labstack/echo"

	//"github.com/paulmach/orb"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/paulmach/orb/encoding/wkb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/project"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var featureSeparator []byte = []byte(",")
var featureEnd []byte = []byte("}\n")
var objectEnd []byte = []byte("}")
var arrayStart []byte = []byte("[")
var arrayEnd []byte = []byte("]")
var featureCollectionStart []byte = []byte(`{"type": "FeatureCollection","features":`)
var proptag string = "prop"

const featureTemplate = `{"type": "Feature","geometry": {"type": "Point","coordinates": [%f, %f]},"properties":`

/*
   Valid API structure output Formats:
	 fc = feature collection (default)
	 fa = feature array
	 fs = feature stream
*/

type ApiHandler struct {
	TempStore *stores.TempStore
	DataStore *stores.DbStore
	Config    config.AppConfig
}

func (api *ApiHandler) ApiHome(c echo.Context) error {
	return c.String(http.StatusOK, "National Structures Inventory APIv2")
}

func (api *ApiHandler) GetStatus(c echo.Context) error {
	id := c.Param("uuid")
	status, err := api.TempStore.GetStatus(id)
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
	object := api.Config.AwsPrefix + file
	log.Printf("Download request for %s\n", object)

	result, err := s3.New(sess).GetObject(&s3.GetObjectInput{
		Bucket: &api.Config.AwsBucket,
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
	nsi := stores.Nsi{}
	err := api.DataStore.Db.Get(&nsi, fmt.Sprintf("%s %s", stores.NsiSelect, criteria), fdId)
	if err != nil {
		return err
	}
	feature := fmt.Sprintf(featureTemplate, nsi.X, nsi.Y)
	props, err := json.Marshal(nsi)
	if err != nil {
		return err
	}
	return c.String(http.StatusOK, fmt.Sprintf("%s %s}", feature, props))
}

func (api *ApiHandler) GetStructures(c echo.Context) error {
	paramKeys := []string{"quality", "dataset", "version", "fips", "bbox", "fmt"}
	urlParams := parseUrlParams(&c, paramKeys)
	d := models.Dataset{
		Name: c.Param("dataset"),
	}
	// if dataset isn't specified, default to designated
	if d.Name == "" {
		urlParams["dataset"] = api.Config.DefaultDatasetName
		urlParams["version"] = api.Config.DefaultDatasetVersion
		urlParams["quality"] = api.Config.DefaultDatasetQuality
	}
	fips := urlParams["fips"]
	bbox := urlParams["bbox"]
	apifmt := urlParams["fmt"]
	if apifmt == "" {
		apifmt = "fc"
	}
	var params []interface{}
	fipsCriteria, params, err := sql.GetFipsCriteria(fips, params)
	if err != nil {
		return err
	}
	bboxCriteria, err := sql.GetBboxCriteria(bbox, 4326)
	if err != nil {
		return err
	}
	criteria := sql.BuildCriteria(bboxCriteria, fipsCriteria)

	q := models.Quality{
		Value: types.QualityReverse[urlParams["quality"]],
	}
	err = api.DataStore.GetQualityId(&q)
	if err != nil {
		return err
	}
	if q.Id == uuid.Nil {
		return fmt.Errorf("quality not found")
	}
	d.Version = urlParams["version"]
	d.QualityId = q.Id
	err = api.DataStore.GetDataset(&d)
	if err != nil {
		return err
	}
	if d.Id == uuid.Nil {
		return fmt.Errorf("dataset not found")
	}
	sfs, err := api.DataStore.GetSchemaFieldsById(d.SchemaId)
	if err != nil {
		return err
	}

	query := "SELECT "
	colStr, err := sql.GenerateSqlColListFromSchemaFields(api.DataStore, &sfs)
	if err != nil {
		return err
	}
	query = query + colStr + fmt.Sprintf(`%s FROM %s %s`, query+colStr, d.TableName, criteria)
	rows, err := api.DataStore.Db.Queryx(query, params...)
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
	geodataPost := gis.GeodataPost{
		EchoContext:     c,
		TempStoragePath: api.Config.TempStoragePath,
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
	rows, err := api.DataStore.Db.Queryx(fmt.Sprintf("%s %s", stores.NsiSelect, "where st_intersects(shape,st_geomfromwkb($1,4326))"), gdalwkb)
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
	geodataPost := gis.GeodataPost{
		EchoContext:     c,
		TempStoragePath: api.Config.TempStoragePath,
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
	rows, err := api.DataStore.Db.Queryx(fmt.Sprintf("%s %s", stores.NsiSelect, "where st_intersects(shape,st_geomfromwkb($1,4326))"), gdalwkb)
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
	bboxCriteria, err := sql.GetBboxCriteria(bbox, 4326)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf("select * from nsi where %s", bboxCriteria)
	uuid, _ := uuid.NewUUID()
	name := uuid.String()
	api.TempStore.PutStatus(name, "Initialized")
	etl := gis.Db2FileEtl{
		DbDriver:     "PostgreSQL",
		UrlTemplate:  "PG: host=%s dbname=%s user=%s password=%s",
		DbDialect:    "POSTGRESQL",
		DbOptions:    []string{"GEOMETRY_NAME=shape"},
		User:         api.Config.Dbuser,
		Pass:         api.Config.Dbpass,
		Host:         api.Config.Dbhost,
		Db:           api.Config.Dbname,
		Sql:          sql,
		FileDriver:   "GPKG",
		NewLayerName: "nsi_export",
		FileOut:      name + ".gpkg",
		Guid:         name,
	}
	go gis.RunDb2FileEtl(&etl, api.TempStore, api.Config.TempStoragePath, &gis.ConsoleReporter{})
	c.String(http.StatusOK, name)
	return nil
}

func (api *ApiHandler) ExportFromUpload(c echo.Context) error {
	geodataPost := gis.GeodataPost{
		EchoContext:     c,
		TempStoragePath: api.Config.TempStoragePath,
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
	etl := gis.Db2FileEtl{
		DbDriver:     "PostgreSQL",
		UrlTemplate:  "PG: host=%s dbname=%s user=%s password=%s",
		DbDialect:    "POSTGRESQL",
		DbOptions:    []string{"GEOMETRY_NAME=shape"},
		User:         api.Config.Dbuser,
		Pass:         api.Config.Dbpass,
		Host:         api.Config.Dbhost,
		Db:           api.Config.Dbname,
		Sql:          "select * from nsi",
		GeomFilter:   filterGeom,
		FileDriver:   "GPKG",
		NewLayerName: "nsi_export",
		FileOut:      name + ".gpkg",
		Guid:         name,
	}
	go gis.RunDb2FileEtl(&etl, api.TempStore, api.Config.TempStoragePath, &gis.ConsoleReporter{})
	c.String(http.StatusOK, name)

	return nil
}

func (api *ApiHandler) GetStats(c echo.Context) error {
	bbox := c.QueryParam("bbox")
	bboxCriteria, err := sql.GetBboxCriteria(bbox, 4326)
	if err != nil {
		return err
	}
	criteria := sql.BuildCriteria(bboxCriteria, "")
	var nsiSummary stores.NsiSummary
	err = api.DataStore.Db.Get(&nsiSummary, fmt.Sprintf("%s %s", stores.NsiStatsSelect, criteria))
	if err == nil {
		c.JSON(http.StatusOK, &nsiSummary)
	}
	return err
}

func (api *ApiHandler) StatsFromUpload(c echo.Context) error {
	geodataPost := gis.GeodataPost{
		EchoContext:     c,
		TempStoragePath: api.Config.TempStoragePath,
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

	var nsiSummary stores.NsiSummary
	err = api.DataStore.Db.Get(&nsiSummary, fmt.Sprintf("%s %s", stores.NsiStatsSelect, "where st_intersects(shape,st_geomfromwkb($1,4326))"), gdalwkb)
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
	path := fmt.Sprintf("%s%s.%s", api.Config.TempStoragePath, uuid.String(), "gpkg")
	path = sanitizePath(path)
	return c.Attachment(path, "nsi_export.gpkg")
}

func (api *ApiHandler) GetHexbins(c echo.Context) error {
	hbds := c.Param("dataset")
	bbox := c.QueryParam("bbox")
	if hbds == "" || bbox == "" {
		return errors.New("Hexbin dataset and bounding box are required")
	}
	dataset := stores.HexbinDatasets[hbds]
	if dataset == "" {
		return errors.New("Invalid hexbin dataset")
	}
	bboxCriteria, err := sql.GetBboxCriteria(bbox, 3857)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf(stores.HexbinSelect, dataset)
	//fmt.Printf(fmt.Sprintf("%s %s\n", sql, bboxCriteria))
	rows, err := api.DataStore.Db.Queryx(fmt.Sprintf("%s where %s", sql, bboxCriteria))
	if err != nil {
		return err
	}
	defer rows.Close()
	err = rowsToGeojsonHb(c, rows)
	return err

}

////////////////////////////////////////////////////////
//  private util funcs
////////////////////////////////////////////////////////

func sanitizePath(path string) string {
	path = filepath.Clean(path)
	return strings.ReplaceAll(path, "..", "")
}

func rowsToGeojsonHb(c echo.Context, rows *sqlx.Rows) error {
	hb := stores.Hexbin{}
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

func hexbinRecToProps(hb *stores.Hexbin) string {
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
	nsi := stores.Nsi{}

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
	nsi := stores.Nsi{}
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

func parseUrlParams(c *echo.Context, keys []string) map[string]string {
	var params = map[string]string{}
	for _, k := range keys {
		params[k] = (*c).QueryParam(k)
	}
	return params
}
