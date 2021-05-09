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
const watchdogClusterConfig = watchdogConfigDir + "/watchdog.cluster.yaml"
const watchdogInstanceConfig = watchdogConfigDir + "/watchdog.instance.yaml"

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
	http.HandleFunc("/network-break", func(writer http.ResponseWriter, request *http.Request) {
		networkModify(writer, request, "blacklist")
	})
	http.HandleFunc("/network-repair", func(writer http.ResponseWriter, request *http.Request) {
		networkModify(writer, request, "whitelist")
	})
	http.Handle("/dist/", http.StripPrefix("/dist/",  http.FileServer(http.Dir(distDir))))
	http.HandleFunc("/config/cluster", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, watchdogClusterConfig)
	})
	http.HandleFunc("/config/instance", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, watchdogInstanceConfig)
	})

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
		http.NotFound(w, request)
		return
	}

	addr, err := cluster.HttpAddressFor(*id)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	messages, done, serve := util.HandleSse(w, request)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				response, err := httpClient().Get(addr + "/state")

				var data []byte

				if err != nil {
					data, err = json.Marshal(struct{
						Id int `json:"id"`
						State string `json:"state"`
					}{int(*id), "down"})

					if err != nil {
						log.Printf("%s\n", err.Error())
						continue
					}
				} else {
					data, err = ioutil.ReadAll(response.Body)
				}

				messages <- data

				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	serve()
}

func networkModify(writer http.ResponseWriter, request *http.Request, operation string) {
	id := extractNodeId(writer, request)

	if id == nil {
		return
	}

	otherId, err := util.GetIntParam("other", request.URL.Query())

	if err != nil {
		http.Error(writer, "Must provide other", 400)
	}

	if err := modifyNetworkBetweenNodes(*id, watchdog.Id(otherId), operation); err != nil {
		http.Error(writer, err.Error(), 500)
	} else if err := modifyNetworkBetweenNodes(watchdog.Id(otherId), *id, operation); err != nil {
		http.Error(writer, err.Error(), 500)
	} else {
		writer.WriteHeader(200)
	}
}

func modifyNetworkBetweenNodes(id watchdog.Id, other watchdog.Id, operation string) error {
	addr, err := cluster.HttpAddressFor(id)

	if err != nil {
		return fmt.Errorf("id %d does not exist in cluster", id)
	}

	resp, err := httpClient().Get(addr + "/" + operation + "?id=" + strconv.Itoa(int(other)))

	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		data, _ := ioutil.ReadAll(resp.Body)

		return fmt.Errorf("request failed: %d - %s", resp.StatusCode, data)
	}

	return nil
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
