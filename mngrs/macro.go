package mngrs
import (
	//"github.com/CyCoreSystems/ari/v5"
	//clientcmd "k8s.io/client-go/1.5/tools/clientcmd"
    //"k8s.io/client-go/kubernetes"
	"context"
	"strings"
	"strconv"
	"path/filepath"
	"encoding/json"
	"github.com/google/uuid"
	    b64 "encoding/base64"
	    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//batchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	bv1 "k8s.io/api/batch/v1"
	    v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
		"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"fmt"
	"errors"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/types"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"lineblocs.com/processor/router"
)

func (*man MacroManager) startGRPCAndRunMacro(macro *WorkspaceMacro) {
	ctx := man.ManagerContext
	log := ctx.Log
	user := ctx.Flow.User
	channel := ctx.Channel
	flow := ctx.Flow
	cell := ctx.Cell
	var conn *grpc.ClientConn

	// get the service URL for the user
	// <service-name>.<namespace>.svc.cluster.local:<service-port>
	name := "voip-users"
	port := "10000"
	svcUri := fmt.Sprintf("%s.%s.svc.cluster.local:%s", user.WorkspaceName, name, port)
	conn, err := grpc.Dial(svcUri, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := router.NewLineblocsWorspaceSvcClient(conn)
	params := make(map[string]string)
	params["channel_id"] = channel.Channel.ID
	params["flow_id"] = strconv.Itoa( flow.FlowId )
	params["cell_id"] = cell.Cell.Id
	params["cell_name"] = cell.Cell.Name
	ctx := router.EventContext{
		Name: macro.Title,
		Event: params }
	response, err := c.CallMacro(context.Background(), &ctx)
	if err != nil {
		log.Fatalf("Error when calling CallMacro: %s", err)
		return
	}
	if response.Error {
		log.Fatalf("macro resulted in error: %s", response.Msg)
		return
	}
	log.Printf("Response from server: %s", response.Result)

}

type MacroManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewMacroManager(mngrCtx *types.Context, flow *types.Flow) (*MacroManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := MacroManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}

func createK8SConfig() (*rest.Config, error) {
	var kubeconfig string
	home := homedir.HomeDir()
	if home == "" {
		return nil, errors.New("cannot get HOME dir")
	}
	kubeconfig = filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		//panic(err.Error())
		return nil, err
	}
	return config, nil
}
func (man *MacroManager) initializeK8sAndExecute(b64code string, params string) (error) {
	ctx := man.ManagerContext
	log := ctx.Log


	log.Debug("Starting K8s job...")
	log.Debug("params: " + params)
	// creates the in-cluster config
	//config, err := rest.InClusterConfig()
	config, err := createK8SConfig()
	if err != nil {
		//panic(err.Error())
		return err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		//panic(err.Error())
		return err
	}
	//ctx := context.Background()
	// get pods in all the namespaces by omitting namespace
	// Or specify namespace to get pods in particular namespace

	uniq, err := uuid.NewUUID()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	jobName := "lineblocs-runner-" + uniq.String()
	//image := "lineblocs/runner"
	image := "754569496111.dkr.ecr.ca-central-1.amazonaws.com/lineblocs-k8s-runner:latest"
	cmd := "node /var/app/index.js"
	err = man.launchK8sJob(clientset, &jobName, &image, &cmd ,&b64code, &params)
	if err != nil {
		//panic(err.Error())
		return err
	}
	return nil
}

