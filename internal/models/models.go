package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/models/types"
)

type Point struct {
	FdId      int      `db:"fd_id"`
	X         float64  `db:"x"`
	Y         float64  `db:"y"`
	Elevation *float64 `db:"ground_elev"` // pointer instead of value for nullable type
}

type Points []*Point

//  Data is organized into the following concepts:
//  Inventory - Table holding actual data ie concrete data within the dataset
//  Dataset - Grouping of data
//      Access - Access definition specific to each dataset
//      Quality - Quality of dataset
//      Schema - Grouping of unified format across multiple datasets
//          Field - Data field tied to each dataset
//          Domain - Set of possible values if the field is discrete categorical

type Domain struct {
	Id      uuid.UUID `db:"id"`
	FieldId uuid.UUID `db:"field_id"`
	Value   string    `db:"value"`
}

type Field struct {
	Id          uuid.UUID      `db:"id"`
	ShpName     string         // unused
	DbName      string         `db:"name"`
	Type        types.Datatype `db:"type"`
	Description string         `db:"description"`
	IsDomain    bool           `db:"is_domain"`
	IsInDb      bool           // store in db or remove
}

type SchemaField struct {
	Id         uuid.UUID `db:"id"` // map to schema_id key
	NsiFieldId uuid.UUID `db:"nsi_field_id"`
	IsPrivate  bool      `db:"private"` // field can be private in one schema but not another
}

type Schema struct {
	Id      uuid.UUID `db:"id"`
	Name    string    `db:"name"`
	Version string    `db:"version"`
	Notes   string    `db:"notes"`
}

type Quality struct {
	Id          uuid.UUID     `db:"id"`
	Value       types.Quality `db:"value"`
	Description string        `db:"description"`
}

type Dataset struct {
	Id          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Version     string    `db:"version"`
	SchemaId    uuid.UUID `db:"nsi_schema_id"`
	TableName   string    `db:"table_name"`
	Shape       []byte    `db:"shape"` // shape is a BBox enveloping all points within the inventory table
	Description string    `db:"description"`
	Purpose     string    `db:"purpose"`
	DateCreated time.Time `db:"date_created"`
	CreatedBy   string    `db:"created_by"`
	QualityId   uuid.UUID `db:"quality_id"`
	GroupId     uuid.UUID `db:"group_id"`
}

type Group struct {
	Id   uuid.UUID `db:"id"`
	Name string    `db:"name"`
}

type Member struct {
	Id      uuid.UUID  `db:"id"`
	GroupId uuid.UUID  `db:"group_id"`
	Role    types.Role `db:"role"`
	UserId  string     `db:"user_id"`
}
