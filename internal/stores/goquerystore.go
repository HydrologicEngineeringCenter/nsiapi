package stores

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/config"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/global"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/models"
	"github.com/usace/goquery"
)

type PSStore struct {
	DS goquery.DataStore
}

func NewGQStore(c config.AppConfig) (*PSStore, error) {
	dbconf := c.Rdbmsconfig()
	ds, err := goquery.NewRdbmsDataStore(&dbconf)
	if err != nil {
		log.Printf("Unable to connect to database during startup: %s", err)
	} else {
		log.Printf("Connected as %s to database %s:%s/%s", c.Dbuser, c.Dbhost, c.Dbport, c.Dbname)
	}

	st := PSStore{ds}
	return &st, nil
}

func (st *PSStore) AddDomain(d *models.Domain) error {
	var dId uuid.UUID
	err := st.DS.Select().
		DataSet(&domainTable).
		StatementKey("insert").
		Params(d.FieldId, d.Value).
		Dest(&dId).
		Fetch()
	if err != nil {
		return err
	}
	d.Id = dId
	return nil
}

func (st *PSStore) AddField(f *models.Field) error {
	var fId uuid.UUID
	err := st.DS.Select().
		DataSet(&fieldTable).
		StatementKey("insert").
		Params(f.DbName, f.Type, f.Description, f.IsDomain).
		Dest(&fId).
		Fetch()
	if err != nil {
		return err
	}
	f.Id = fId
	return nil
}

func (st *PSStore) AddMember(m *models.Member) error {
	var mId uuid.UUID
	err := st.DS.Select().
		DataSet(&memberTable).
		StatementKey("insert").
		Params(m.GroupId, m.Role, m.UserId).
		Dest(&mId).
		Fetch()
	if err != nil {
		return err
	}
	m.Id = mId
	return nil
}

func (st *PSStore) AddSchemaFieldAssociation(sf models.SchemaField) error {
	var schemaId uuid.UUID
	err := st.DS.Select().
		DataSet(&schemaFieldTable).
		StatementKey("insert").
		Params(sf.Id, sf.NsiFieldId, sf.IsPrivate).
		Dest(&schemaId).
		Fetch()
	if err != nil {
		return err
	}
	return nil
}

func (st *PSStore) AddSchema(schema *models.Schema) error {
	var schemaId uuid.UUID
	err := st.DS.Select().
		DataSet(&schemaTable).
		StatementKey("insert").
		Params(schema.Name, schema.Version, schema.Notes).
		Dest(&schemaId).
		Fetch()
	if err != nil {
		return err
	}
	schema.Id = schemaId
	return err
}

func (st *PSStore) AddDataset(d *models.Dataset) error {
	var ids []uuid.UUID
	err := st.DS.
		Select().
		DataSet(&datasetTable).
		StatementKey("insertNullShape").
		Params(
			d.Name,
			d.Version,
			d.SchemaId,
			d.TableName,
			d.Description,
			d.Purpose,
			d.CreatedBy,
			d.QualityId,
			d.GroupId,
		).
		Dest(&ids).
		Fetch()
	if err != nil {
		return err
	}
	d.Id = ids[0]
	return err
}

func (st *PSStore) AddGroup(g *models.Group) error {
	var id uuid.UUID
	err := st.DS.Select().
		DataSet(&groupTable).
		StatementKey("insert").
		Params(g.Name).
		Dest(&id).
		Fetch()
	if err != nil {
		return err
	}
	g.Id = id
	return err
}

func (st *PSStore) GetDomainId(d models.Domain) (uuid.UUID, error) {
	var ids []uuid.UUID
	err := st.DS.
		Select().
		DataSet(&schemaTable).
		StatementKey("selectId").
		Params(d.FieldId, d.Value).
		Dest(&ids).
		Fetch()
	if err != nil {
		return uuid.UUID{}, err
	}
	if len(ids) == 0 {
		return uuid.UUID{}, nil
	}
	if len(ids) > 1 {
		return uuid.UUID{}, errors.New("more than 1 id exists for domain.field_id=" + d.FieldId.String() + ", domain.value=" + d.Value)
	}
	return ids[0], err
}

