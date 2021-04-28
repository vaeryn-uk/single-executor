package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"single-executor/internal/watchdog"
	"strconv"
)

const templateDir = "/www/templates"
const watchdogConfigDir = "/www/watchdog-config"

var cluster watchdog.Cluster

func writeError(writer http.ResponseWriter, err error) {
	writer.WriteHeader(500)
	if _, err = writer.Write([]byte(err.Error())); err != nil {
		log.Fatalln(err)
	}
}

func templateFile(name string) string {
	return fmt.Sprintf("%s/%s", templateDir, name)
}

type nodeData struct {
	RawJson []byte
	Name string
	Address string
}

func demo(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles(templateFile("demo.html"))

	if err != nil {
		writeError(w, err)
		return
	}

	responses := make([]nodeData, 0)

	for _, node := range cluster.Nodes() {
		response, err := http.Get(node.HttpAddr())

		nodeData := nodeData{
			[]byte("\"Could not retrieve data. Node is down?\""),
			strconv.Itoa(int(node.Id())),
			node.HttpAddr(),
		}

		if err == nil {
			data, err := ioutil.ReadAll(response.Body)

			if err != nil {
				writeError(w, err)
				return
			}

			var indented bytes.Buffer

			err = json.Indent(&indented, data, "", "  ")

			nodeData.RawJson = indented.Bytes()

			if err != nil {
				writeError(w, err)
				return
			}
		} else {
			log.Println(err)
		}

		responses = append(responses, nodeData)
	}

	err = t.Execute(w, responses)

	if err != nil {
		writeError(w, err)
	}
}

func main() {
	in, err := ioutil.ReadFile(watchdogConfigDir + "/watchdog.cluster.yaml")

	if err != nil {
		log.Fatalln(err)
	}

	cluster, err = watchdog.ParseCluster(in)

	if err != nil {
		log.Fatalln(err)
	}

	http.HandleFunc("/demo", demo)

	err = http.ListenAndServe("0.0.0.0:80", nil)

	if err != nil {
		log.Fatalln(err)
	}
}