func (man *MacroManager) launchK8sJob(clientset *kubernetes.Clientset, jobName *string, image *string, cmd *string, script *string, params *string) (error) {
	ctx := man.ManagerContext
	log := ctx.Log
	user := ctx.Flow.User
	channel := ctx.Channel

	token := user.Token
	secret := user.Secret
	workspace := strconv.Itoa( user.Workspace.Id )
	userId := strconv.Itoa( user.Id )
	domain := user.Workspace.Domain


	log.Debug("Running..")
    jobs := clientset.BatchV1().Jobs("default")
    var backOffLimit int32 = 0
	/*
LINEBLOCS_TOKEN=
LINEBLOCS_SECRET=
LINEBLOCS_WORKSPACE_ID=1
LINEBLOCS_USER_ID=1
LINEBLOCS_DOMAIN=workspace.lineblocs.com
CHANNEL_ID=1
PARAMS={"test": 123}
*/
    //jobSpec := bv1.Job("t", "123")
	jobSpec := bv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      *jobName,
            Namespace: "default",
        },
        Spec: bv1.JobSpec{
            Template: v1.PodTemplateSpec{
                Spec: v1.PodSpec{
                    Containers: []v1.Container{
                        {
                            Name:    *jobName,
                            Image:   *image,
                            Command: strings.Split(*cmd, " "),
                    		ImagePullPolicy: v1.PullPolicy("IfNotPresent"),
                    		//ImagePullPolicy: v1.PullPolicy("Always"),
							Env: []v1.EnvVar{
								{
									Name: "LINEBLOCS_TOKEN",
									Value: token,
								},
								{
									Name: "LINEBLOCS_SECRET",
									Value: secret,
								},
								{
									Name: "LINEBLOCS_WORKSPACE_ID",
									Value: workspace,
								},
								{
									Name: "LINEBLOCS_USER_ID",
									Value: userId,
								},
								{
									Name: "LINEBLOCS_DOMAIN",
									Value: domain,
								},
								{
									Name: "CHANNEL_ID",
									Value: channel.Channel.ID(),
								},
								{
									Name: "PARAMS",
									Value: *params,
								},
								{
									Name: "SCRIPT_B64",
									Value: *script,
								},
							},
                        },
                    },
                    RestartPolicy: v1.RestartPolicyNever,
					ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: "lineblocs-regcred",
						},
					},
                },
            },
            BackoffLimit: &backOffLimit,
        },
    }

    _, err := jobs.Create(context.TODO(), &jobSpec, metav1.CreateOptions{})
    if err != nil {
        fmt.Println("Failed to create K8s job. error: " + err.Error())
		return err
    }

    //print job details
    fmt.Println("Created K8s job successfully")
	return nil
}

func (man *MacroManager) executeMacro() {
	log := man.ManagerContext.Log
	cell := man.ManagerContext.Cell
	channel := man.ManagerContext.Channel
	flow := man.ManagerContext.Flow
	model := cell.Model
	log.Debug("running macro script..");

	function := model.Data["function"].ValueStr
	params := model.Data["params"].ValueObj

	completed, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Completed")
	errorLink, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Error")

	var foundFn *types.WorkspaceMacro



	// find the code
	for _, macro := range flow.WorkspaceFns {
		if macro.Title ==  function {
			foundFn = macro
		}
	}
	paramsEncoded, err := json.Marshal(params)
	if err != nil {
		log.Error("error occured: " + err.Error());
		resp := types.ManagerResponse{
			Channel: channel,
			Link: errorLink }
		man.ManagerContext.RecvChannel <- &resp
		return
	}

	err = man.startGRPCAndRunMacro(foundFn)

	if foundFn == nil {
		log.Debug("could not find macro function...")
		resp := types.ManagerResponse{
			Channel: channel,
			Link: errorLink }
		man.ManagerContext.RecvChannel <- &resp
		return
	}
	//sEnc := b64.StdEncoding.EncodeToString([]byte(foundFn.CompiledCode))
	//err := man.initializeK8sAndExecute(sEnc)

	if err != nil {
		log.Error("error occured: " + err.Error());
		resp := types.ManagerResponse{
			Channel: channel,
			Link: errorLink }
		man.ManagerContext.RecvChannel <- &resp
		return
	}
	resp := types.ManagerResponse{
		Channel: channel,
		Link: completed }
	man.ManagerContext.RecvChannel <- &resp
}
func (man *MacroManager) StartProcessing() {
	go man.executeMacro();
}
