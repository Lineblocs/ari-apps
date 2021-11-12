package utils
import (
	v8 "rogchap.com/v8go"
	"fmt"
)

var iso *v8.Isolate;
func createIso() (*v8.Isolate, error) {

	var err error
	if  iso == nil {
		iso, err = v8.NewIsolate()
		if  err != nil {
			return nil, err
		}
	}
	return iso, nil
}

func RunScriptInContext(script string) (error) {
	iso, err := createIso()
	if err != nil {
		return err
	}
	ctx, err := v8.NewContext(iso) // another context on the same VM
	if err != nil {
		return err
	}
	
	fmt.Printf("running script: %s\r\n", script)
	if _, err := ctx.RunScript(script, "main.js"); err != nil {
		// this will error as multiply is not defined in this context
		fmt.Println("error occured:" + err.Error());
		return err
	}

	fmt.Println("script completed..");
	return nil
}
