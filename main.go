package main

import (
	GoHclGen "Terrapin/gohcl-gen"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud"
	"log"
)

var logger = hclog.L()

func PrettyPrint(a ...interface{}) {
	for _, value := range a {
		b, _ := json.MarshalIndent(value, " ", "    ")
		fmt.Println(string(b))
	}
}

func TerraformExec() {
	//tmpDir, err := ioutil.TempDir("", "tfinstall")
	//if err != nil {
	//	panic(err)
	//}
	//defer os.RemoveAll(tmpDir)
	//execPath, err := tfinstall.Find(context.Background(), tfinstall.LatestVersion(tmpDir, false))
	//if err != nil {
	//	panic(err)
	//}
	tcProvider := tencentcloud.Provider()

	if provider, ok := (tcProvider).(*schema.Provider); ok {
		for k, v := range provider.ResourcesMap["tencentcloud_instance"].Schema {
			if v.Optional || v.Required {
				logger.Info(
					"tencentcloud_instance schema", "key", k, "type", v.Type, "elem", v.Elem, "Optional", v.Optional,
					"Required", v.Required,
				)
			}

		}
	} else {
		panic(fmt.Errorf("wrong provider"))
	}

	//PrettyPrint(tcProvider.Resources())

	execPath, err := tfinstall.Find(context.Background(), tfinstall.LookPath())

	if true {
		return
	}
	if err != nil {
		panic(err)
	}

	workingDir := "/Users/ezonghu/Downloads/tmp/terraform/qcloud"
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		panic(err)
	}

	err = tf.Init(context.Background(), tfexec.Upgrade(false), tfexec.LockTimeout("60s"), tfexec.VerifyPlugins(false))
	if err != nil {
		panic(err)
	}

	err = tf.Apply(context.Background())
	if err != nil {
		panic(err)
	}

	state, err := tf.Show(context.Background())
	if err != nil {
		panic(err)
	}

	PrettyPrint(state.FormatVersion, state.Values) // "0.1"

	err = tf.Destroy(context.Background())
	if err != nil {
		panic(err)
	}

	state, err = tf.Show(context.Background())
	if err != nil {
		panic(err)
	}

	PrettyPrint(state.FormatVersion, state.Values) // "0.1"
}

func ExampleHclEncodeAndDecode() {
	type DataDisks struct {
		DataDiskSize int32   `hcl:"data_disk_size"`
		DataDiskType string  `hcl:"data_disk_type"`
		DataDiskId   *string `hcl:"data_disk_id"`
	}

	type ResourceTcInstance struct {
		Type      string      `hcl:",label"`
		Name      string      `hcl:",label"`
		ImageId   string      `hcl:"image_id"`
		DataDisks []DataDisks `hcl:"data_disks,block"`
	}

	type App struct {
		//Name             string            `hcl:"name"`
		//Desc             string            `hcl:"description"`
		Resources []ResourceTcInstance `hcl:"resource,block"`
	}

	disk_demo_id := "disk-1243323"
	app := App{
		//Name: "awesome-app",
		//Desc: "Such an awesome application",
		//Constraints: &Constraints{
		//	OS:   "linux",
		//	Arch: "amd64",
		//},
		Resources: []ResourceTcInstance{
			{
				Name:    "web",
				Type:    "http",
				ImageId: "img-123",
				DataDisks: []DataDisks{
					{DataDiskSize: 10, DataDiskType: "CLOUD_PREMIUM"},
					{DataDiskSize: 20, DataDiskType: "CLOUD_PREMIUM", DataDiskId: &disk_demo_id},
				},
			},
			{
				Name:    "work",
				Type:    "grpc",
				ImageId: "img-222",
			},
		},
	}

	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(&app, f.Body())
	fmt.Printf("%s", f.Bytes())
	var config App
	err := hclsimple.Decode("example.hcl", f.Bytes(), nil, &config)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	fmt.Printf("Configuration is %v\n", config)
	// Output:
	// name        = "awesome-app"
	// description = "Such an awesome application"
	//
	// constraints {
	//   os   = "linux"
	//   arch = "amd64"
	// }
	//
	// service "web" {
	//   executable = ["./web", "--listen=:8080"]
	// }
	// service "worker" {
	//   executable = ["./worker"]
	// }
}

func main() {
	//TerraformExec()
	ExampleHclEncodeAndDecode()
	GoHclGen.TfReader()
}
