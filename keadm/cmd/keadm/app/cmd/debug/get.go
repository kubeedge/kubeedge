/*
Copyright 2020 The KubeEdge Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/klog/v2"
	api "k8s.io/kubernetes/pkg/apis/core"
	k8sprinters "k8s.io/kubernetes/pkg/printers"
	printersinternal "k8s.io/kubernetes/pkg/printers/internalversion"
	"k8s.io/kubernetes/pkg/printers/storage"

	"github.com/kubeedge/beehive/pkg/common/util"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	edgecoreCfg "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const (
	// DefaultErrorExitCode defines exit the code for failed action generally
	DefaultErrorExitCode = 1
	// ResourceTypeAll defines resource type all
	ResourceTypeAll = "all"
	// FormatTypeWIDE defines output format wide
	FormatTypeWIDE = "wide"
)

var (
	debugGetLong = `
Prints a table of the most important information about the specified resource from the local database of the edge node.`
	debugGetExample = `
# List all pod in namespace test
keadm debug get pod -n test
# List a single configmap  with specified NAME
keadm debug get configmap web -n default
# List the complete information of the configmap with the specified name in the yaml output format
keadm debug get configmap web -n default -o yaml
# List the complete information of all available resources of edge nodes using the specified format (default: yaml)
keadm debug get all -o yaml`

	// availableResources Convert flag to currently supports available Resource types in EdgeCore database.
	availableResources = map[string]string{
		"all":        ResourceTypeAll,
		"po":         model.ResourceTypePod,
		"pod":        model.ResourceTypePod,
		"pods":       model.ResourceTypePod,
		"no":         model.ResourceTypeNode,
		"node":       model.ResourceTypeNode,
		"nodes":      model.ResourceTypeNode,
		"svc":        constants.ResourceTypeService,
		"service":    constants.ResourceTypeService,
		"services":   constants.ResourceTypeService,
		"secret":     model.ResourceTypeSecret,
		"secrets":    model.ResourceTypeSecret,
		"cm":         model.ResourceTypeConfigmap,
		"configmap":  model.ResourceTypeConfigmap,
		"configmaps": model.ResourceTypeConfigmap,
		"ep":         constants.ResourceTypeEndpoints,
		"endpoint":   constants.ResourceTypeEndpoints,
		"endpoints":  constants.ResourceTypeEndpoints,
	}
)

// NewCmdDebugGet returns keadm debug get command.
func NewCmdDebugGet(out io.Writer, getOption *GetOptions) *cobra.Command {
	if getOption == nil {
		getOption = NewGetOptions()
	}

	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Display one or many resources",
		Long:    debugGetLong,
		Example: debugGetExample,
		Run: func(cmd *cobra.Command, args []string) {
			if err := getOption.Validate(args); err != nil {
				CheckErr(err, fatal)
			}
			if err := getOption.Run(args, out); err != nil {
				CheckErr(err, fatal)
			}
		},
	}
	addGetOtherFlags(cmd, getOption)

	return cmd
}

// fatal prints the message if set and then exits.
func fatal(msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}

		fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(code)
}

// CheckErr formats a given error as a string and calls the passed handleErr
// func with that string and an exit code.
func CheckErr(err error, handleErr func(string, int)) {
	switch err.(type) {
	case nil:
		return
	default:
		handleErr(err.Error(), DefaultErrorExitCode)
	}
}

// addGetOtherFlags
func addGetOtherFlags(cmd *cobra.Command, getOption *GetOptions) {
	cmd.Flags().StringVarP(&getOption.Namespace, "namespace", "n", getOption.Namespace, "List the requested object(s) in specified namespaces")
	cmd.Flags().StringVarP(getOption.PrintFlags.OutputFormat, "output", "o", *getOption.PrintFlags.OutputFormat, "Indicate the output format. Currently supports formats such as yaml|json|wide")
	cmd.Flags().StringVarP(&getOption.LabelSelector, "selector", "l", getOption.LabelSelector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	cmd.Flags().StringVarP(&getOption.DataPath, "edgedb-path", "p", getOption.DataPath, "Indicate the edge node database path, the default path is \"/var/lib/kubeedge/edgecore.db\"")
	cmd.Flags().BoolVarP(&getOption.AllNamespace, "all-namespaces", "A", getOption.AllNamespace, "List the requested object(s) across all namespaces")
}

// NewGetOptions returns a GetOptions with default EdgeCore database source.
func NewGetOptions() *GetOptions {
	opts := &GetOptions{
		Namespace:  "default",
		DataPath:   edgecoreCfg.DataBaseDataSource,
		PrintFlags: NewGetPrintFlags(),
	}

	return opts
}

// GetOptions contains the input to the get command.
type GetOptions struct {
	AllNamespace  bool
	Namespace     string
	LabelSelector string
	DataPath      string

	PrintFlags *PrintFlags
}

// Run performs the get operation.
func (g *GetOptions) Run(args []string, out io.Writer) error {
	resType := args[0]
	resNames := args[1:]
	results, err := g.queryDataFromDatabase(availableResources[resType], resNames)
	if err != nil {
		return err
	}

	if len(g.LabelSelector) > 0 {
		results, err = FilterSelector(results, g.LabelSelector)
		if err != nil {
			return err
		}
	}

	if g.AllNamespace {
		if err := g.PrintFlags.EnsureWithNamespace(); err != nil {
			return err
		}
	}

	printer, err := g.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}

	if len(results) == 0 {
		if _, err := fmt.Fprintf(out, "No resources found in %v namespace.\n", g.Namespace); err != nil {
			return err
		}
		return nil
	}
	if *g.PrintFlags.OutputFormat == "" || *g.PrintFlags.OutputFormat == FormatTypeWIDE {
		return HumanReadablePrint(results, printer, out)
	}

	return JSONYamlPrint(results, printer, out)
}

// IsAllowedFormat verification support format
func (g *GetOptions) IsAllowedFormat(f string) bool {
	allowedFormats := g.PrintFlags.AllowedFormats()
	for _, v := range allowedFormats {
		if f == v {
			return true
		}
	}

	return false
}

// Validate checks the set of flags provided by the user.
func (g *GetOptions) Validate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("You must specify the type of resource to get. ")
	}
	if !isAvailableResources(args[0]) {
		return fmt.Errorf("Unrecognized resource type: %v. ", args[0])
	}
	if len(g.DataPath) == 0 {
		fmt.Printf("Not specified the EdgeCore database path, use the default path: %v. ", g.DataPath)
	}
	if !isFileExist(g.DataPath) {
		return fmt.Errorf("EdgeCore database file %v not exist. ", g.DataPath)
	}

	if err := InitDB(edgecoreCfg.DataBaseDriverName, edgecoreCfg.DataBaseAliasName, g.DataPath); err != nil {
		return fmt.Errorf("Failed to initialize database: %v ", err)
	}
	if len(*g.PrintFlags.OutputFormat) > 0 {
		format := strings.ToLower(*g.PrintFlags.OutputFormat)
		g.PrintFlags.OutputFormat = &format
		if !g.IsAllowedFormat(*g.PrintFlags.OutputFormat) {
			return fmt.Errorf("Invalid output format: %v, currently supports formats such as yaml|json|wide. ", *g.PrintFlags.OutputFormat)
		}
	}
	if args[0] == ResourceTypeAll && len(args) >= 2 {
		return fmt.Errorf("You must specify only one resource. ")
	}

	return nil
}

func (g *GetOptions) queryDataFromDatabase(resType string, resNames []string) ([]dao.Meta, error) {
	var result []dao.Meta

	switch resType {
	case model.ResourceTypePod:
		pods, err := g.getPodsFromDatabase(g.Namespace, resNames)
		if err != nil {
			return nil, err
		}
		result = append(result, pods...)
	case model.ResourceTypeNode:
		node, err := g.getNodeFromDatabase(g.Namespace, resNames)
		if err != nil {
			return nil, err
		}
		result = append(result, node...)
	case model.ResourceTypeConfigmap, model.ResourceTypeSecret, constants.ResourceTypeEndpoints, constants.ResourceTypeService:
		value, err := g.getResourceFromDatabase(g.Namespace, resNames, resType)
		if err != nil {
			return nil, err
		}
		result = append(result, value...)
	case ResourceTypeAll:
		pods, err := g.getPodsFromDatabase(g.Namespace, resNames)
		if err != nil {
			return nil, err
		}
		result = append(result, pods...)

		resTypes := []string{model.ResourceTypeConfigmap, model.ResourceTypeSecret, constants.ResourceTypeEndpoints, constants.ResourceTypeService}
		for _, v := range resTypes {
			value, err := g.getResourceFromDatabase(g.Namespace, resNames, v)
			if err != nil {
				return nil, err
			}
			result = append(result, value...)
		}
	default:
		return nil, fmt.Errorf("Query resource type: %v in namespaces: %v failed. ", resType, g.Namespace)
	}

	return result, nil
}

func (g *GetOptions) getPodsFromDatabase(resNS string, resNames []string) ([]dao.Meta, error) {
	var results []dao.Meta
	podJSON := make(map[string]interface{})
	podStatusJSON := make(map[string]interface{})

	podRecords, err := dao.QueryAllMeta("type", model.ResourceTypePod)
	if err != nil {
		return nil, err
	}
	for _, v := range *podRecords {
		namespaceParsed, _, _, _ := util.ParseResourceEdge(v.Key, model.QueryOperation)
		if namespaceParsed != resNS && !g.AllNamespace {
			continue
		}
		if len(resNames) > 0 && !isExistName(resNames, v.Key) {
			continue
		}

		podKey := strings.Replace(v.Key, constants.ResourceSep+model.ResourceTypePod+constants.ResourceSep,
			constants.ResourceSep+model.ResourceTypePodStatus+constants.ResourceSep, 1)
		podStatusRecords, err := dao.QueryMeta("key", podKey)
		if err != nil {
			return nil, err
		}
		if len(*podStatusRecords) <= 0 {
			results = append(results, v)
			continue
		}
		if err := json.Unmarshal([]byte(v.Value), &podJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte((*podStatusRecords)[0]), &podStatusJSON); err != nil {
			return nil, err
		}
		podJSON["status"] = podStatusJSON["Status"]
		data, err := json.Marshal(podJSON)
		if err != nil {
			return nil, err
		}
		v.Value = string(data)
		results = append(results, v)
	}

	return results, nil
}

func (g *GetOptions) getNodeFromDatabase(resNS string, resNames []string) ([]dao.Meta, error) {
	var results []dao.Meta
	nodeJSON := make(map[string]interface{})
	nodeStatusJSON := make(map[string]interface{})

	nodeRecords, err := dao.QueryAllMeta("type", model.ResourceTypeNode)
	if err != nil {
		return nil, err
	}
	for _, v := range *nodeRecords {
		namespaceParsed, _, _, _ := util.ParseResourceEdge(v.Key, model.QueryOperation)
		if namespaceParsed != resNS && !g.AllNamespace {
			continue
		}
		if len(resNames) > 0 && !isExistName(resNames, v.Key) {
			continue
		}

		nodeKey := strings.Replace(v.Key, constants.ResourceSep+model.ResourceTypeNode+constants.ResourceSep,
			constants.ResourceSep+model.ResourceTypeNodeStatus+constants.ResourceSep, 1)
		nodeStatusRecords, err := dao.QueryMeta("key", nodeKey)
		if err != nil {
			return nil, err
		}
		if len(*nodeStatusRecords) <= 0 {
			results = append(results, v)
			continue
		}
		if err := json.Unmarshal([]byte(v.Value), &nodeJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte((*nodeStatusRecords)[0]), &nodeStatusJSON); err != nil {
			return nil, err
		}
		nodeJSON["status"] = nodeStatusJSON["Status"]
		data, err := json.Marshal(nodeJSON)
		if err != nil {
			return nil, err
		}
		v.Value = string(data)
		results = append(results, v)
	}

	return results, nil
}

func (g *GetOptions) getResourceFromDatabase(resNS string, resNames []string, resType string) ([]dao.Meta, error) {
	var results []dao.Meta

	resRecords, err := dao.QueryAllMeta("type", resType)
	if err != nil {
		return nil, err
	}
	for _, v := range *resRecords {
		namespaceParsed, _, _, _ := util.ParseResourceEdge(v.Key, model.QueryOperation)
		if namespaceParsed != resNS && !g.AllNamespace {
			continue
		}
		if len(resNames) > 0 && !isExistName(resNames, v.Key) {
			continue
		}
		results = append(results, v)
	}

	return results, nil
}

// FilterSelector filter resource by selector
func FilterSelector(data []dao.Meta, selector string) ([]dao.Meta, error) {
	var results []dao.Meta
	var jsonValue = make(map[string]interface{})

	selectors, err := SplitSelectorParameters(selector)
	if err != nil {
		return nil, err
	}
	for _, v := range data {
		err := json.Unmarshal([]byte(v.Value), &jsonValue)
		if err != nil {
			return nil, err
		}
		labels := jsonValue["metadata"].(map[string]interface{})["labels"]
		if labels == nil {
			results = append(results, v)
			continue
		}
		flag := true
		for _, v := range selectors {
			if !v.Exist {
				flag = flag && labels.(map[string]interface{})[v.Key] != v.Value
				continue
			}
			flag = flag && (labels.(map[string]interface{})[v.Key] == v.Value)
		}
		if flag {
			results = append(results, v)
		}
	}

	return results, nil
}

// IsAvailableResources verification support resource type
func isAvailableResources(rsT string) bool {
	_, ok := availableResources[rsT]
	return ok
}

// IsFileExist check file is exist
func isFileExist(path string) bool {
	_, err := os.Stat(path)

	return err == nil || os.IsExist(err)
}

// InitDB Init DB info
func InitDB(driverName, dbName, dataSource string) error {
	if err := orm.RegisterDriver(driverName, orm.DRSqlite); err != nil {
		return fmt.Errorf("Failed to register driver: %v ", err)
	}
	if err := orm.RegisterDataBase(
		dbName,
		driverName,
		dataSource); err != nil {
		return fmt.Errorf("Failed to register db: %v ", err)
	}
	orm.RegisterModel(new(dao.Meta))

	// create orm
	dbm.DBAccess = orm.NewOrm()
	if err := dbm.DBAccess.Using(dbName); err != nil {
		return fmt.Errorf("Using db access error %v ", err)
	}
	return nil
}

// IsExistName verify the filed in the resNames exists in the name
func isExistName(resNames []string, name string) bool {
	value := false
	for _, v := range resNames {
		if strings.Contains(name, v) {
			value = true
		}
	}

	return value
}

// Selector filter structure
type Selector struct {
	Key   string
	Value string
	Exist bool
}

// SplitSelectorParameters Split selector args (flag: -l)
func SplitSelectorParameters(args string) ([]Selector, error) {
	var results = make([]Selector, 0)
	var sel Selector
	labels := strings.Split(args, ",")
	for _, label := range labels {
		if strings.Contains(label, "==") {
			labs := strings.Split(label, "==")
			if len(labs) != 2 {
				return nil, fmt.Errorf("Arguments in selector form may not have more than one \"==\". ")
			}
			sel.Key = labs[0]
			sel.Value = labs[1]
			sel.Exist = true
			results = append(results, sel)
			continue
		}
		if strings.Contains(label, "!=") {
			labs := strings.Split(label, "!=")
			if len(labs) != 2 {
				return nil, fmt.Errorf("Arguments in selector form may not have more than one \"!=\". ")
			}
			sel.Key = labs[0]
			sel.Value = labs[1]
			sel.Exist = false
			results = append(results, sel)
			continue
		}
		if strings.Contains(label, "=") {
			labs := strings.Split(label, "=")
			if len(labs) != 2 {
				return nil, fmt.Errorf("Arguments in selector may not have more than one \"=\". ")
			}
			sel.Key = labs[0]
			sel.Value = labs[1]
			sel.Exist = true
			results = append(results, sel)
		}
	}
	return results, nil
}

func HumanReadablePrint(results []dao.Meta, printer printers.ResourcePrinter, out io.Writer) error {
	res, err := ParseMetaToAPIList(results)
	if err != nil {
		klog.Fatal(err)
	}
	for _, r := range res {
		table, err := ConvertDataToTable(r)
		if err != nil {
			klog.Fatal(err)
		}
		if err := printer.PrintObj(table, out); err != nil {
			klog.Fatal(err)
		}
		if _, err := fmt.Fprintln(out); err != nil {
			klog.Fatal(err)
		}
	}
	return nil
}

// xParseMetaToAPIList Convert the data to the corresponding list type according to the apiserver usage type
// Only use this type definition to get the table header processing handle,
// and automatically obtain the ColumnDefinitions of the table according to the type
// Only used by HumanReadablePrint.
func ParseMetaToAPIList(metas []dao.Meta) (res []runtime.Object, err error) {
	var (
		podList       api.PodList
		serviceList   api.ServiceList
		secretList    api.SecretList
		configMapList api.ConfigMapList
		endPointsList api.EndpointsList
		nodeList      api.NodeList
	)
	value := make(map[string]interface{})

	for _, v := range metas {
		if err := json.Unmarshal([]byte(v.Value), &value); err != nil {
			return nil, err
		}
		metadata, err := json.Marshal(value["metadata"])
		switch v.Type {
		case model.ResourceTypePod:
			var pod api.Pod

			if err = json.Unmarshal([]byte(v.Value), &pod); err != nil {
				return nil, err
			}
			if err = json.Unmarshal(metadata, &pod.ObjectMeta); err != nil {
				return nil, err
			}
			podList.Items = append(podList.Items, pod)
		case constants.ResourceTypeService:
			var svc api.Service

			if err != nil {
				return nil, err
			}
			if err = json.Unmarshal([]byte(v.Value), &svc); err != nil {
				return nil, err
			}
			if err = json.Unmarshal(metadata, &svc.ObjectMeta); err != nil {
				return nil, err
			}
			serviceList.Items = append(serviceList.Items, svc)
		case model.ResourceTypeSecret:
			var secret api.Secret
			if err = json.Unmarshal([]byte(v.Value), &secret); err != nil {
				return nil, err
			}
			if err = json.Unmarshal(metadata, &secret.ObjectMeta); err != nil {
				return nil, err
			}
			secretList.Items = append(secretList.Items, secret)
		case model.ResourceTypeConfigmap:
			var cm api.ConfigMap
			if err = json.Unmarshal([]byte(v.Value), &cm); err != nil {
				return nil, err
			}
			if err = json.Unmarshal(metadata, &cm.ObjectMeta); err != nil {
				return nil, err
			}
			configMapList.Items = append(configMapList.Items, cm)
		case constants.ResourceTypeEndpoints:
			var ep api.Endpoints
			if err = json.Unmarshal([]byte(v.Value), &ep); err != nil {
				return nil, err
			}
			if err = json.Unmarshal(metadata, &ep.ObjectMeta); err != nil {
				return nil, err
			}
			endPointsList.Items = append(endPointsList.Items, ep)
		case model.ResourceTypeNode:
			var no api.Node
			if err = json.Unmarshal([]byte(v.Value), &no); err != nil {
				return nil, err
			}
			if err = json.Unmarshal(metadata, &no.ObjectMeta); err != nil {
				return nil, err
			}
			nodeList.Items = append(nodeList.Items, no)
		}
	}
	res = append(res, &podList, &serviceList, &secretList, &configMapList, &endPointsList, &nodeList)
	return
}

// ConvertDataToTable Convert the data into table kind to simulate the data sent by api-server
func ConvertDataToTable(obj runtime.Object) (runtime.Object, error) {
	to := metav1.TableOptions{}
	tc := storage.TableConvertor{TableGenerator: k8sprinters.NewTableGenerator().With(printersinternal.AddHandlers)}

	return tc.ConvertToTable(context.TODO(), obj, &to)
}

// JSONYamlPrint Output the data in json|yaml format
func JSONYamlPrint(results []dao.Meta, printer printers.ResourcePrinter, out io.Writer) error {
	var obj runtime.Object
	list := v1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		ListMeta: metav1.ListMeta{},
	}

	objectList, err := ParseMetaToV1List(results)
	if err != nil {
		return err
	}

	if len(objectList) != 1 {
		for _, info := range objectList {
			if info == nil {
				continue
			}
			o := info.DeepCopyObject()
			list.Items = append(list.Items, runtime.RawExtension{Object: o})
		}

		listData, err := json.Marshal(list)
		if err != nil {
			return err
		}

		converted, err := runtime.Decode(unstructured.UnstructuredJSONScheme, listData)
		if err != nil {
			return err
		}
		obj = converted
	} else {
		obj = objectList[0]
	}
	if err := PrintGeneric(printer, obj, out); err != nil {
		return err
	}

	return nil
}

// ParseMetaToV1List Convert the data to the corresponding list type
// The type definition used by apiserver does not have the omitempty definition of json, will introduce a lot of useless null information
// Use v1 type definition to get data here
// Only used by JSONYamlPrint.
func ParseMetaToV1List(results []dao.Meta) ([]runtime.Object, error) {
	value := make(map[string]interface{})
	list := make([]runtime.Object, 0)

	for _, v := range results {
		if err := json.Unmarshal([]byte(v.Value), &value); err != nil {
			return nil, err
		}
		metadata, err := json.Marshal(value["metadata"])
		if err != nil {
			return nil, err
		}

		switch v.Type {
		case model.ResourceTypePod:
			pod := v1.Pod{}

			status, err := json.Marshal(value["status"])
			if err != nil {
				return nil, err
			}
			spec, err := json.Marshal(value["spec"])
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(metadata, &pod.ObjectMeta); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(spec, &pod.Spec); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(status, &pod.Status); err != nil {
				return nil, err
			}
			pod.APIVersion = "v1"
			pod.Kind = v.Type
			list = append(list, pod.DeepCopyObject())

		case constants.ResourceTypeService:
			svc := v1.Service{}

			status, err := json.Marshal(value["status"])
			if err != nil {
				return nil, err
			}
			spec, err := json.Marshal(value["spec"])
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(metadata, &svc.ObjectMeta); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(spec, &svc.Spec); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(status, &svc.Status); err != nil {
				return nil, err
			}
			svc.APIVersion = "v1"
			svc.Kind = v.Type
			list = append(list, svc.DeepCopyObject())
		case model.ResourceTypeSecret:
			secret := v1.Secret{}

			data, err := json.Marshal(value["data"])
			if err != nil {
				return nil, err
			}
			typeTmp, err := json.Marshal(value["type"])
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(metadata, &secret.ObjectMeta); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(data, &secret.Data); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(typeTmp, &secret.Type); err != nil {
				return nil, err
			}
			secret.APIVersion = "v1"
			secret.Kind = v.Type
			list = append(list, secret.DeepCopyObject())
		case model.ResourceTypeConfigmap:
			cmp := v1.ConfigMap{}
			data, err := json.Marshal(value["data"])
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(metadata, &cmp.ObjectMeta); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(data, &cmp.Data); err != nil {
				return nil, err
			}
			cmp.APIVersion = "v1"
			cmp.Kind = v.Type
			list = append(list, cmp.DeepCopyObject())
		case constants.ResourceTypeEndpoints:
			ep := v1.Endpoints{}
			if err := json.Unmarshal([]byte(v.Value), &value); err != nil {
				return nil, err
			}
			metadata, err := json.Marshal(value["metadata"])
			if err != nil {
				return nil, err
			}
			subsets, err := json.Marshal(value["subsets"])
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(metadata, &ep.ObjectMeta); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(subsets, &ep.Subsets); err != nil {
				return nil, err
			}
			ep.APIVersion = "v1"
			ep.Kind = v.Type
			list = append(list, ep.DeepCopyObject())
		case model.ResourceTypeNode:
			node := v1.Node{}
			status, err := json.Marshal(value["status"])
			if err != nil {
				return nil, err
			}
			spec, err := json.Marshal(value["spec"])
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(metadata, &node.ObjectMeta); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(status, &node.Status); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(spec, &node.Spec); err != nil {
				return nil, err
			}
			node.APIVersion = "v1"
			node.Kind = v.Type
			list = append(list, node.DeepCopyObject())
		default:
			return nil, fmt.Errorf("Parsing failed, unrecognized type: %v. ", v.Type)
		}
	}
	return list, nil
}

// PrintGeneric Output object data to out stream through printer
func PrintGeneric(printer printers.ResourcePrinter, obj runtime.Object, out io.Writer) error {
	isList := meta.IsListType(obj)
	if isList {
		items, err := meta.ExtractList(obj)
		if err != nil {
			return err
		}

		// take the items and create a new list for display
		list := &unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"kind":       "List",
				"apiVersion": "v1",
				"metadata":   map[string]interface{}{},
			},
		}
		if listMeta, err := meta.ListAccessor(obj); err == nil {
			list.Object["metadata"] = map[string]interface{}{
				"selfLink":        listMeta.GetSelfLink(),
				"resourceVersion": listMeta.GetResourceVersion(),
			}
		}

		for _, item := range items {
			list.Items = append(list.Items, *item.(*unstructured.Unstructured))
		}
		if err := printer.PrintObj(list, out); err != nil {
			return err
		}
	} else {
		var value map[string]interface{}
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		if err := printer.PrintObj(&unstructured.Unstructured{Object: value}, out); err != nil {
			return err
		}
	}

	return nil
}
