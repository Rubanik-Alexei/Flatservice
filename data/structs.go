package data

import (
	"encoding/xml"
)

type Metro struct {
	XMLName  xml.Name   `xml:"metro"`
	Location []Location `xml:"location"`
}
type Location struct {
	Loc string `xml:",chardata"`
	Id  string `xml:"id,attr"`
}

type Amount struct {
	Id         int //`field:"id"`
	Station_id int
	Amount     int
	Date       string
}
