package stores

import (
	"fmt"

	"github.com/hydrologicengineeringcenter/nsiapi/internal/global"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/models"
	"github.com/usace/goquery"
)

const (
	DbSchema = global.DB_SCHEMA
)

var datasetTable = goquery.TableDataSet{
	Name:   "dataset",
	Schema: DbSchema,
	Statements: map[string]string{
		"selectId":   `select id from dataset where name=$1 and version=$2 and purpose=$3 and quality_id=$4`,
		"select":     `select * from dataset where name=$1 and version=$2 and quality_id=$3`,
		"selectById": `select * from dataset where id=$1`,
		"insertNullShape": `insert into dataset (
            name,
            version,
            nsi_schema_id,
            table_name,
            shape,
            description,
            purpose,
            created_by,
            quality_id,
            group_id
        ) values ($1, $2, $3, $4, ST_Envelope('POLYGON((0 0, 0 0, 0 0, 0 0))'::geometry), $5, $6, $7, $8, $9) returning id`,
		"updateBBox":            fmt.Sprintf(`update dataset set shape=(select ST_Envelope(ST_Collect(shape)) from %s.{table_name}) where id=$1`, DbSchema),
		"structureInInventory":  fmt.Sprintf(`select fd_id from %s.{table_name} where X=$1 and Y=$2`, DbSchema),
		"elevationColumnExists": `select exists (select 1 from information_schema.columns where table_schema=$1 and table_name=$2 and column_name=$3)`,
		"addElevColumn":         fmt.Sprintf(`alter table %s.{table_name} add column %s double precision`, DbSchema, global.ELEVATION_COLUMN_NAME),
		"selectEmptyElevationCoords": fmt.Sprintf(
			"select fd_id, X, Y, %s from %s.{table_name} where %s is null limit $1 offset $2",
			global.ELEVATION_COLUMN_NAME,
			DbSchema,
			global.ELEVATION_COLUMN_NAME,
		),
		"updateElevation": fmt.Sprintf("update %s.{table_name} set %s=$1 where fd_id=$2", DbSchema, global.ELEVATION_COLUMN_NAME),
	},
}

var domainTable = goquery.TableDataSet{
	Name:   "domain",
	Schema: DbSchema,
	Statements: map[string]string{
		"selectId": `select id from domain where field_id=$1 and value=$2`,
		"insert":   `insert into domain (field_id, value) values ($1, $2) returning id`,
	},
	Fields: models.Domain{},
}

var fieldTable = goquery.TableDataSet{
	Name:   "field",
	Schema: DbSchema,
	Statements: map[string]string{
		"select":     `select id from field where name=$1`,
		"selectById": `select * from field where id=$1`,
		"insert":     `insert into field (name, type, description, is_domain) values ($1, $2, $3, $4) returning id`,
	},
	Fields: models.Field{},
}

var groupTable = goquery.TableDataSet{
	Name:   "access",
	Schema: DbSchema,
	Statements: map[string]string{
		"selectId": `select id from nsi_group where name=$1`,
		"insert":   `insert into nsi_group (name) values ($1) returning id`,
	},
	Fields: models.Group{},
}

var memberTable = goquery.TableDataSet{
	Name:   "group_member",
	Schema: DbSchema,
	Statements: map[string]string{
		"selectId":   `select id from group_member where group_id=$1 and user_id=$2`,
		"insert":     `insert into group_member (group_id, role, user_id) values ($1, $2, $3) returning id`,
		"updateRole": `update group_member set role=$2 where id=$1`,
	},
	Fields: models.Group{},
}

var qualityTable = goquery.TableDataSet{
	Name:   "quality",
	Schema: DbSchema,
	Statements: map[string]string{
		"selectId": `select id from quality where value=$1`,
		"select":   `select * from quality where value=$1`,
		"insert":   `insert into quality (value, description) values ($1, $2) returning id`,
	},
	Fields: models.Quality{},
}

var schemaFieldTable = goquery.TableDataSet{
	Name:   "schema_field",
	Schema: DbSchema,
	Statements: map[string]string{
		"selectId": `select id from schema_field where id=$1 and field_id=$2`,
		"insert":   `insert into schema_field (id, field_id, is_private) values ($1, $2, $3) returning id`,
	},
	Fields: models.Field{},
}

var schemaTable = goquery.TableDataSet{
	Name:   "schema",
	Schema: DbSchema,
	Statements: map[string]string{
		"select":     `select * from nsi_schema where name=$1 and version=$2`,
		"selectId":   `select id from nsi_schema where name=$1 and version=$2`,
		"selectById": `select * from nsi_schema where id=$1`,
		"insert":     `insert into nsi_schema (name, version, notes) values ($1, $2, $3) returning id`,
	},
	Fields: models.Schema{},
}
