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
