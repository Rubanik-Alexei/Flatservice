package handlers

import (
	"Flatservice/data"
	"Flatservice/database"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/corpix/uarand"
	"github.com/gorilla/mux"
	"github.com/streadway/amqp"
	"github.com/xuri/excelize/v2"
)

type MyLog struct {
	l *log.Logger
}

func NewLog(l *log.Logger) *MyLog {
	return &MyLog{l}
}
func GetStationId(station_name string) (string, error) {
	absPath, eer := filepath.Abs("../Flatservice/data/metros-petersburg.xml")
	if eer != nil {
		return "-1", errors.New(eer.Error())
	}
	f, err := ioutil.ReadFile(absPath)
	if err != nil {
		return "-1", errors.New(err.Error())
	}
	metro := data.Metro{}
	subway_id := "0"
	xml.Unmarshal(f, &metro)
	for i := 0; i < len(metro.Location); i++ {
		if metro.Location[i].Loc == station_name {
			subway_id = metro.Location[i].Id
			break
		}
	}
	return subway_id, nil
}
func GetCian(uurl string, prx *url.URL, wait time.Duration) (soup.Root, error) {
	//client := &http.Client{}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(prx)}}
	req, err := http.NewRequest("GET", uurl, nil)
	res := soup.Root{}
	//req.Header.Set("User-Agent", uarand.GetRandom())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.54 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	time.Sleep(wait)
	body, err1 := ioutil.ReadAll(resp.Body)
	if err1 != nil {
		return res, err1
	}
	res = soup.HTMLParse(string(body))
	return res, nil
}

func SendUpdateMessage(station_name string) (error, string) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return err, "Failed to connect to RabbitMQ"
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err, "Failed to open a channel"
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"Station_DB_Update", // name
		"topic",             // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return err, "Failed to declare an exchange"
	}
	text := "Data in DB for station " + station_name + " has been updated"
	err = ch.Publish(
		"Station_DB_Update", // exchange
		station_name,        // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(text),
		})
	if err != nil {
		return err, "Failed to publish a message"
	}
	log.Printf(" [x] Sent %s", text)
	return nil, "OK"
}

func (p *MyLog) GetAmount(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	station := vars["station"]
	p.l.Println("Handle GET Amount")
	subway_id, err := GetStationId(station)
	if err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	} else if subway_id == "0" {
		http.Error(rw, "Incorrect station name", http.StatusBadRequest)
		return
	}
	url := string("https://spb.cian.ru/cat.php?deal_type=rent&engine_version=2&foot_min=45&metro%5B0%5D=" + subway_id + "&offer_type=flat&only_foot=2&p=1&region=2&type=4")
	res, err := GetCian(url, nil, 0*time.Second)
	if err != nil {
		http.Error(rw, "Problem with cian site", http.StatusBadGateway)
		return
	}
	//fmt.Println(res)
	string_amount := res.Find("div", "class", "_93444fe79c--header--3pKNW").Children()[0].FullText()
	re := regexp.MustCompile(`[0-9]+`)
	amount := re.FindString(string_amount)
	e := json.NewEncoder(rw)
	db_err := database.Insert_Station_Amount(subway_id, amount, time.Now())
	if db_err != nil {
		http.Error(rw, "Cannot add amount to db", http.StatusInternalServerError)
	}
	err, comment := SendUpdateMessage(station)
	if err != nil {
		http.Error(rw, comment, http.StatusInternalServerError)
	}
	err = e.Encode(amount)
	if err != nil {
		http.Error(rw, "Unable to marshal json", http.StatusInternalServerError)
	}
}

func (p *MyLog) GetAllAmount(rw http.ResponseWriter, r *http.Request) {
	f, err := ioutil.ReadFile("metros-petersburg.xml")
	if err != nil {
		panic(err)
	}
	metro := data.Metro{}
	xml.Unmarshal(f, &metro)
	xlsx := excelize.NewFile()
	size := len(metro.Location)
	//for testing
	// size = 5
	xlsx.SetSheetRow("Sheet1", "A1", &[]string{"Станция", "Число объявлений"})
	c := make(chan string, 2*size)
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = len(metro.Location)
	prxs := GetProxiesList()
	var wg sync.WaitGroup
	cntr := 0
	for i := 0; i < size; i++ {
		wg.Add(1)
		go GetStationAmount(&wg, rw, metro.Location[i].Loc, metro.Location[i].Id, c, prxs[cntr])
		cntr++
		time.Sleep(2000000000)
	}
	wg.Wait()
	for j := 0; j < size; j++ {
		tmpstations, tmpamount := <-c, <-c
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("A%d", j+2), tmpstations)
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("B%d", j+2), tmpamount)
	}

	rw.Header().Set("Content-Disposition", "attachment; filename="+"amount_all"+".xlsx")
	rw.Header().Set("Content-Transfer-Encoding", "binary")
	rw.Header().Set("Expires", "0")
	xlsx.Write(rw)
}

