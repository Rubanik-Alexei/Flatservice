package handlers

import (
	"testing"
)

func TestGetId(t *testing.T) {
	res, err := GetStationId("Парнас")
	if err != nil {
		t.Fatal(err)
	}
	if res != "186" {
		t.Fatal("wrong id")
	}
	wrongres, _ := GetStationId("Типостанция")
	if wrongres != "0" {
		t.Fatal("Несуществующая станция")
	}
}
