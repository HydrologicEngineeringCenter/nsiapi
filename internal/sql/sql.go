package sql

import (
	"errors"
	"fmt"
	"log"

	"github.com/hydrologicengineeringcenter/nsiapi/internal/gis"
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
