package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"splitwiseAngularAPI/controller"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var router = mux.NewRouter()
var Trace *log.Logger

func main() {

	//initialize logger
	logFilePathPtr := flag.String("log", "splitwiseAPIServer.log", "log file path - default splitwiseAPIServer.log will be used")

	//read config
	configFilePathPtr := flag.String("config", "config.json", "config file path - default splitwiseconfig.json will be used")
	flag.Parse()

	//controller logger
	traceFile, _ := os.OpenFile(*logFilePathPtr, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
	controller.InitLogger(traceFile)
	defer traceFile.Close()

	controller.InitializeConfig(*configFilePathPtr)

	//initialize router
	http.Handle("/", router)

	//add handlers
	router.HandleFunc("/", controller.IndexHandler)
	router.HandleFunc("/logout", controller.Logout).Methods("GET")
	router.HandleFunc("/expenses", controller.CompleteAuth)
	router.HandleFunc("/getGroups", controller.GetGroups).Methods("GET")
	router.HandleFunc("/GetGroupData", controller.GetGroupData).Methods("GET")
	router.HandleFunc("/GetGroupUsers", controller.GetGroupUsers).Methods("GET")

	//allow headers
	headers := handlers.AllowedMethods([]string{"X-Requested-With", "Content-Type", "Authorization"})
	methods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"})
	origins := handlers.AllowedOrigins([]string{"*"})

	//listen
	err := http.ListenAndServe(":9094", handlers.CORS(headers, methods, origins)(router))
	if err != nil {
		Trace.Fatal("ListenAndServe", err)
	}
}
