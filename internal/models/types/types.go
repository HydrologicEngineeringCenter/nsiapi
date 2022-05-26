package types

type Shape string

type Datatype string

// Field type uses mapping from go-shp
const (
	Char   Datatype = "text"
	Number          = "numeric"
	Float           = "float"
	Date            = "date"
)

var (
	DatatypeReverse = map[string]Datatype{
		"C": Char,
		"N": Number,
		"F": Float,
		"D": Date,
	}
	DatatypeReadable = map[Datatype]string{
		Char:   "text",
		Number: "numeric",
		Float:  "float",
		Date:   "date",
	}
)

func (t Datatype) String() string {
	return DatatypeReadable[t]
}

type Quality string

const (
	High   Quality = "high"
	Medium         = "med"
	Low            = "low"
)

var (
	QualityReverse = map[string]Quality{
		"high": High,
		"med":  Medium,
		"low":  Low,
	}
)

type Role string

const (
	Admin Role = "admin"
	Owner      = "owner"
	User       = "user"
)

var (
	RolePermission = map[Role]string{
		Admin: "read add delete update",
		Owner: "read add update",
		User:  "read",
	}
)

// type Permission string

// const (
// 	Read   Permission = "Read"
// 	Add               = "Add"
// 	Edit              = "Edit"
// 	Delete            = "Delete"
// 	All               = "All"
// )

// FROM go-shp
// //  is a identifier for the the type of shapes.
// type  int32

// // These are the possible shape types.
// const (
// 	NULL         = 0
// 	POINT        = 1
// 	POLYLINE     = 3
// 	POLYGON      = 5
// 	MULTIPOINT   = 8
// 	POINTZ       = 11
// 	POLYLINEZ    = 13
// 	POLYGONZ     = 15
// 	MULTIPOINTZ  = 18
// 	POINTM       = 21
// 	POLYLINEM    = 23
// 	POLYGONM     = 25
// 	MULTIPOINTM  = 28
// 	MULTIPATCH   = 31
// )

type Mode string

const (
	Prep      Mode = "prep"
	Upload         = "upload"
	Access         = "access"
	Elevation      = "elevation"
)

var (
	ModeReverse = map[string]Mode{
		"prep":      Prep,
		"upload":    Upload,
		"access":    Access,
		"elevation": Elevation,
	}
)
