package database

import (
	"Flatservice/data"
	"database/sql"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func ConnectToDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", "Lexa:Admin_111@tcp(127.0.0.1:3306)/StationsDB")
	if err != nil {
		return nil, err
	}
	return db, nil
}
func CheckTable(db *sql.DB, name string) (string, error) {
	_, err := db.Query("select * from " + name + ";")
	if err != nil {
		return name + " not in DB", err
	}
	return name + " in DB already", nil
}

func CheckTables(db *sql.DB) error {
	// db, err := sql.Open("mysql", "Lexa:Admin_111@tcp(127.0.0.1:3306)/")
	// if err != nil {
	// 	return err
	// }
	// defer db.Close()
	_, amount_err := CheckTable(db, "Amount")
	if amount_err != nil {
		return amount_err
	}
	_, ids_err := CheckTable(db, "stations_id")
	if ids_err != nil {
		return ids_err
	}
	return nil
}

func Fill_Ids(db *sql.DB, name string) error {
	_, err := CheckTable(db, "stations_id")
	if err != nil {
		return err
	}
	absPath, _ := filepath.Abs("../data/" + name)
	f, err := ioutil.ReadFile(absPath)
	if err != nil {
		return errors.New(err.Error())
	}
	metro := data.Metro{}
	xml.Unmarshal(f, &metro)
	// db.Exec("SET sql_mode='ANSI_QUOTES'")
	for i := 0; i < len(metro.Location); i++ {
		qur := "insert ignore into Stations_id(station_id,station_name) values(?,?)"
		_, err := db.Query(qur, metro.Location[i].Id, metro.Location[i].Loc)
		if err != nil {
			return err
		}
	}
	return nil
}

func Insert_Station_Amount(station_id string, amount string, date time.Time) error {
	db, err := ConnectToDB()
	defer db.Close()
	if err != nil {
		return err
	}
	qur := "insert ignore into Amount(station_id,amount,date) values(?,?,?)"
	_, err = db.Query(qur, station_id, amount, date.Format("01-02-2006"))
	if err != nil {
		return err
	}
	return nil
}
