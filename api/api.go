package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/rampredd/url-shortener/storage"
)

func ShortenUrl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Body == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	type RequestType struct {
		Destination string `json:"destination"`
	}

	var req RequestType
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	uri, err := url.ParseRequestURI(req.Destination)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid URL"))
		return
	}

	//Save in DB
	shortUrl, err := storage.Save(uri.String())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error in saving in DB " + err.Error()))
		return
	}

	response := fmt.Sprintf("Generated link: %s", shortUrl)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(response))
}

func Metrics(w http.ResponseWriter, r *http.Request) {
	response, err := storage.LoadInfo()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	}

	res := ""

	for _, v := range response {
		res += fmt.Sprintf("%s:%d\n", v.Url, v.Score)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(res))
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["shortlink"]

	uri, err := storage.Load(code)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	http.Redirect(w, r, uri, http.StatusMovedPermanently)
}