func (st *PSStore) GetGroupId(g *models.Group) error {
	var ids []uuid.UUID
	err := st.DS.
		Select().
		DataSet(&groupTable).
		StatementKey("selectId").
		Params(g.Name).
		Dest(&ids).
		Fetch()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 1 {
		return errors.New("more than 1 id exists for group.name=" + g.Name)
	}
	g.Id = ids[0]
	return nil
}

func (st *PSStore) GetMemberId(m *models.Member) error {
	var ids []uuid.UUID
	err := st.DS.
		Select().
		DataSet(&memberTable).
		StatementKey("selectId").
		Params(m.GroupId, m.UserId).
		Dest(&ids).
		Fetch()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 1 {
		return fmt.Errorf("more than 1 id exists for group_member.group_id=%s and group_member.user_id=%s", m.GroupId.String(), m.UserId)
	}
	m.Id = ids[0]
	return nil
}

func (st *PSStore) GetDatasetId(d *models.Dataset) error {
	var ids []uuid.UUID
	err := st.DS.
		Select().
		DataSet(&datasetTable).
		StatementKey("selectId").
		Params(d.Name, d.Version, d.Purpose, d.QualityId).
		Dest(&ids).
		Fetch()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 1 {
		return fmt.Errorf(`more than 1 id exists for
        dataset.name=%s
        dataset.version=%s
        dataset.shape=%s
        dataset.purpose=%s
        dataset.quality_id=%s`,
			d.Name,
			d.Version,
			d.Shape,
			d.Purpose,
			d.QualityId,
		)
	}
	d.Id = ids[0]
	return err
}

// GetDataset queries based on its Name, Version, Purpose, and QualityId
func (st *PSStore) GetDataset(d *models.Dataset) error {
	var ds []models.Dataset
	err := st.DS.
		Select().
		DataSet(&datasetTable).
		StatementKey("select").
		Params(d.Name, d.Version, d.QualityId).
		Dest(&ds).
		Fetch()
	if err != nil {
		return err
	}
	if len(ds) == 0 {
		d.Id = uuid.Nil
	} else {
		*d = ds[0]
	}
	return nil
}

func (st *PSStore) GetFieldId(f *models.Field) error {
	var ids []uuid.UUID
	err := st.DS.
		Select().
		DataSet(&fieldTable).
		StatementKey("select").
		Params(f.DbName).
		Dest(&ids).
		Fetch()
	if len(ids) == 0 {
		f.Id = uuid.Nil
		return err
	}
	if len(ids) > 1 {
		return errors.New("more than 1 id exists for field.name=" + f.DbName + " and field.type=" + string(f.Type))
	}
	f.Id = ids[0]
	return err
}

// GetSchemaId queries the database based on the supplied schema name and version.
// Replaces Id field if a corresponding entry exists, otherwise change Id field to uuid.Nil
func (st *PSStore) GetSchemaId(s *models.Schema) error {
	var ids []uuid.UUID
	err := st.DS.
		Select().
		DataSet(&schemaTable).
		StatementKey("selectId").
		Params(s.Name, s.Version).
		Dest(&ids).
		Fetch()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		s.Id = uuid.Nil
		return nil
	}
	if len(ids) > 1 {
		return errors.New("more than 1 id exists for schema.name=" + s.Name + " and schema.version=" + s.Version)
	}
	s.Id = ids[0]
	return nil
}

func (st *PSStore) GetQuality(q *models.Quality) error {
	var qDb models.Quality
	err := st.DS.
		Select().
		DataSet(&qualityTable).
		StatementKey("select").
		Params(q.Value).
		Dest(&qDb).
		Fetch()
	if err != nil {
		return err
	}
	*q = qDb
	return nil
}

func (st *PSStore) GetQualityId(q *models.Quality) error {
	var ids []uuid.UUID
	err := st.DS.
		Select().
		DataSet(&qualityTable).
		StatementKey("selectId").
		Params(q.Value).
		Dest(&ids).
		Fetch()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 1 {
		return errors.New("more than 1 id exists for quality.value=" + string(q.Value))
	}
	q.Id = ids[0]
	return nil
}

// Check if table exists in database
func (st *PSStore) TableExists(schema string, table string) (bool, error) {
	var result bool
	err := st.DS.Select(`
    SELECT EXISTS (
        SELECT FROM pg_tables
        WHERE
            schemaname='$1' AND
            tablename='$2'
    )
    `).
		Params(schema, table).
		Dest(&result).
		Fetch()
	return result, err
}

