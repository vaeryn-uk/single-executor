package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"single-executor/internal/util"
	"single-executor/internal/watchdog"
	"strconv"
)

const templateDir = "/www/templates"
const watchdogConfigDir = "/www/watchdog-config"

var cluster watchdog.Cluster

func templateFile(name string) string {
	return fmt.Sprintf("%s/%s", templateDir, name)
}

type nodeData struct {
	RawJson []byte
	Name string
	Address string
	Others []string
}

func demo(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles(templateFile("demo.html"))

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	responses := make([]nodeData, 0)

	for _, node := range cluster.Nodes() {
		response, err := http.Get(node.HttpAddr() + "/state")

		nodeData := nodeData{
			[]byte("\"Could not retrieve data. Node is down?\""),
			strconv.Itoa(int(node.Id())),
			node.HttpAddr(),
			make([]string, 0),
		}

		for _, otherNode := range cluster.Nodes() {
			if node.Id() == otherNode.Id() {
				continue
			}

			nodeData.Others = append(nodeData.Others, strconv.Itoa(int(otherNode.Id())))
		}

		if err == nil {
			data, err := ioutil.ReadAll(response.Body)

			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			var indented bytes.Buffer

			err = json.Indent(&indented, data, "", "  ")

			nodeData.RawJson = indented.Bytes()

			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		} else {
			log.Println(err)
		}

		responses = append(responses, nodeData)
	}

	err = t.Execute(w, responses)

	if err != nil {
		http.Error(w, err.Error(), 500)
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

	http.HandleFunc("/dashboard", demo)
	http.HandleFunc("/network-sever", networkSever)

	err = http.ListenAndServe("0.0.0.0:80", nil)

	if err != nil {
		log.Fatalln(err)
	}
}

func networkSever(writer http.ResponseWriter, request *http.Request) {
	id, err := util.GetIntParam("id", request.URL.Query())

	if err != nil {
		http.Error(writer, "Must provide id", 400)
	}

	addr, err := cluster.HttpAddressFor(watchdog.Id(id))

	if err != nil {
		http.Error(writer, "Id does not exist", 400)
	}

	other, err := util.GetIntParam("other", request.URL.Query())

	if err != nil {
		http.Error(writer, "Must provide other", 400)
	}

	resp, err := http.Get(addr + "/blacklist?id=" + strconv.Itoa(other))

	if err != nil {
		http.Error(writer, err.Error(), 500)
	} else if resp.StatusCode != http.StatusOK {
		data, _ := ioutil.ReadAll(resp.Body)

		http.Error(writer, fmt.Sprintf("Request failed: %d - %s", resp.StatusCode, data), 500)
	} else {
		util.NoCache(writer)
		http.Redirect(writer, request, "/dashboard", http.StatusMovedPermanently)
	}
}