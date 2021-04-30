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
