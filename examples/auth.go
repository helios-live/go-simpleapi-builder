package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	apicontroller "github.com/Alex-Eftimie/api-controller"
)

func authCallback(token string) (id string, err error) {
	if token == "goodtoken" {
		return "1", nil
	}
	return "", errors.New("Bad Token")
}

func main() {
	c := apicontroller.NewController()
	defer c.Stop()

	c.AuthCallback = authCallback
	c.AddHandler("/profile", func(w http.ResponseWriter, r *http.Request) {

		// Authed already
		id := w.Header().Get("X-ID")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, your id is: " + id))
	}, "GET")
	go func() {
		c.Run(":8080")
	}()

	// init request
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080/profile", nil)
	if err != nil {
		log.Fatalln(err)
	}

	// Test good token
	// req.Header.Add("Authorization", "Bearer badtoken")
	req.Header.Add("Authorization", "Bearer goodtoken")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode == 200 {
		log.Println("Status:", resp.StatusCode, "Response:", string(buf))
	} else {
		log.Println("Status:", resp.StatusCode, "Error:", resp.Header.Get("X-Error"))
	}
}
