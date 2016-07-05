package main

import (
	"fmt"
	"net/http"
	"runtime"
	"encoding/JSON"
)

//
// main
//
func main() {
	fmt.Printf("Initialising Boshing Sequence on %s....\n",
		runtime.GOOS)
	http.HandleFunc("/getanalytics", requestHandler)
	http.ListenAndServe(":8080", nil)
}


func requestHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("URL: %s\n", r.URL.String())
	fmt.Printf("METHOD: %s\n", r.Method)

	//
	// Only supported method is GET
	//
	if r.Method != "GET" {
		http.Error(w, "Invalid request method", 405)
	}

	//
	// Read query parameters
	//
	err := r.ParseForm()
	if err != nil	{
		fmt.Printf("Failed to parse form\n")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	//
	// "fields" parameter is an array
	//
	values := r.Form["fields"]
	fmt.Printf("Request specifies '%d' fields\n", len(values))

	if 0 < len(values) {
		for index := 0; index < len(values); index++ {
			fmt.Printf("Field %d: '%s'\n", index, values[index])
		}
	}


	//
	// Add response headers
	//
	w.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")


	//
	// Send JSON response
	//
	type analyticsResponse struct {
		Name string
		Time int64
	}

	myResponse := analyticsResponse{"ANiceResponse", 1294706395881547000}
	responseJSON, err := json.Marshal(myResponse)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(responseJSON)

}
