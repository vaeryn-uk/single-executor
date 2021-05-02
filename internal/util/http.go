package util

import (
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func GetIntParam(name string, values url.Values) (int, error) {
	input := values.Get(name)

	val, err := strconv.Atoi(input)

	if err != nil {
		return 0, err
	}

	return val, nil
}

func NoCache(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Unix(0, 0).Format(time.RFC1123))
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Pragma", "no-cache")
}

func ResponseWithJson(w http.ResponseWriter, data []byte) error {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")

	_, err := w.Write(data)

	return err
}
