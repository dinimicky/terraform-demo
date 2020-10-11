package GoHclGen

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud"
	"go/build"
	"go/format"
	"log"
	"os"
	"text/template"
)

type Status int

//go:generate go run "github.com/dinimicky/myenumstr" -type Status,Color
const (
	Offline Status = iota
	Online
	Disable
	Added
	Deleted
)

type Color int

const (
	Write Color = iota
	Red
	Blue
)

var (
	pkgInfo *build.Package
	err     error
	logger  = hclog.L()
)

type HclSchema interface {
}
type hclSchema struct {
	TypeName  string
	ValueType schema.ValueType
	GoType    string
	Optional  bool
	Required  bool
	Elem      interface{}
}

type HclResource interface {
	GoString() []byte
}
type hclResource struct {
	ResourceName string
	LabelNames   []string
	HclTag       string
	HclSchemas   []*hclSchema
}

/*
type DataDisks struct {
	DataDiskSize int32   `hcl:"data_disk_size"` //Required
	DataDiskType string  `hcl:"data_disk_type"` //Required
	DataDiskId   *string `hcl:"data_disk_id"` //Optional
}
type ${provider}Resource${resType} struct {
		Type      string      `hcl:"type,label"`
		Name      string      `hcl:"name,label"`
		ImageId   string      `hcl:"image_id"`
		DataDisks []DataDisks `hcl:"data_disks,block"`
}

Root Structure
type ${provider}Resources struct {
		${provider}Resource${resType} `hcl:"resource,block"`
}

*/
func NewHclSchema(typeName string, sa *schema.Schema) HclSchema {
	return &hclSchema{
		TypeName:  typeName,
		ValueType: sa.Type,
		Optional:  sa.Optional,
		Required:  sa.Required,
		Elem:      sa.Elem,
	}
}

func NewHclResource(resName string, res *schema.Resource, label ...string) HclResource {
	saList := make([]*hclSchema, len(res.Schema))
	i := 0
	for k, v := range res.Schema {
		hs := NewHclSchema(k, v)
		if ptrHs, ok := hs.(*hclSchema); ok {
			saList[i] = ptrHs
		}
		i++
	}
	return &hclResource{
		ResourceName: resName,
		LabelNames:   label,
		HclTag:       "`hcl:\",label\"`",
		HclSchemas:   saList,
	}
}

func (hr *hclResource) GoString() []byte {
	const strTmp = `type {{.ResourceName}} struct {
{{$tag:=.HclTag}}
{{range  .LabelNames}}
{{.}} string {{$tag}}
{{end}}
}`
	//利用模板库，生成代码文件
	t, err := template.New("").Parse(strTmp)
	if err != nil {
		log.Fatal(err)
	}
	buff := bytes.NewBufferString("")
	err = t.Execute(buff, *hr)
	if err != nil {
		log.Fatal(err)
	}
	//格式化
	src, err := format.Source(buff.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	return src
}
func GetElemType(elem interface{}) string {
	switch elem.(type) {
	case *schema.Resource:
		return "resource"
	case *schema.Schema:
		return "schema"
	case nil:
		return ""
	default:
		panic(fmt.Errorf("invalid elem type %v", elem))
	}
}

func TfReader() {
	tcProvider := tencentcloud.Provider()

	if provider, ok := (tcProvider).(*schema.Provider); ok {
		resName := "tencentcloud_instance"
		ResourceTcInstance := provider.ResourcesMap[resName]
		coreSchema := schema.InternalMap(ResourceTcInstance.Schema).CoreConfigSchema()
		logger.Info(
			"code generate", "context", string(NewHclResource(resName, ResourceTcInstance, "Type", "Name").GoString()),
		)

		for k, v := range coreSchema.Attributes {
			if true || v.Optional || v.Required {
				logger.Info(
					"tencentcloud_instance attr", "key", k, "type", v.Type.GoString(), "Optional",
					v.Optional,
					"Required", v.Required,
				)
			}

		}

		for k, v := range coreSchema.BlockTypes {

			logger.Info("tencentcloud_instance block", "key", k, "type", v.BlockTypes, "value", v.Block)

		}
	} else {
		panic(fmt.Errorf("wrong provider"))
	}

}

func ReadPackage() {
	pkgInfo, err = build.ImportDir(".", 0)
	if err != nil {
		panic(err)
	}
	fmt.Println("%v", pkgInfo)
	pkgInfo, err = build.ImportDir(
		"/Users/ezonghu/go/pkg/mod/github.com/tencentcloudstack/terraform-provider-tencentcloud@v1.44.0/gendoc", 0,
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("%v", pkgInfo)

	GOMODCACHE := os.Getenv("GOMODCACHE")
	fmt.Println("%v", GOMODCACHE)

}
