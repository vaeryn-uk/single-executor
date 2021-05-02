package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"single-executor/internal/util"
	"single-executor/internal/watchdog"
	"strconv"
	"time"
)

const templateDir = "/www/templates"
const distDir = "/www/dist"
const watchdogConfigDir = "/www/watchdog-config"

var cluster watchdog.Cluster
var resolvedHttpClient *http.Client

func templateFile(name string) string {
	return fmt.Sprintf("%s/%s", templateDir, name)
}

type nodeData struct {
	StateEndpoint string `json:"stateEndpoint"`
	Name string `json:"name"`
	Address string `json:"address"`
	Others []string `json:"peers"`
}

func dashboard(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles(templateFile("dashboard.html"))

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = t.Execute(w, struct{}{})

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

	http.HandleFunc("/dashboard", dashboard)
	http.HandleFunc("/network", networkState)
	http.HandleFunc("/cluster-info", clusterInfo)
	http.HandleFunc("/node-state", nodeState)
	http.HandleFunc("/network-sever", networkSever)
	http.Handle("/dist/", http.StripPrefix("/dist/",  http.FileServer(http.Dir(distDir))))

	err = http.ListenAndServe("0.0.0.0:80", nil)

	if err != nil {
		log.Fatalln(err)
	}
}

func clusterInfo(w http.ResponseWriter, request *http.Request) {
	type nodeInfo struct {
		Id int `json:"id"`
	}

	result := struct{
		Nodes []nodeInfo `json:"nodes"`
	}{
		make([]nodeInfo, 0),
	}

	for _, node := range cluster.Nodes() {
		result.Nodes = append(result.Nodes, nodeInfo{int(node.Id())})
	}

	data, err := json.Marshal(result)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := util.ResponseWithJson(w, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func nodeState(w http.ResponseWriter, request *http.Request) {
	id, err := util.GetIntParam("id", request.URL.Query())

	if err != nil {
		http.Error(w, "Must provide a node ID", 400)
		return
	}

	addr, err := cluster.HttpAddressFor(watchdog.Id(id))

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	response, err := httpClient().Get(addr + "/state")

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data, err := ioutil.ReadAll(response.Body)

	err = util.ResponseWithJson(w, data)

	if err != nil {
		fmt.Println(err)
	}
}

func networkState(w http.ResponseWriter, request *http.Request) {
	responses := make([]nodeData, 0)

	for _, node := range cluster.Nodes() {
		nodeData := nodeData{
			fmt.Sprintf("/node-state?id=%d", node.Id()),
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

		responses = append(responses, nodeData)
	}

	data, err := json.Marshal(responses)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(data); err != nil {
		log.Println(err)
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

	resp, err := httpClient().Get(addr + "/blacklist?id=" + strconv.Itoa(other))

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

func httpClient() *http.Client {
	if resolvedHttpClient == nil {
		resolvedHttpClient = new(http.Client)

		// 0.5 second timeout
		resolvedHttpClient.Timeout = time.Duration(500 * time.Millisecond)
	}

	return resolvedHttpClient
}
