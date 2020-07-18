package debug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/astaxie/beego/orm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

const DefaultDbPath = "/var/lib/kubeedge/edgecore.db"

// NewCmdDebugGet represents the debug get command
func NewCmdDebugGet(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get and format data of available resource types in the local database of the edge node",
		RunE: func(cmd *cobra.Command, args []string) error {
			initDb(getDbPath(cmd))
			if len(args) != 1 {
				klog.Fatal("need to specify exactly one type of output, e.g: keadm debug get pod")
			}
			resourceType := args[0]

			result, err := getResult(resourceType)
			if err != nil {
				return err
			}

			return printResult(result, out, cmd)
		},
	}

	cmd.Flags().StringP("input", "i", DefaultDbPath, "Indicate the edge node database path, the default path is `/var/lib/kubeedge/edgecore.db`")
	cmd.Flags().StringP("output", "o", "", "Indicate the output format. Currently supports formats such as yaml|json|wide")
	return cmd
}

func getResult(resourceType string) (*[]dao.Meta, error) {
	var result *[]dao.Meta
	var err error
	if resourceType == "all" {
		meta := new([]dao.Meta)
		_, err := dbm.DBAccess.QueryTable(dao.MetaTableName).All(meta)
		if err != nil {
			return nil, err
		}
		result = meta
	} else {
		result, err = dao.QueryAllMeta("type", resourceType)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func getDbPath(cmd *cobra.Command) string {
	const flag = "input"
	dbPath := os.Getenv("EDGECORE_DB_PATH")
	if len(dbPath) == 0 {
		dbPath = DefaultDbPath
	}

	if cmd.Flags().Changed(flag) {
		var err error
		dbPath, err = cmd.Flags().GetString(flag)

		if err != nil {
			klog.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
		}
	}

	return dbPath
}

func initDb(dbPath string) {
	const dbName = "default"
	const driverName = "sqlite3"

	orm.RegisterModel(new(dao.Meta))

	// most of the implementation below is from InitDBConfig, except that sync is unnecessary here
	if err := orm.RegisterDriver(driverName, orm.DRSqlite); err != nil {
		klog.Fatalf("Failed to register driver: %v", err)
	}
	if err := orm.RegisterDataBase(
		dbName,
		driverName,
		dbPath); err != nil {
		klog.Fatalf("Failed to register db: %v", err)
	}
	dbm.DBAccess = orm.NewOrm()
	if err := dbm.DBAccess.Using(dbName); err != nil {
		klog.Fatalf("Using db access error %v", err)
	}
}

func printResult(metas *[]dao.Meta, out io.Writer, cmd *cobra.Command) error {
	const flag = "output"
	of, err := cmd.Flags().GetString(flag)
	if err != nil {
		return err
	}

	list := corev1.List{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		ListMeta: metav1.ListMeta{},
	}
	// most of implementation below is from kubectl get
	// convert list to runtime.Object
	for _, v := range *metas {
		byteJSON := []byte(v.Value)
		jsonMap := make(map[string]interface{})
		err := json.Unmarshal(byteJSON, &jsonMap)
		jsonMap["apiVersion"] = "v1"
		jsonMap["kind"] = v.Type

		byteJSON, err = json.Marshal(jsonMap)
		if err != nil {
			return err
		}

		converted, err := runtime.Decode(unstructured.UnstructuredJSONScheme, byteJSON)
		if err != nil {
			return err
		}

		list.Items = append(list.Items, runtime.RawExtension{
			Object: converted,
		})
	}

	jsonlistData, err := json.Marshal(list)
	if err != nil {
		return err
	}
	converted, err := runtime.Decode(unstructured.UnstructuredJSONScheme, jsonlistData)
	if err != nil {
		return err
	}
	// convert to list for display
	items, err := meta.ExtractList(converted)
	if err != nil {
		return err
	}

	displayList := &unstructured.UnstructuredList{
		Object: map[string]interface{}{
			"kind":       "List",
			"apiVersion": "v1",
			"metadata":   map[string]interface{}{},
		},
	}
	if listMeta, err := meta.ListAccessor(converted); err == nil {
		displayList.Object["metadata"] = map[string]interface{}{
			"selfLink":        listMeta.GetSelfLink(),
			"resourceVersion": listMeta.GetResourceVersion(),
		}
	}

	for _, item := range items {
		displayList.Items = append(displayList.Items, *item.(*unstructured.Unstructured))
	}

	switch of {
	case "":
		fmt.Fprintln(out, "KEY")
		for _, v := range *metas {
			fmt.Fprintf(out, "%s\n", v.Key)
		}
	case "json":
		var byteContentIndented bytes.Buffer
		byteContent, err := json.Marshal(displayList)
		if err != nil {
			return err
		}

		err = json.Indent(&byteContentIndented, byteContent, "", "\t")
		if err != nil {
			return err
		}

		content := byteContentIndented.String()
		fmt.Fprintln(out, content)
	case "yaml":
		byteContent, err := json.Marshal(displayList)
		if err != nil {
			return err
		}
		yamlMap := make(map[string]interface{})
		err = json.Unmarshal(byteContent, &yamlMap)
		if err != nil {
			return err
		}

		byteContent, err = yaml.Marshal(yamlMap)
		if err != nil {
			return err
		}
		content := string(byteContent)
		fmt.Fprintln(out, content)
	}
	return nil
}
