package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/hydrologicengineeringcenter/nsiapi/internal/models/types"
)

// ducktyping in go with reflect is a bad idea
// Point should not be used in production, just for mocking
type Point struct {
	Bid        string  `db:"bid"` //
	Cbfips2010 string  `db:"cbfips2010"`
	St_damcat  string  `db:"st_damcat"`
	Occtype    string  `db:"occtype"`
	Num_story  int32   `db:"num_story"` //
	Height     float64 `db:"height"`    //
	Sqft       float64 `db:"sqft"`
	Ftprntsqft float64 `db:"ftprntsqft"` //
	Found_ht   float64 `db:"found_ht"`
	Extwall    string  `db:"extwall"` //
	Fndtype    string  `db:"fndtype"`
	Bsmnt      string  `db:"bsmnt"`
	P_extwall  string  `db:"p_extwall"`  //
	P_fndtype  string  `db:"p_fndtype"`  //
	P_bsmnt    string  `db:"p_bsmnt"`    //
	Total_room int32   `db:"total_room"` //
	Bedrooms   int32   `db:"bedrooms"`
	Total_bath int32   `db:"total_bath"` //
	P_garage   string  `db:"p_garage"`   //
	Parkingsp  int32   `db:"parkingsp"`  //
	Yrbuilt    int32   `db:"yrbuilt"`
	Med_yr_blt int32   `db:"med_yr_blt"`
	Naics      string  `db:"naics"`      //
	Bldcostcat string  `db:"bldcostcat"` //
	Val_struct float64 `db:"val_struct"`
	Val_cont   float64 `db:"val_cont"`
	Val_vehic  float64 `db:"val_vehic"`
	Numvehic   int32   `db:"numvehic"`  //
	Ftprntid   string  `db:"ftprntid"`  //
	Ftprntsrc  string  `db:"ftprntsrc"` //
	Source     string  `db:"source"`
	Resunits   int32   `db:"resunits"`
	Empnum     int32   `db:"empnum"`
	Students   int32   `db:"students"`
	Surplus    int32   `db:"surplus"`    //
	Othinstpop int32   `db:"othinstpop"` //
	Nursghmpop int32   `db:"nursghmpop"` //
	Pop2amu65  int32   `db:"pop2amu65"`
	Pop2amo65  int32   `db:"pop2amo65"`
	Pop2pmu65  int32   `db:"pop2pmu65"`
	Pop2pmo65  int32   `db:"pop2pmo65"`
	O65disable float64 `db:"o65disable"`
	U65disable float64 `db:"u65disable"`
	X          float64 `db:"x"`
	Y          float64 `db:"y"`
	Apn        string  `db:"apn"`        //
	Censregion string  `db:"censregion"` //
	Firmzone   string  `db:"firmzone"`
	Firmdate   string  `db:"firmdate"` //
}

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
	ShpName     string         // field name from shapefile
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