func (st *PSStore) SchemaFieldAssociationExists(sf models.SchemaField) (bool, error) {
	var ids []uuid.UUID
	var result bool
	err := st.DS.
		Select().
		DataSet(&schemaFieldTable).
		StatementKey("selectId").
		Params(sf.Id, sf.NsiFieldId).
		Dest(&ids).
		Fetch()
	if err != nil {
		return false, err
	}
	if len(ids) > 0 {
		result = true
	} else {
		result = false
	}
	return result, err
}

func (st *PSStore) UpdateDatasetBBox(d models.Dataset) error {
	// hacky way to dynamically generate table_name since identifiers cannot be used as variables
	// should be safe from sql injection since all table names are generated internally from guids
	var ids []interface{}
	err := st.DS.
		Select(strings.ReplaceAll(datasetTable.Statements["updateBBox"], "{table_name}", d.TableName)).
		Params(d.Id).
		Dest(&ids). // interface doesn't work without a dest sink
		Fetch()
	return err
}

// ShpDataInStore checks if shp file has already been uploaded to database
// func (st *PSStore) ShpDataInStore(d models.Dataset, s *shp.Reader) (bool, error) {
// 	// algo takes a set of random sample points, if any sample matches with
// 	// an entry in the db, return true
// 	var ids []int
// 	sampleSize := 50

// 	xIdx, err := shape.FieldIdx(s, "X")
// 	if err != nil {
// 		return false, err
// 	}
// 	yIdx, err := shape.FieldIdx(s, "Y")
// 	if err != nil {
// 		return false, err
// 	}

// 	for i := 0; i < sampleSize; i++ {
// 		sampleIdx := rand.Int() % s.AttributeCount()
// 		x := s.ReadAttribute(sampleIdx, xIdx)
// 		y := s.ReadAttribute(sampleIdx, yIdx)
// 		err := st.DS.
// 			Select(strings.ReplaceAll(datasetTable.Statements["structureInInventory"], "{table_name}", d.TableName)).
// 			Params(x, y).
// 			Dest(&ids).
// 			Fetch()
// 		if err != nil {
// 			return false, err
// 		}
// 		if len(ids) > 0 {
// 			return true, nil
// 		}
// 	}
// 	return false, nil
// }

func (st *PSStore) UpdateMemberRole(m *models.Member) error {
	var ids []interface{}
	err := st.DS.
		Select().
		DataSet(&memberTable).
		StatementKey("updateRole").
		Params(m.Id, m.Role).
		Dest(&ids). // interface doesn't work without a dest sink
		Fetch()
	return err
}

// ElevationColumnExists tests if elevation column exists for inventory table
func (st *PSStore) ElevationColumnExists(d models.Dataset) (bool, error) {
	var res bool
	err := st.DS.
		Select(datasetTable.Statements["elevationColumnExists"]).
		Params(global.DB_SCHEMA, d.TableName, global.ELEVATION_COLUMN_NAME).
		Dest(&res).
		Fetch()
	if err != nil {
		return false, err
	}
	return res, nil
}

func (st *PSStore) AddElevationColumn(d models.Dataset) error {
	sql := strings.ReplaceAll(datasetTable.Statements["addElevColumn"], "{table_name}", d.TableName)
	tx, err := st.DS.Transaction()
	err = st.DS.Exec(&tx, sql)
	if err != nil {
		return err
	}
	err = tx.Commit()
	return err
}

func (st *PSStore) GetEmptyElevationPoints(d models.Dataset, count int, offset int) (models.Points, error) {
	sql := strings.ReplaceAll(datasetTable.Statements["selectEmptyElevationCoords"], "{table_name}", d.TableName)
	var coords models.Points
	err := st.DS.
		Select(sql).
		Params(count, offset).
		Dest(&coords).
		Fetch()
	if err != nil {
		return nil, err
	}
	return coords, nil
}

