package mngrs
import (
	//"github.com/CyCoreSystems/ari/v5"
	//clientcmd "k8s.io/client-go/1.5/tools/clientcmd"
    //"k8s.io/client-go/kubernetes"
	"context"
	"strings"
	"strconv"
	    b64 "encoding/base64"
	    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//batchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	bv1 "k8s.io/api/batch/v1"
	    v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"fmt"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/types"
)
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
func (man *MacroManager) initializeK8sAndExecute(b64code string) (error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
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
	jobName := "lineblocs-runner"
	//image := "lineblocs/runner"
	image := "754569496111.dkr.ecr.ca-central-1.amazonaws.com/lineblocs-k8s-runner:latest"
	cmd := "node /var/app/index.js"
	err = man.launchK8sJob(clientset, &jobName, &image, &cmd ,&b64code)
	if err != nil {
		//panic(err.Error())
		return err
	}
	return nil
}

func (man *MacroManager) launchK8sJob(clientset *kubernetes.Clientset, jobName *string, image *string, cmd *string, script *string) (error) {
	ctx := man.ManagerContext
	user := ctx.Flow.User
	channel := ctx.Channel

	token := user.Token
	secret := user.Secret
	workspace := strconv.Itoa( user.Workspace.Id )
	userId := strconv.Itoa( user.Id )
	domain := user.Workspace.Domain
	params := "{}"
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
									Value: params,
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
        fmt.Println("Failed to create K8s job.")
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

	completed, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Completed")
	errorLink, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Error")

	var foundFn *types.WorkspaceMacro
	// find the code
	for _, macro := range flow.WorkspaceFns {
		if macro.Title ==  function {
			foundFn = macro
		}
	}
	if foundFn == nil {
		log.Debug("could not find macro function...")
		resp := types.ManagerResponse{
			Channel: channel,
			Link: errorLink }
		man.ManagerContext.RecvChannel <- &resp
		return
	}
	sEnc := b64.StdEncoding.EncodeToString([]byte(foundFn.CompiledCode))
	err := man.initializeK8sAndExecute(sEnc)

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
