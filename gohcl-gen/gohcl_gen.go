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

type Hcl interface {
	GoString() string
	GoType() string
	HclTag() string
}

type hclResource struct {
	ResourceName string
	LabelNames   []string
	HclLabelTag  string
	HclSchemas   []Hcl
}

func (hr *hclResource) HclTag() string {
	return fmt.Sprintf("`hcl:\"%v,block\"`", hr.ResourceName)
}

type hclSchema struct {
	TypeName  string
	ValueType schema.ValueType
	Optional  bool
	Required  bool
	Elem      Hcl
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

func NewHclSchema(typeName string, sa *schema.Schema) Hcl {
	hs := &hclSchema{
		TypeName:  typeName,
		ValueType: sa.Type,
		Optional:  sa.Optional,
		Required:  sa.Required,
	}
	switch sa.Elem.(type) {
	case *schema.Resource:
		hs.Elem = NewHclResource(typeName, sa.Elem.(*schema.Resource))
	case *schema.Schema:
		hs.Elem = NewHclSchema(typeName, sa.Elem.(*schema.Schema))
	case nil:
	default:
		panic(fmt.Errorf("Unsupported Elem type %T", sa.Elem))
	}

	return hs
}

func NewHclResource(resName string, res *schema.Resource, label ...string) Hcl {
	saList := make([]Hcl, len(res.Schema))
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
		HclLabelTag:  "`hcl:\",label\"`",
		HclSchemas:   saList,
	}
}

func (hs *hclSchema) GoString() string {
	if hs.Optional || hs.Required {
		return fmt.Sprintf("%v %v %v", Case2Camel(hs.TypeName), hs.GoType(), hs.HclTag())
	}
	return ""
}

func (hs *hclSchema) GoType() string {
	switch hs.ValueType {
	case schema.TypeBool:
		if hs.Optional {
			return "*bool"
		}
		return "bool"
	case schema.TypeInt:
		if hs.Optional {
			return "*int"
		}
		return "int"
	case schema.TypeFloat:
		if hs.Optional {
			return "*float"
		}
		return "float"
	case schema.TypeString:
		if hs.Optional {
			return "*string"
		}
		return "string"
	case schema.TypeList, schema.TypeSet:
		return fmt.Sprintf("[]%v", hs.Elem.GoType())
	case schema.TypeMap:
		if hs.Elem == nil {
			return fmt.Sprintf("map[string]string")
		}
		return fmt.Sprintf("map[string]%v", hs.Elem.GoType())
	default:
		return ""
	}
}

func (hr *hclSchema) HclTag() string {
	if ehr, ok := hr.Elem.(*hclResource); hr.Elem != nil && ok {
		return ehr.HclTag()
	}
	return fmt.Sprintf("`hcl:\"%v\"`", hr.TypeName)

}
func (hr *hclResource) GoString() string {
	const strTmp = `type {{ Case2Camel .ResourceName}} struct {
{{$tag:=.HclLabelTag}}
{{range  .LabelNames}}{{.}} string {{$tag}} 
{{end}}
{{range .HclSchemas}} {{.GoString }}
{{end}}
}`
	return render(strTmp, hr)
}

func (hr *hclResource) GoType() string {
	return Case2Camel(hr.ResourceName)
}

func render(strTmp string, params interface{}) string {
	//利用模板库，生成代码文件
	t, err := template.New("").Funcs(template.FuncMap{
		"Case2Camel": Case2Camel,
	}).Parse(strTmp)
	if err != nil {
		log.Fatal(err)
	}
	buff := bytes.NewBufferString("")
	err = t.Execute(buff, params)
	if err != nil {
		log.Fatal(err)
	}
	//格式化
	src, err := format.Source(buff.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	return string(src)
}

func CollectHclResources(hcl Hcl) []Hcl {
	res := make([]Hcl, 0)
	if hr, ok := hcl.(*hclResource); ok {
		res = append(res, hr)
		for _, hs := range hr.HclSchemas {
			res = append(res, CollectHclResources(hs)...)
		}
	}
	if hs, ok := hcl.(*hclSchema); ok {
		res = append(res, CollectHclResources(hs.Elem)...)
	}
	return res
}

func RootGoString(resName string, hcls []Hcl) string {
	hclResource := &hclResource{
		ResourceName: resName,
		HclSchemas:   hcls,
	}
	const strTmp = `type {{ Case2Camel .ResourceName}} struct {
{{$tag:=.HclLabelTag}}
{{range  .LabelNames}}{{.}} string {{$tag}} 
{{end}}
{{range .HclSchemas}} {{.GoType }} {{.HclTag}}
{{end}}
}`
	return render(strTmp, hclResource)
}
func HclRW() {
	tcProvider := tencentcloud.Provider()
	tcResList := make([]Hcl, 0)
	if provider, ok := (tcProvider).(*schema.Provider); ok {
		for k, res := range provider.ResourcesMap {
			hclres := NewHclResource(k, res, "Type", "Name")
			tcResList = append(tcResList, hclres)
			resList := CollectHclResources(hclres)
			logger.Info("====================", "res", k)
			for _, h := range resList {
				logger.Info(
					"code generate", "context", h.GoString(),
				)
			}

		}
		logger.Info("====================", "res", "ROOT")
		logger.Info("code generate", "context", RootGoString("tencent_cloud_stack", tcResList))
	}

}

func TfReader() {
	tcProvider := tencentcloud.Provider()

	if provider, ok := (tcProvider).(*schema.Provider); ok {
		resName := "tencentcloud_instance"
		ResourceTcInstance := provider.ResourcesMap[resName]

		coreSchema := schema.InternalMap(ResourceTcInstance.Schema).CoreConfigSchema()

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
