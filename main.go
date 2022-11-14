package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"text/template"

	"golang.org/x/exp/slices"
)

type Shortener struct {
	Key  string
	Dest string
}

var shortenings []Shortener

func init() {

	// If the file doesn't exist, create it, or append to the file
	_, err := os.OpenFile("dbfile.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	// Do log if error failed open / create dbfile if not exist
	if err != nil {
		log.Fatal(err)
	}

	jsonFile, errRead := ioutil.ReadFile("./dbfile.json")

	//
	if errRead != nil {
		log.Fatal(err)
	}

	switch {
	case len(jsonFile) > 0:
		{
			if errDecod := json.Unmarshal([]byte(jsonFile), &shortenings); errDecod != nil {
				panic(errDecod)
			}
		}
	default:
		shortenings = []Shortener{}
	}
}

func saveDbfile() {
	jsonShortenings, errEncode := json.Marshal(shortenings)
	if errEncode != nil {
		panic(errEncode)
	}

	errWriteDbfile := ioutil.WriteFile("dbfile.json", jsonShortenings, 0644)
	if errWriteDbfile != nil {
		panic(errWriteDbfile)
	}
}

var validShortenerPath = regexp.MustCompile("^/([a-zA-Z0-9]+)$")
var validEditorPath = regexp.MustCompile("^/(add|edit|remove)/([a-zA-Z0-9]+)$")

func wrapperEditorHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url := validEditorPath.FindStringSubmatch(r.URL.Path)
		if url == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, url[2])
	}
}

func renderEditorTpl(w http.ResponseWriter, shortener Shortener) {
	html, err := template.ParseFiles("shortenerForm.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	html.Execute(w, shortener)
}

func getShortenerIndex(key string) (int, error) {
	indexShortener := slices.IndexFunc(shortenings, func(shortening Shortener) bool { return shortening.Key == key })
	if indexShortener == -1 {
		return 0, errors.New("invalid shortener key")
	}
	return indexShortener, nil
}

func getShortener(key string) (Shortener, error) {
	indexShortener, err := getShortenerIndex(key)
	if err != nil {
		return Shortener{}, errors.New("invalid shortener key")
	}
	return shortenings[indexShortener], nil
}

func shorteningHandler(w http.ResponseWriter, r *http.Request) {
	uri := validShortenerPath.FindStringSubmatch(r.URL.Path)
	if uri == nil {
		http.NotFound(w, r)
		return
	}

	shortener, err := getShortener(uri[1])
	if err != nil {
		http.Redirect(w, r, "/add/"+uri[1], http.StatusFound)
		return
	}

	http.Redirect(w, r, shortener.Dest, http.StatusMovedPermanently)
}

func addHandler(w http.ResponseWriter, r *http.Request, key string) {
	_, err := getShortener(key)
	if err == nil {
		http.Redirect(w, r, "/edit/"+key, http.StatusFound)
		return
	}

	shortener := Shortener{Key: key}
	renderEditorTpl(w, shortener)
}

func editHandler(w http.ResponseWriter, r *http.Request, key string) {
	shortener, err := getShortener(key)
	if err != nil {
		http.Redirect(w, r, "/add/"+key, http.StatusFound)
		return
	}

	renderEditorTpl(w, shortener)
}

func removeHandler(w http.ResponseWriter, r *http.Request, key string) {
	indexShortener, err := getShortenerIndex(key)
	if err != nil {
		panic(err)
	}

	shortenings = append(shortenings[:indexShortener], shortenings[indexShortener+1:]...)

	saveDbfile()

	http.Redirect(w, r, "/add/"+key, http.StatusFound)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	dest := r.FormValue("dest")

	indexShortener, err := getShortenerIndex(key)
	if err != nil {
		shortenings = append(shortenings, Shortener{Key: key, Dest: dest})
	} else {
		shortenings[indexShortener].Dest = dest
	}

	saveDbfile()

	http.Redirect(w, r, "/edit/"+key, http.StatusFound)
}

func helpHandler(w http.ResponseWriter, r *http.Request) {

	//
	fmt.Fprintf(w, "%-40s%s", "/[YOUR-SHORTENER-KEY]", "URI that your shortener works. \n")
	fmt.Fprintf(w, "%-40s%s", "", "If your shortener exist, you will redirected to destination location. \n")
	fmt.Fprintf(w, "%-40s%s", "", "And if not exist, you will be redirected to add shortener.\n\n")

	//
	fmt.Fprintf(w, "%-40s%s", "/add/[YOUR-SHORTENER-KEY]", "URI that you can add new shortener. \n")
	fmt.Fprintf(w, "%-40s%s", "", "If your shortener not exist, you can add new shortener. \n")
	fmt.Fprintf(w, "%-40s%s", "", "if exist, you will be redirected to edit shortener.\n\n")

	//
	fmt.Fprintf(w, "%-40s%s", "/edit/[YOUR-SHORTENER-KEY]", "URI that you can edit shortener. \n")
	fmt.Fprintf(w, "%-40s%s", "", "If your shortener exist, you can edit shortener. \n")
	fmt.Fprintf(w, "%-40s%s", "", "if not exist, you will be redirected to add shortener.\n\n")

	//
	fmt.Fprintf(w, "%-40s%s", "/remove/[YOUR-SHORTENER-KEY]", "URI that you can remove shortener. \n\n")

	//
	fmt.Fprintf(w, "%-40s%s", "/help", "Your current URI that show you URI information. \n\n")

}

func main() {

	http.HandleFunc("/", shorteningHandler)
	http.HandleFunc("/save", saveHandler)
	http.HandleFunc("/add/", wrapperEditorHandler(addHandler))
	http.HandleFunc("/edit/", wrapperEditorHandler(editHandler))
	http.HandleFunc("/remove/", wrapperEditorHandler(removeHandler))
	http.HandleFunc("/help", helpHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))

}
