package gsb_test

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	gsb "github.com/ideatocode/go-simpleapi-builder"
	"github.com/stretchr/testify/assert"
)

var addr string = "127.0.0.1:9999"
var message string = "Ok!"
var url string = "http://" + addr + "/testing"
var c *gsb.Controller

func runServer(ac gsb.AuthCallback) {

	c = gsb.NewController()

	c.AddHandler("/testing", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Write([]byte(message))
			return
		}
		b, _ := ioutil.ReadAll(r.Body)
		w.Write(b)
	}, "GET", "POST", "PUT")

	if ac != nil {
		c.AuthCallback = ac
	}

	go func() {
		c.Run(addr)
	}()
	time.Sleep(30 * time.Millisecond)
}
func response(resp *http.Response) string {

	defer resp.Body.Close()

	buf := make([]byte, len(message))

	io.ReadFull(resp.Body, buf)

	return string(buf)
}
func TestAddHandler(t *testing.T) {
	runServer(nil)
	defer c.Stop()
	resp, err := http.Get(url)
	if ok := assert.Nil(t, err, "Error should be nil"); ok != true {
		return
	}

	if ok := assert.Equal(t, 200, resp.StatusCode, "Code should be 200"); ok != true {
		return
	}

	result := response(resp)

	if ok := assert.Equal(t, message, result, "The response should be: "+message); ok != true {
		return
	}
}

func TestMethods(t *testing.T) {
	runServer(nil)
	defer c.Stop()

	var ok bool

	message = "test=testing&test2=testing2"

	reader := strings.NewReader(message)

	resp, err := http.Post(url, "application/x-www-form-urlencoded", reader)
	if ok = assert.Nil(t, err, "Error should be nil"); !ok {
		return
	}
	if ok = assert.Equal(t, 200, resp.StatusCode, "Status should be 200"); !ok {
		return
	}

	result := response(resp)
	if ok = assert.Equal(t, message, result, "The response should be: "+message); !ok {
		return
	}

	// re-init reader
	reader = strings.NewReader(message)

	// init request
	req, err := http.NewRequest(http.MethodPut, url, reader)
	if ok = assert.Nil(t, err, "Error should be nil"); !ok {
		return
	}

	// perform the put request
	resp, err = http.DefaultClient.Do(req)
	if ok = assert.Nil(t, err, "Error should be nil"); !ok {
		return
	}

	if ok = assert.Equal(t, 200, resp.StatusCode, "Status should be 200"); !ok {
		return
	}

	// check the response
	result = response(resp)
	if ok = assert.Equal(t, message, result, "The response should be: "+message); !ok {
		return
	}

}

func TestAuthGoodToken(t *testing.T) {
	runServer(func(token string, req *http.Request) (payload interface{}, err error) {
		if token == "goodtoken" {
			return "1", nil
		}
		return "", errors.New("Token not found")
	})
	defer c.Stop()

	var ok bool

	// init request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if ok = assert.Nil(t, err, "Error should be nil"); !ok {
		return
	}

	// Test good token
	req.Header.Add("Authorization", "Bearer goodtoken")
	// perform the put request
	resp, err := http.DefaultClient.Do(req)
	if ok = assert.Nil(t, err, "Error should be nil"); !ok {
		return
	}

	if ok = assert.Equal(t, 200, resp.StatusCode, "Status should be 200"); !ok {
		return
	}

	// check the response
	result := response(resp)
	if ok = assert.Equal(t, message, result, "The response should be: "+message); !ok {
		return
	}
}

func TestAuthBadToken(t *testing.T) {
	errorMessage := "Token not found"
	runServer(func(token string, req *http.Request) (payload interface{}, err error) {
		if token == "goodtoken" {
			return "1", nil
		}
		return "", errors.New(errorMessage)
	})
	defer c.Stop()

	var ok bool

	// init request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if ok = assert.Nil(t, err, "Error should be nil"); !ok {
		return
	}

	// Test good token
	req.Header.Add("Authorization", "Bearer badtoken")
	// perform the put request
	resp, err := http.DefaultClient.Do(req)
	if ok = assert.Nil(t, err, "Error should be nil"); !ok {
		return
	}

	if ok = assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Status should be 401"); !ok {
		return
	}

	// check the response
	result := resp.Header.Get("X-Error")
	if ok = assert.Equal(t, errorMessage, result, "Error should be: "+errorMessage); !ok {
		return
	}
}
