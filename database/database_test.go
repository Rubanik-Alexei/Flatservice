package database

import (
	//"database/sql"
	"testing"
)

func TestConnect(t *testing.T) {
	db, err := ConnectToDB()
	if err != nil {
		t.Fatal()
	}
	defer db.Close()
	// _, err = db.Query("USE StationsDB")
	// if err != nil {
	// 	t.Fatal()
	// }
	CheckTables(db)

	id_err := Fill_Ids(db, "metros-petersburg.xml")
	if id_err != nil {
		t.Log(id_err)
		t.Fatal()
	}
}
