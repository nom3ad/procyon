package main

import (
	"io/ioutil"
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"
)

var composeFile ComposeFileStruct

var manager *Manager

func main() {
	addr := os.Args[1]
	configFile := os.Args[2]

	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(bytes, &composeFile)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	log.Println("Loaded compose file", composeFile)
	manager = NewManger(composeFile)
	log.Println("Starting process mananger")
	manager.run()
	log.Println("Starting http server", addr)
	HttpServer(addr)
}