func GetStationAmount(wg *sync.WaitGroup, rw http.ResponseWriter, station string, subway_id string, c chan string, proxi string) {
	defer wg.Done()
	var string_amount soup.Root = soup.Root{nil, "", nil}
	fmt.Println("handle amount " + station)
	//proxyUrl, err := url.Parse("http://" + proxi)
	myurl := string("https://spb.cian.ru/cat.php?deal_type=rent&engine_version=2&foot_min=45&metro%5B0%5D=" + subway_id + "&offer_type=flat&only_foot=2&p=1&region=2&type=4")
	fmt.Println(myurl)

	res, err := GetCian(myurl, nil, 5*time.Second)
	if err != nil {
		http.Error(rw, "Problem with cian site", http.StatusBadGateway)
		return
	}
	fmt.Println(res)
	string_amount = res.Find("div", "class", "_93444fe79c--header--3pKNW") //[0].FullText()
	time.Sleep(1000000000)
	if string_amount.Error != nil {
		c <- station
		c <- "-2"
		return
	}
	fmt.Println(string_amount)
	re := regexp.MustCompile(`[0-9]+`)
	amount := re.FindString(string_amount.Children()[0].FullText())
	c <- station
	c <- amount

}
func (p *MyLog) GetStationFromDB(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	station := vars["station"]
	p.l.Println("Handle GET From DB")
	subway_id, err := GetStationId(station)
	if err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	} else if subway_id == "0" {
		http.Error(rw, "Incorrect station name", http.StatusBadRequest)
		return
	}
	hstr_rows, err := database.Get_Amount_DB(subway_id)
	if err != nil {
		http.Error(rw, "Cannot connect to DB", http.StatusInternalServerError)
		return
	}
	res := make([]data.Amount, 0)
	defer hstr_rows.Close()
	for hstr_rows.Next() {
		tmp := data.Amount{}
		err := hstr_rows.Scan(&tmp.Id, &tmp.Station_id, &tmp.Amount, &tmp.Date)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(rw, "No information for station "+station, http.StatusBadRequest)
				return
			} else {
				http.Error(rw, "Cannot retrieve data from DB", http.StatusBadRequest)
				return
			}
		}
		res = append(res, tmp)
	}
	xlsx := excelize.NewFile()
	xlsx.SetSheetRow("Sheet1", "A1", &[]string{"id", "Станция", "Число объявлений", "дата"})
	for j, v := range res {
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("A%d", j+2), v.Id)
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("B%d", j+2), station)
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("C%d", j+2), v.Amount)
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("D%d", j+2), v.Date)
	}
	rw.Header().Set("Content-Disposition", "attachment; filename="+station+"DB"+".xlsx")
	rw.Header().Set("Content-Transfer-Encoding", "binary")
	rw.Header().Set("Expires", "0")
	xlsx.Write(rw)
}

func (p *MyLog) GetAdresses(rw http.ResponseWriter, r *http.Request) {
	stations := make([]string, 0)
	adresses := make([]string, 0)
	vars := mux.Vars(r)
	station := vars["station"]
	p.l.Println("Handle GET Amount")
	subway_id, err := GetStationId(station)
	if err != nil {
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	} else if subway_id == "0" {
		http.Error(rw, "Incorrect station name", http.StatusBadRequest)
		return
	}
	url := string("https://spb.cian.ru/cat.php?deal_type=rent&engine_version=2&foot_min=45&metro%5B0%5D=" + subway_id + "&offer_type=flat&only_foot=2&p=1&region=2&type=4")
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	req.Header.Set("User-Agent", uarand.GetRandom())
	resp, err := client.Do(req)
	if err != nil {
		http.Error(rw, "Problem with cian site", http.StatusForbidden)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(rw, "Problem with cian site", http.StatusForbidden)
		return
	}
	res := soup.HTMLParse(string(body))
	if err != nil {
		http.Error(rw, "Problem with cian site", http.StatusForbidden)
		return
	}
	string_amount := res.Find("div", "class", "_93444fe79c--header--3pKNW").Children()[0].FullText()
	re := regexp.MustCompile(`[0-9]+`)
	amount := re.FindString(string_amount)
	int_amount, _ := strconv.Atoi(amount)
	page_amount := int_amount / 28
	if int(page_amount)%28 != 0 {
		page_amount += 1
	}
	cnt := 1
	for cnt != page_amount {
		res_adress := res.FindAll("div", "class", "_93444fe79c--content--2IC7j")
		for _, value := range res_adress {
			adrs := value.Find("div", "class", "_93444fe79c--labels--1J6M3").FindAll("a")
			tmp := ""
			for _, val := range adrs {
				tmp += val.Text()
				tmp += ", "
			}
			stations = append(stations, station)
			adresses = append(adresses, tmp[:len(tmp)-2])
		}
		cnt++
		url = "https://spb.cian.ru/cat.php?deal_type=rent&engine_version=2&foot_min=45&metro%5B0%5D=" + station + "&offer_type=flat&only_foot=2&p=" + strconv.Itoa(cnt) + "&region=2&type=4"
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)

		req.Header.Set("User-Agent", uarand.GetRandom())
		resp, err := client.Do(req)
		if err != nil {
			http.Error(rw, "Problem with cian site", http.StatusForbidden)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(rw, "Problem with cian site", http.StatusForbidden)
			return
		}
		res = soup.HTMLParse(string(body))
	}
	xlsx := excelize.NewFile()
	xlsx.SetSheetRow("Sheet1", "A1", &[]string{"Станция", "Адрес"})
	for i := 0; i < len(adresses); i++ {
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+2), stations[i])
		xlsx.SetCellValue("Sheet1", fmt.Sprintf("B%d", i+2), adresses[i])
	}
	//rw.Header().Set("Content-Type", "application/octet-stream")
	rw.Header().Set("Content-Disposition", "attachment; filename="+station+".xlsx")
	rw.Header().Set("Content-Transfer-Encoding", "binary")
	rw.Header().Set("Expires", "0")
	xlsx.Write(rw)

}

