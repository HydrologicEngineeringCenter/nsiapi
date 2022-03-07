package stores

import (
	"fmt"
	"log"

	"di2e.net/cwbi/nsiv2-api/config"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

const NsiSelect = `SELECT fd_id,x,y,cbfips,occtype,yrbuilt,num_story,resunits,stacked,
					source,empnum,teachers,students,sqft,pop2amu65,pop2amo65,pop2pmu65,
					pop2pmo65,st_damcat,basement,bldgtype,found_ht,found_type,val_struct,
					val_cont,val_vehic,med_yr_blt,fipsentry,firmzone,o65disable,
					u65disable,ground_elv 
				   FROM nsi `

type Nsi struct {
	Fd_id      int32   `db:"fd_id" json:"fd_id"`
	X          float64 `db:"x" json:"x"`
	Y          float64 `db:"y" json:"y"`
	CbFips     string  `db:"cbfips" json:"cbfips"`
	Occtype    string  `db:"occtype" json:"occtype"`
	Yrbuilt    int32   `db:"yrbuilt" json:"yrbuilt"`
	Num_story  int32   `db:"num_story" json:"num_story"`
	Resunits   int32   `db:"resunits" json:"resunits"`
	Stacked    string  `db:"stacked" json:"stacked"`
	Source     string  `db:"source" json:"source"`
	Empnum     int32   `db:"empnum" json:"empnum"`
	Teachers   int32   `db:"teachers" json:"teachers"`
	Students   int32   `db:"students" json:"students"`
	Sqft       float64 `db:"sqft" json:"sqft"`
	Pop2amu65  int32   `db:"pop2amu65" json:"pop2amu65"`
	Pop2amo65  int32   `db:"pop2amo65" json:"pop2amo65"`
	Pop2pmu65  int32   `db:"pop2pmu65" json:"pop2pmu65"`
	Pop2pmo65  int32   `db:"pop2pmo65" json:"pop2pmo65"`
	St_damcat  string  `db:"st_damcat" json:"st_damcat"`
	Basement   int32   `db:"basement" json:"basement"`
	Bldgtype   string  `db:"bldgtype" json:"bldgtype"`
	Found_ht   float64 `db:"found_ht" json:"found_ht"`
	Found_type string  `db:"found_type" json:"found_type"`
	Val_struct float64 `db:"val_struct" json:"val_struct"`
	Val_cont   float64 `db:"val_cont" json:"val_cont"`
	Val_vehic  float64 `db:"val_vehic" json:"val_vehic"`
	Med_yr_blt int32   `db:"med_yr_blt" json:"med_yr_blt"`
	Fipsentry  int32   `db:"fipsentry" json:"fipsentry"`
	Firmzone   string  `db:"firmzone" json:"firmzone"`
	O65disable float64 `db:"o65disable" json:"o65disable"`
	U65disable float64 `db:"u65disable" json:"u65disable"`
	Ground_elv float64 `db:"ground_elv" json:"ground_elv"`
}

const NsiStatsSelect = `select
							count(fd_id) as num_structures, 
							min(yrbuilt) as yrbuilt_min,
							max(yrbuilt) as yrbuilt_max,
							avg(num_story) as num_story_mean,
							sum(resunits) as resunits_sum,
							sum(empnum) as empnum_sum,
							sum(teachers) as teachers_sum,
							sum(students) as students_sum,
							avg(sqft) as sqft_mean,
							sum(sqft) as sqft_sum,
							sum(pop2amu65) as pop2amu65_sum,
							sum(pop2amo65) as pop2amo65_sum,
							sum(pop2pmu65) as pop2pmu65_sum,
							sum(pop2pmo65) as pop2pmo65_sum,
							sum(val_struct) as val_struct_sum,
							sum(val_cont) as val_cont_sum,
							sum(val_vehic) as val_vehic_sum,
							min(med_yr_blt) as med_yr_blt_min, 
							max(med_yr_blt) as med_yr_blt_max,
							max(ground_elv)as ground_elv_max,
							min(ground_elv) as ground_elv_min
							from nsi `

type NsiSummary struct {
	Num_structures int64   `db:"num_structures" json:"num_structures"`
	Yrbuilt_min    int32   `db:"yrbuilt_min" json:"yrbuilt_min"`
	Yrbuilt_max    int32   `db:"yrbuilt_max" json:"yrbuilt_max"`
	Num_story_mean float64 `db:"num_story_mean" json:"num_story_mean"`
	Resunits_sum   int64   `db:"resunits_sum" json:"resunits_sum"`
	Empnum_sum     int64   `db:"empnum_sum" json:"empnum_sum"`
	Teachers_sum   int64   `db:"teachers_sum" json:"teachers_sum"`
	Students_sum   int64   `db:"students_sum" json:"students_sum"`
	Sqft_mean      float64 `db:"sqft_mean" json:"sqft_mean"`
	Sqft_sum       float64 `db:"sqft_sum" json:"sqft_sum"`
	Pop2amu65_sum  int64   `db:"pop2amu65_sum" json:"pop2amu65_sum"`
	Pop2amo65_sum  int64   `db:"pop2amo65_sum" json:"pop2amo65_sum"`
	Pop2pmu65_sum  int64   `db:"pop2pmu65_sum" json:"pop2pmu65_sum"`
	Pop2pmo65_sum  int64   `db:"pop2pmo65_sum" json:"pop2pmo65_sum"`
	Val_struct_sum float64 `db:"val_struct_sum" json:"val_struct_sum"`
	Val_cont_sum   float64 `db:"val_cont_sum" json:"val_cont_sum"`
	Val_vehic_sum  float64 `db:"val_vehic_sum" json:"val_vehic_sum"`
	Med_yr_blt_min int32   `db:"med_yr_blt_min" json:"med_yr_blt_min"`
	Med_yr_blt_max int32   `db:"med_yr_blt_max" json:"med_yr_blt_max"`
	Ground_elv_max float64 `db:"ground_elv_max" json:"ground_elv_max"`
	Ground_elv_min float64 `db:"ground_elv_min" json:"ground_elv_min"`
}

const HexbinSelect = `select
							id,
							st_asbinary(shape) as shape,
							num_structures, 
							yrbuilt_min,
							yrbuilt_max,
							num_story_mean,
							resunits_sum,
							empnum_sum,
							teachers_sum,
							students_sum,
							sqft_mean,
							sqft_sum,
							pop2amu65_sum,
							pop2amo65_sum,
							pop2pmu65_sum,
							pop2pmo65_sum,
							val_struct_sum,
							val_cont_sum,
							val_vehic_sum,
							med_yr_blt_min, 
							med_yr_blt_max,
							ground_elv_max,
							ground_elv_min
							from %s `

type Hexbin struct {
	ID    int32  `db:"id" json:"id"`
	Shape []byte `db:"shape" json:"shape"`
	NsiSummary
}

var HexbinDatasets = map[string]string{
	"hb10k":  "hexbin_10000",
	"hb2500": "hexbin_2500",
	"hb500":  "hexbin_500",
}

type DbStore struct {
	Db *sqlx.DB
}

func InitDbStore(appConfig config.AppConfig) (*DbStore, error) {
	log.Printf("Connecting to: %s/%s", appConfig.Dbhost, appConfig.Dbname)
	log.Printf("Using pool size of %d", appConfig.DbMaxConnections)
	dburl := fmt.Sprintf("user=%s password=%s host=%s port=5432 database=%s sslmode=disable",
		appConfig.Dbuser, appConfig.Dbpass, appConfig.Dbhost, appConfig.Dbname)
	con, err := sqlx.Connect("pgx", dburl)
	if err != nil {
		log.Println(err)
	}
	con.SetMaxOpenConns(appConfig.DbMaxConnections)
	store := DbStore{
		Db: con,
	}
	return &store, err
}
