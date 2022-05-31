package sql

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hydrologicengineeringcenter/nsiapi/internal/gis"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/models"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/stores"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/utils"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/paulmach/orb/project"
)

var validFipsLengths []int = []int{2, 5, 11, 12, 15}

func GetFipsCriteria(fips string, params []interface{}) (string, []interface{}, error) {
	var fipsCriteria string
	if fips != "" {
		if !utils.Contains(validFipsLengths, len(fips)) {
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

func GetBboxCriteria(bbox string, crs int) (string, error) {
	bboxCriteria := ""
	if bbox != "" {
		coords, err := gis.StringToCoords(bbox)
		if err != nil {
			log.Printf("Unable to convert bbox coordinates: %s; Error was %s", bbox, err.Error())
			return "", err
		} else {
			ls := gis.CoordsToLineString(coords)
			poly := gis.LineStringToPoly(ls)
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

func BuildCriteria(bboxCriteria string, fipsCritiera string) string {
	var builder strings.Builder
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

// generateSqlColFromSchemaFields generates a list of columns from a list of SchemaFields
func GenerateSqlColListFromSchemaFields(s *stores.DbStore, sfs *[]models.SchemaField) (string, error) {
	var buf strings.Builder
	var f models.Field
	buf.WriteString("fd_id,")
	for i, sf := range *sfs {
		f.Id = sf.NsiFieldId
		err := s.GetField(&f)
		if err != nil {
			return "", err
		}
		buf.WriteString(f.DbName)
		if i < len(*sfs) {
			buf.WriteString(",")
		}
	}
	return buf.String(), nil
}