func (st *PSStore) UpdateElevationAtPoint(d models.Dataset, points models.Points) error {
	// batchSize here is the db update batchSize, not to be confused with the goroutine batchSize
	batchSize := 1000
	var tx goquery.Tx
	var err error
	tx, err = st.DS.Transaction()
	if err != nil {
		return err
	}
	for i, p := range points {
		if i%batchSize == 0 {
		}
		sql := strings.ReplaceAll(datasetTable.Statements["updateElevation"], "{table_name}", d.TableName)
		err = st.DS.Exec(&tx, sql, *p.Elevation, p.FdId)
		if err != nil {
			return err
		}
		// commit batch, create new Tx
		if i%batchSize == 0 {
			err = tx.Commit()
			if err != nil {
				return err
			}
			tx, err = st.DS.Transaction()
			if err != nil {
				return err
			}
		}
	}
	// flush Tx queue
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

//////////////////////////////////////////////////
// Add row to sql table generically - TODO unstable, need to test
//////////////////////////////////////////////////

// appmodels. contrainsts generic database access to only a list of structs
type appmodels interface {
	models.Field | models.Schema | models.Domain | models.SchemaField | models.Dataset | models.Group | models.Member | models.Quality
}

func getAppmodels[T appmodels](m *T) uuid.UUID {
	return reflect.ValueOf(*m).FieldByName("Id").Interface().(uuid.UUID)
}

func setAppmodels[T appmodels](m *T, id uuid.UUID) {
	reflect.ValueOf(*m).FieldByName("Id").SetBytes(id[:])
}

type insertConfig struct {
	StatementKey string
	FieldOrder   []string
	QueryTable   *goquery.TableDataSet
}

var (
	// this mapper should be app specific
	insertConfigMapper = map[reflect.Type]insertConfig{
		// quality should not be insertable
		reflect.TypeOf(models.Dataset{}): {
			StatementKey: "insertNullShape",
			FieldOrder: []string{
				"Name", "Version", "SchemaId", "TableName", "Description", "Purpose", "CreatedBy", "QualityId", "GroupId",
			},
			QueryTable: &datasetTable,
		},
		reflect.TypeOf(models.Domain{}): {
			StatementKey: "insert",
			FieldOrder: []string{
				"FieldId", "Value",
			},
			QueryTable: &domainTable,
		},
		reflect.TypeOf(models.Field{}): {
			StatementKey: "insert",
			FieldOrder: []string{
				"DbName", "Type", "Description", "IsDomain",
			},
			QueryTable: &fieldTable,
		},
		reflect.TypeOf(models.Schema{}): {
			StatementKey: "insert",
			FieldOrder: []string{
				"Name", "Version", "Notes",
			},
			QueryTable: &schemaTable,
		},
		reflect.TypeOf(models.SchemaField{}): {
			StatementKey: "insert",
			FieldOrder: []string{
				"NsiFieldId", "IsPrivate",
			},
			QueryTable: &schemaFieldTable,
		},
		reflect.TypeOf(models.Group{}): {
			StatementKey: "insert",
			FieldOrder: []string{
				"Name",
			},
			QueryTable: &groupTable,
		},
		reflect.TypeOf(models.Member{}): {
			StatementKey: "insert",
			FieldOrder: []string{
				"GroupId", "Role", "UserId",
			},
			QueryTable: &memberTable,
		},
	}
)

// AddRow adds row to table based on a models.struct
func AddRow[T appmodels](st *PSStore, m *T) error {
	var params []interface{}
	modelType := reflect.TypeOf(*m)
	cfg := insertConfigMapper[modelType]
	// loop over all insertable fields
	for _, f := range cfg.FieldOrder {
		fieldVal := reflect.ValueOf(*m).FieldByName(f)
		valKind := fieldVal.Kind()
		switch valKind {
		case reflect.Bool:
			params = append(params, fieldVal.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			params = append(params, fieldVal.Int())
		case reflect.Float32, reflect.Float64:
			params = append(params, fieldVal.Float())
		case reflect.String:
			params = append(params, fieldVal.String())
		case reflect.TypeOf(uuid.Nil).Kind():
			// uuid  Kind() is Ox17 Array, potentially not safe
			params = append(params, fieldVal.Interface().(uuid.UUID))
		default:
			return fmt.Errorf("Generic AddRow does not support param of type: %s", valKind)
		}
	}

	var id uuid.UUID
	err := st.DS.
		Select().
		DataSet(cfg.QueryTable).
		StatementKey(cfg.StatementKey).
		Params(params...).
		Dest(&id).
		Fetch()
	if err != nil {
		return err
	}
	if getAppmodels(m) == uuid.Nil && id != uuid.Nil {
		setAppmodels(m, id)
	}
	return nil
}
