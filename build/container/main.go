package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/lendi-au/helm-janitor/cmd/scan"
	"github.com/lendi-au/helm-janitor/internal/config"
	log "github.com/sirupsen/logrus"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	logLevel := "info"
	if os.Getenv("LOG_LEVEL") != "" {
		logLevel = os.Getenv("LOG_LEVEL")
	}
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Errorf("Dodgy log level set: %s", logLevel)
		log.SetLevel(log.WarnLevel)
	} else {
		log.SetLevel(level)
	}
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Access-Token")
	AUTH_TOKEN := config.GetenvWithDefault("HTTP_AUTH_TOKEN", "magic")
	scanner := scan.NewScanClient()
	if token == AUTH_TOKEN {
		fmt.Fprintf(w, "You have some magic in you\n")
		log.Println("Allowed an access attempt")
	} else {
		http.Error(w, "You don't have enough magic in you", http.StatusForbidden)
		log.Println("Denied an access attempt")
	}
	scanner.Dryrun = config.GetenvWithDefaultBool("DRY_RUN", false)
	switch r.Method {
	case "GET":
		log.Info("GET yo")
	case "POST":
		log.Debugf("Post yo")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		fmt.Print(string(body[:]))
		test, _ := json.Marshal(string(body[:]))
		log.Debugf("body was: %v", test)
	}
}

func main() {
	http.HandleFunc("/", mainHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
