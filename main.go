package main

import (
	"Flatservice/handlers"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
)

/*
type Metro struct {
	XMLName  xml.Name   `xml:"metro"`
	Location []Location `xml:"location"`
}
type Location struct {
	Loc string `xml:",chardata"`
	Id  string `xml:"id,attr"`
	//Loc string `xml:"location"`
}
*/

// func BetterGetAmounts() {
// 	url := string("https://spb.cian.ru/snyat-kvartiru/")
// 	client := &http.Client{}
// 	req, err := http.NewRequest("GET", url, nil)

// 	req.Header.Set("User-Agent", uarand.GetRandom())
// 	resp, err := client.Do(req)
// 	//cont, err := soup.GetWithClient(url,client)
// 	if err != nil {
// 		//http.Error(rw, "Problem with cian site", http.StatusForbidden)
// 		return
// 	}
// 	defer resp.Body.Close()
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		//http.Error(rw, "Problem with cian site", http.StatusForbidden)
// 		return
// 	}
// 	res := soup.HTMLParse(string(body))
// 	if err != nil {
// 		//http.Error(rw, "Problem with cian site", http.StatusForbidden)
// 		return
// 	}

// 	result := exec.Command()
// 	ss := result.CombinedOutput()
// 	fmt.Println(ss)
// 	stations := make([]string, 0)
// 	amounts := make([]string, 0)
// 	f, err := ioutil.ReadFile("metros-petersburg.xml")
// 	if err != nil {
// 		panic(err)
// 	}
// 	metro := data.Metro{}
// 	xml.Unmarshal(f, &metro)
// 	xlsx := excelize.NewFile()
// 	size := len(metro.Location)
// 	//for testing
// 	// size = 5
// 	xlsx.SetSheetRow("Sheet1", "A1", &[]string{"Станция", "Число объявлений"})
// 	for i := 0; i < size; i++ {

// 		stations = append(stations)
// 		amounts = append(amounts)
// 		time.Sleep(2000000000)
// 	}
// 	fmt.Println(amounts)
// 	fmt.Println(stations)
// 	// for j := 0; j < size; j++ {
// 	// 	xlsx.SetCellValue("Sheet1", fmt.Sprintf("A%d", j+2), stations[j])
// 	// 	xlsx.SetCellValue("Sheet1", fmt.Sprintf("B%d", j+2), amounts[j])
// 	// }

// 	// rw.Header().Set("Content-Disposition", "attachment; filename="+"amount_all"+".xlsx")
// 	// rw.Header().Set("Content-Transfer-Encoding", "binary")
// 	// rw.Header().Set("Expires", "0")
// 	// xlsx.Write(rw)
// }

func main() {
	// database.CheckTables()
	l := log.New(os.Stdout, "products-api ", log.LstdFlags)

	// create the handlers
	ph := handlers.NewLog(l)

	// create a new serve mux and register the handlers
	sm := mux.NewRouter()

	getRouter := sm.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/amount/{station}", ph.GetAmount)
	getRouter.HandleFunc("/adress/{station}", ph.GetAdresses)
	getRouter.HandleFunc("/all_adress", ph.GetAllAdresses)
	getRouter.HandleFunc("/all_amount", ph.GetAllAmount)
	getRouter.HandleFunc("/history/{station}", ph.GetStationFromDB)
	//getRouter.HandleFunc("/", ph.GetAmount)

	//sm.Handle("/products", ph)

	// create a new server
	s := http.Server{
		Addr:         ":9090",           // configure the bind address
		Handler:      sm,                // set the default handler
		ErrorLog:     l,                 // set the logger for the server
		ReadTimeout:  10 * time.Second,  // max time to read request from the client
		WriteTimeout: 700 * time.Second, // max time to write response to the client
		IdleTimeout:  700 * time.Second, // max time for connections using TCP Keep-Alive
	}

	// start the server
	go func() {
		l.Println("Starting server on port 9090")

		err := s.ListenAndServe()
		if err != nil {
			l.Printf("Error starting server: %s\n", err)
			os.Exit(1)
		}
	}()

	// trap sigterm or interupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// Block until a signal is received.
	sig := <-c
	log.Println("Got signal:", sig)

	// gracefully shutdown the server, waiting max 30 seconds for current operations to complete
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(ctx)

	// /*
	// 	f, err := ioutil.ReadFile("metros-petersburg.xml")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	metro := Metro{}
	// 	subway_id := "0"
	// 	xml.Unmarshal(f, &metro)

	// 	// vars := mux.Vars(r)
	// 	// station := vars["station"]
	// 	station := "Выборгская"
	// 	for i := 0; i < len(metro.Location); i++ {
	// 		if metro.Location[i].Loc == station {
	// 			subway_id = metro.Location[i].Id
	// 		}
	// 	}
	// 	if subway_id == "0" {
	// 		//http.Error(rw, "Incorrect station name", http.StatusBadRequest)
	// 		return
	// 	}
	// 	url := string("https://spb.cian.ru/cat.php?deal_type=rent&engine_version=2&foot_min=45&metro%5B0%5D=" + subway_id + "&offer_type=flat&only_foot=2&p=1&region=2&type=4")
	// 	cont, err := soup.Get(url)
	// 	if err != nil {
	// 		//http.Error(rw, "Problem with target site", http.StatusForbidden)
	// 		return
	// 	}
	// 	res := soup.HTMLParse(cont)
	// 	string_amount := res.Find("div", "class", "_93444fe79c--header--3pKNW").Children()[0].FullText()
	// 	re := regexp.MustCompile(`[0-9]+`)
	// 	amount := re.FindString(string_amount)
	// 	fmt.Println(amount)
	// */
}
