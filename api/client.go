package api
import (
	"bytes"
	"errors"
	"io/ioutil"
	"fmt"
	"net/http"
	"net/url"
	"encoding/json"
)

type APIResponse struct {
	Headers http.Header
	Body []byte
}
type CallerIdResponse struct {
	Valid bool `json:"valid"`
}
func SendHttpRequest(path string, payload []byte) (*APIResponse, error) {
    url := "https://internals.lineblocs.com" + path
    fmt.Println("URL:>", url)

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
    req.Header.Set("X-Custom-Header", "myvalue")
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
		return nil, err
    }
    defer resp.Body.Close()

	var headers http.Header




    fmt.Println("response Status:", resp.Status)
    fmt.Println("response Headers:", resp.Header)

	headers = resp.Header
    body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyAsString := string(body)
    fmt.Println("response Body:", bodyAsString)
    fmt.Println("response Status:", resp.Status)

status := resp.StatusCode
	if !(status >= 200 && status <= 299) {
		return nil, errors.New("Status: " + resp.Status + " result: " + bodyAsString)
	}

	return &APIResponse{  
		Headers: headers,
		Body: body }, nil

}


func SendPutRequest(path string, payload []byte) (string, error) {
    url := "https://internals.lineblocs.com" + path
    fmt.Println("URL:>", url)

    req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
		return "", err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	bodyAsString := string(body)
    fmt.Println("response Body:", bodyAsString)
    fmt.Println("response Status:", resp.Status)
status := resp.StatusCode
	if !(status >= 200 && status <= 299) {
		return "", errors.New("Status: " + resp.Status + " result: " + bodyAsString)
	}
	return bodyAsString, nil

}

func SendGetRequest(path string, vals map[string] string) (string, error) {
    fullUrl := "https://internals.lineblocs.com" + path + "?"

	for k,v := range vals {
		fullUrl = fullUrl + (k + "=" + url.QueryEscape(v)) + "&"
	}
    fmt.Println("URL:>", fullUrl)

    req, err := http.NewRequest("GET", fullUrl, bytes.NewBuffer([]byte("")))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
		return "", err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	bodyAsString := string(body)

    fmt.Println("response Body:", bodyAsString)
    fmt.Println("response Status:", resp.Status)
	status := resp.StatusCode
	if !(status >= 200 && status <= 299) {
		return "", errors.New("Status: " + resp.Status + " result: " + bodyAsString)
	}
	return bodyAsString, nil
}

func VerifyCallerId( workspaceId string, callerId string) (bool, error) {
	params := make( map[string]string )
	fmt.Println("looking up workspace: " + workspaceId)
	fmt.Println("number: " + callerId)
	params["workspace_id"] = workspaceId
	params["number"] = callerId
	res, err := SendGetRequest("/user/verifyCaller", params)
	if err != nil {
		return false, err
	}
	var data CallerIdResponse
 	err = json.Unmarshal( []byte(res), &data  )
	if err != nil {
		return false, err
	}
	return data.Valid, nil
}
