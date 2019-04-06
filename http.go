package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func HttpServer(addr string) {
	r := mux.NewRouter()
	r.HandleFunc("/services/{serviceName}/restart", ServiceRestartHandler).Methods("POST")
	r.HandleFunc("/services/{serviceName}/stop", ServiceStopHandler).Methods("POST")
	r.HandleFunc("/services/{serviceName}/start", ServiceStartHandler).Methods("POST")
	r.HandleFunc("/services", ServiceListHandler).Methods("GET")
	http.Handle("/", r)
	http.ListenAndServe(addr, nil)
}

func ServiceRestartHandler(w http.ResponseWriter, r *http.Request) {
	serviceName := mux.Vars(r)["serviceName"]
	log.Printf("restart: %s", serviceName)
	if isRunning, _ := manager.IsRunning(serviceName); isRunning {
		manager.Stop(serviceName)
	}

	err := manager.Start(serviceName)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())

	} else {
		w.WriteHeader(200)
		w.Write([]byte("restarted"))
	}
}
func ServiceStopHandler(w http.ResponseWriter, r *http.Request) {
	serviceName := mux.Vars(r)["serviceName"]
	log.Printf("stop: %s", serviceName)
	err := manager.Stop(serviceName)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())

	} else {
		w.WriteHeader(200)
		w.Write([]byte("stopped"))
	}
}
func ServiceStartHandler(w http.ResponseWriter, r *http.Request) {
	serviceName := mux.Vars(r)["serviceName"]
	log.Printf("start: %s", serviceName)
	err := manager.Start(serviceName)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, err.Error())

	} else {
		w.WriteHeader(200)
		w.Write([]byte("started"))
	}
}
func ServiceListHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	fmt.Fprint(w, "Name\tState\tPid\n")
	for _, s := range manager.List() {
		fmt.Fprintf(w, "%s\t%s\t%d\n", s.Name, s.State, s.Pid)
	}
}
