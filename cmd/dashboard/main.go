package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
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
	http.HandleFunc("/cluster-info", clusterInfo)
	http.HandleFunc("/node-state", nodeState)
	http.HandleFunc("/node-stop", nodeStop)
	http.HandleFunc("/node-start", nodeStart)
	http.HandleFunc("/network-sever", networkSever)
	http.Handle("/dist/", http.StripPrefix("/dist/",  http.FileServer(http.Dir(distDir))))

	err = http.ListenAndServe("0.0.0.0:80", nil)

	if err != nil {
		log.Fatalln(err)
	}
}

func nodeStart(writer http.ResponseWriter, request *http.Request) {
	id := extractNodeId(writer, request)

	if id == nil {
		return
	}

	dockerRun(writer, "start", *id)
}

func nodeStop(writer http.ResponseWriter, request *http.Request) {
	id := extractNodeId(writer, request)

	if id == nil {
		return
	}

	dockerRun(writer, "stop", *id)
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
	id := extractNodeId(w, request)

	if id == nil {
		return
	}

	addr, err := cluster.HttpAddressFor(*id)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	response, err := httpClient().Get(addr + "/state")

	if err != nil {
		if err := util.ResponseWithJson(w, nil); err != nil {
			http.Error(w, err.Error(), 500)
		}
		return
	}

	data, err := ioutil.ReadAll(response.Body)

	err = util.ResponseWithJson(w, data)

	if err != nil {
		fmt.Println(err)
	}
}

func networkSever(writer http.ResponseWriter, request *http.Request) {
	id := extractNodeId(writer, request)

	if id == nil {
		return
	}

	addr, err := cluster.HttpAddressFor(*id)

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

func extractNodeId(w http.ResponseWriter, request *http.Request) *watchdog.Id {
	id, err := util.GetIntParam("id", request.URL.Query())

	if err != nil {
		http.Error(w, "Must provide a node ID", 400)
		return nil
	}

	result := watchdog.Id(id)

	return &result
}

func dockerRun(w http.ResponseWriter, command string, id watchdog.Id) {
	cmd := exec.Command("docker-compose", "-p", "single-executor", command, fmt.Sprintf("validator%d", id))

	err := cmd.Run()

	if err != nil {
		output, _ := cmd.Output()

		http.Error(w, fmt.Sprintf("%s: %s. %s", cmd.String(), err.Error(), output), 500)
	}

	w.WriteHeader(200)
}
