package api
import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"fmt"
	"time"
	"net/http"
	"net/url"
	"encoding/json"
	"lineblocs.com/processor/types"
)

type APIResponse struct {
	Headers http.Header
	Body []byte
}
type VerifyCallerIdResponse struct {
	Valid bool `json:"valid"`
}

type CallerIdResponse struct {
	CallerId string `json:"caller_id"`
}
type DomainResponse struct {
	Id int `json:"id"`
	WorkspaceId int `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
}
type FlowResponse struct {
	FlowId int `json:"flow_id"`
	FlowJson string `json:"flow_json"`
}

type SubFlow struct {
	Vars *types.FlowVars
	FlowId int `json:"flow_id"`
}
type ConfParams struct {
	Name string `json:"name"`
	WorkspaceId int `json:"workspace_id"`	
}


type CallResponse struct {
  From string `json:"from"`
  To string `json:"to"`
  Status string `json:"status"`
  Direction string `json:"direction"`
  Duration string `json:"duration"`
  UserId int `json:"user_id"`
  WorkspaceId int  `json:"workspace_id"`
  APIId string `json:"api_id"`
  SourceIp string `json:"source_ip"`
  ChannelId string `json:"channel_id"`
  StartedAt string `json:"started_at"`
  CreatedAt string `json:"created_at"`
  UpdatedAt string `json:"updated_at"`
  PlanSnapshot string `json:"plan_snapshot"`
}
type ConferenceResponse struct {
	Id string `json:"id"`
}

type SettingsResponse struct {
  	AwsAccessKeyId string `json:"aws_access_key_id"`
	AwsSecretAccessKey string `json:"aws_secret_access_key"`
	AwsRegion string `json:"aws_region"`
	StripePubKey string `json:"stripe_pub_key"`
	StripePrivateKey string `json:"stripe_private_key"`
	StripeTestPubKey string `json:"stripe_test_pub_key"`
	StripeTestPrivateKey string `json:"stripe_test_private_key"`
	StripeMode string `json:"stripe_mode"`
	SmtpHost string `json:"smtp_host"`
	SmtpPort string `json:"smtp_port"`
	SmtpUser string `json:"smtp_user"`
	SmtpPassword string `json:"smtp_password"`
	SmtpTls string `json:"smtp_tls"`
	GoogleServiceAccountJson string `json:"google_service_account_json"`
}


var baseUrl string = "https://internals." + os.Getenv("DEPLOYMENT_DOMAIN")

func SendHttpRequest(path string, payload []byte) (*APIResponse, error) {
    url := baseUrl + path
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
    url := baseUrl + path
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
    fullUrl := baseUrl + path + "?"

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
	var data VerifyCallerIdResponse
 	err = json.Unmarshal( []byte(res), &data  )
	if err != nil {
		return false, err
	}
	return data.Valid, nil
}

func GetUserByTrunkSourceIp ( trunkSourceIp string ) (*DomainResponse, error) {
	params := make( map[string]string )
	fmt.Println("looking up trunk source ip: " + trunkSourceIp)
	params["source_ip"] = trunkSourceIp
	res, err := SendGetRequest("/user/getUserByTrunkSourceIp", params)
	if err != nil {
		return nil, err
	}
	var data DomainResponse
 	err = json.Unmarshal( []byte(res), &data  )
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func GetUserByDID ( did string ) (*DomainResponse, error) {
	params := make( map[string]string )
	fmt.Println("looking up user for did: " + did)
	params["did"] = did
	res, err := SendGetRequest("/user/getUserByDID", params)
	if err != nil {
		return nil, err
	}
	var data DomainResponse
 	err = json.Unmarshal( []byte(res), &data  )
	if err != nil {
		return nil, err
	}
	return &data, nil
}
func GetUserByDomain( domain string ) (*DomainResponse, error) {
	params := make( map[string]string )
	fmt.Println("looking up domain: " + domain)
	params["domain"] = domain
	res, err := SendGetRequest("/user/getUserByDomain", params)
	if err != nil {
		return nil, err
	}
	var data DomainResponse
 	err = json.Unmarshal( []byte(res), &data  )
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func UpdateCall( call *types.Call, status string ) (error) {
	call.Ended = time.Now()
	params := types.StatusParams{
		CallId: call.CallId,
		//Ip: utils.GetPublicIp(),
		Ip: "",
		Status: status  }
	body, err := json.Marshal( params )

	if err != nil {
		return err
	}

	_, err = SendHttpRequest( "/call/updateCall", body)
	if err != nil {
		return err
	}
	return nil
}

func GetCallerId( domain string, extension string ) (*CallerIdResponse, error) {
	params := make( map[string]string )
	fmt.Println("looking up caller id for: " + extension)
	params["domain"] = domain
	params["extension"] = extension
	res, err := SendGetRequest("/user/getCallerIdToUse", params)
	if err != nil {
		return nil, err
	}

	var data CallerIdResponse
 	err = json.Unmarshal( []byte(res), &data  )
	if err != nil {
		return nil, err
	}

	return &data, nil
}


func GetExtensionFlowInfo(workspace string, extension string) (*SubFlow, error) {
	params := make( map[string]string )
	fmt.Println("looking up caller id for: " + extension)
	params["workspace"] = workspace
	params["extension"] = extension
	res, err := SendGetRequest("/user/getExtensionFlowInfo", params)
	if err != nil {
		return nil, err
	}


	var subFlow SubFlow
	var data FlowResponse
	var flowJson types.FlowVars
	err = json.Unmarshal( []byte(res), &data )
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}

	subFlow = SubFlow{ FlowId: data.FlowId }

	err = json.Unmarshal( []byte(data.FlowJson), &flowJson )
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}

	subFlow.Vars = &flowJson

	return &subFlow, nil
}

func GetFlowInfo(workspace string, flowId string) (*SubFlow, error) {
	params := make( map[string]string )
	fmt.Println("looking up flow for: " + flowId)
	params["workspace"] = workspace
	params["flow_id"] = flowId
	res, err := SendGetRequest("/user/getFlowInfo", params)
	if err != nil {
		return nil, err
	}


	var subFlow SubFlow
	var data FlowResponse
	var flowJson types.FlowVars
	err = json.Unmarshal( []byte(res), &data )
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}

	subFlow = SubFlow{ FlowId: data.FlowId }

	err = json.Unmarshal( []byte(data.FlowJson), &flowJson )
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}

	subFlow.Vars = &flowJson

	return &subFlow, nil
}
func FetchCall(callId string) (*CallResponse, error) {
	params := make( map[string]string )
	fmt.Println("looking up call id: " + callId)
	params["id"] = callId
	res, err := SendGetRequest("/call/fetchCall", params)
	if err != nil {
		return nil, err
	}


	var data CallResponse
	err = json.Unmarshal( []byte(res), &data )
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}

	return &data, nil
}
func CreateConference(workspaceId int, name string) (*ConferenceResponse, error) {
	fmt.Println("creating conference...")
	params := ConfParams{
		Name: name,
		WorkspaceId: workspaceId }
	body, err := json.Marshal( params )

	if err != nil {
		return nil, err
	}

	resp, err := SendHttpRequest( "/conference/createConference", body)
	if err != nil {
		return nil, err
	}

	id := resp.Headers.Get("x-conference-id")
	return &ConferenceResponse{Id: id}, nil
}

func GetSettings() (*SettingsResponse, error) {
	fmt.Println("getting settings")

	params := make(map[string]string)
	res, err := SendGetRequest("/user/getSettings",params)
	if err != nil {
		return nil, err
	}


	var data SettingsResponse
	err = json.Unmarshal( []byte(res), &data )
	if err != nil {
		fmt.Println("get settings err " + err.Error())
		return nil, err
	}

	return &data, nil
}