func (p *MyLog) GetAllAdresses(rw http.ResponseWriter, r *http.Request) {
	f, err := ioutil.ReadFile("metros-petersburg.xml")
	if err != nil {
		panic(err)
	}
	metro := data.Metro{}
	xml.Unmarshal(f, &metro)
	xlsx := excelize.NewFile()
	xlsx.SetSheetRow("Sheet1", "A1", &[]string{"Станция", "Адрес"})
	c := make(chan []string, 2*len(metro.Location))
	prxs := GetProxiesList()
	var wg sync.WaitGroup
	cntr := 0
	for i := 0; i < len(metro.Location); i++ {
		wg.Add(1)
		go GetStation(&wg, rw, metro.Location[i].Loc, metro.Location[i].Id, c, prxs[cntr])
		cntr++
		time.Sleep(500000000)
	}
	wg.Wait()
	for j := 0; j < len(metro.Location); j++ {
		tmpstations, tmpadresses := <-c, <-c
		for i := 0; i < len(tmpadresses); i++ {
			xlsx.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+2), tmpstations[i])
			xlsx.SetCellValue("Sheet1", fmt.Sprintf("B%d", i+2), tmpadresses[i])
		}
	}
	rw.Header().Set("Content-Disposition", "attachment; filename="+"all"+".xlsx")
	rw.Header().Set("Content-Transfer-Encoding", "binary")
	rw.Header().Set("Expires", "0")
	xlsx.Write(rw)
}

func GetStation(wg *sync.WaitGroup, rw http.ResponseWriter, station string, subway_id string, c chan []string, proxi string) {
	defer wg.Done()
	fmt.Println("handle" + station)
	stations := make([]string, 0)
	adresses := make([]string, 0)
	proxyUrl, err := url.Parse("http://" + proxi)
	url := string("https://spb.cian.ru/cat.php?deal_type=rent&engine_version=2&foot_min=45&metro%5B0%5D=" + subway_id + "&offer_type=flat&only_foot=2&p=1&region=2&type=4")
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", uarand.GetRandom())
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	res := soup.HTMLParse(string(body))
	string_amount := res.Find("div", "class", "_93444fe79c--header--3pKNW").Children()[0].FullText()
	re := regexp.MustCompile(`[0-9]+`)
	amount := re.FindString(string_amount)
	int_amount, _ := strconv.Atoi(amount)
	page_amount := int_amount / 28
	if int(page_amount)%28 != 0 {
		page_amount += 1
	}
	cnt := 1
	for cnt != page_amount {
		res_adress := res.FindAll("div", "class", "_93444fe79c--content--2IC7j")
		for _, value := range res_adress {
			adrs := value.Find("div", "class", "_93444fe79c--labels--1J6M3").FindAll("a")
			tmp := ""
			for _, val := range adrs {
				tmp += val.Text()
				tmp += ", "
			}
			stations = append(stations, station)
			adresses = append(adresses, tmp[:len(tmp)-2])
		}
		cnt++
		url = "https://spb.cian.ru/cat.php?deal_type=rent&engine_version=2&foot_min=45&metro%5B0%5D=" + station + "&offer_type=flat&only_foot=2&p=" + strconv.Itoa(cnt) + "&region=2&type=4"
		time.Sleep(15 * time.Second)
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", uarand.GetRandom())
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		res = soup.HTMLParse(string(body))
	}
	c <- stations
	c <- adresses
}
