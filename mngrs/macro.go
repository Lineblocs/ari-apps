package mngrs
import (
	//"github.com/CyCoreSystems/ari/v5"
	//clientcmd "k8s.io/client-go/1.5/tools/clientcmd"
    //"k8s.io/client-go/kubernetes"
	"context"
	"strings"
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

func initializeK8sAndExecute() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		//ctx := context.Background()
		// get pods in all the namespaces by omitting namespace
		// Or specify namespace to get pods in particular namespace
		jobName := "lineblocs-runner"
		image := "lineblocs/runner"
		cmd := "-c BASE64"
		err = launchK8sJob(clientset, &jobName, &image, &cmd)
		if err != nil {
			panic(err.Error())
		}
	}
}
func launchK8sJob(clientset *kubernetes.Clientset, jobName *string, image *string, cmd *string) (error) {
    jobs := clientset.BatchV1().Jobs("default")
    var backOffLimit int32 = 0

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
                        },
                    },
                    RestartPolicy: v1.RestartPolicyNever,
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

	err := utils.RunScriptInContext(foundFn.CompiledCode)
	if err != nil {
		fmt.Println( err.Error ());

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
