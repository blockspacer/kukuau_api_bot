package main
import (
	"net/http"
	"fmt"
	"log"
	t "msngr/taxi"
	m "msngr"
	d "msngr/db"

	"encoding/json"
)

func main() {

	conf := m.ReadConfig()

	d.DELETE_DB = true
	if d.DELETE_DB {
		log.Println("!start at test mode!")
		conf.Database.Name = conf.Database.Name + "_test"
	}


	taxi_conf := conf.Taxis["fake"]
	address_supplier := t.NewGoogleAddressHandler(conf.Main.GoogleKey, taxi_conf.GeoOrbit)

	streets_address := fmt.Sprintf("/taxi/%v/streets", taxi_conf.Name)

	http.HandleFunc(streets_address, func(w http.ResponseWriter, r *http.Request) {
		t.StreetsSearchController(w, r, address_supplier)
	})

	server_address := fmt.Sprintf(":%v", conf.Main.Port)
	server := &http.Server{
		Addr: server_address,
	}
	test_url := "http://localhost" + server_address + streets_address
	log.Printf("start server... send tests to: %v?=", test_url)

	go server.ListenAndServe()

	for _, q := range []string{"ktc", "ktcj", "ktcjc", "лес", "лесосе"} {
		log.Printf(">>> %v", q)
		body, err := t.GET(test_url, &map[string]string{"q":"лес"})
		if body != nil {
			log.Printf("<<<<<< %q", string(*body))
			var results []t.DictItem
			err = json.Unmarshal(*body, &results)
			log.Printf("err: %v \nunmarshaled:%+v", err, results)
		}
		if err != nil {
			log.Printf("!!!ERRRR!!! %+v", err)
		}
	}


}