package database

type Amount struct {
	id         int
	station_id int
	amount     int
	date       string
}

type Station_id struct {
	id           int
	station_name string
}
