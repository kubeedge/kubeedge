package debug

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/astaxie/beego/orm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
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
				klog.Error("need to specify exactly one type of output, e.g: keadm debug get pod")
			}
			result, err := dao.QueryAllMeta("type", args[0])
			if err != nil {
				return err
			}

			return printResult(result, out, cmd)
		},
	}

	cmd.Flags().StringP("dbPath", "d", DefaultDbPath, fmt.Sprintf("dbPath; path to edgecore.db, default: %s", DefaultDbPath))
	cmd.Flags().StringP("output", "o", "", "Output format; available options are 'yaml', 'json'")
	return cmd
}

func getDbPath(cmd *cobra.Command) string {
	dbPath := os.Getenv("EDGECORE_DB_PATH")
	if len(dbPath) == 0 {
		dbPath = DefaultDbPath
	}

	if cmd.Flags().Changed("dbPath") {
		const flag = "dbPath"
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
	if err := orm.RegisterDriver(driverName, orm.DRSqlite); err != nil {
		klog.Fatalf("Failed to register driver: %v", err)
	}
	if err := orm.RegisterDataBase(
		dbName,
		driverName,
		dbPath); err != nil {
		klog.Fatalf("Failed to register db: %v", err)
	}
	// if err := orm.RunSyncdb(dbName, false, true); err != nil {
	// 	klog.Errorf("run sync db error %v", err)
	// }
	dbm.DBAccess = orm.NewOrm()
	if err := dbm.DBAccess.Using(dbName); err != nil {
		klog.Errorf("Using db access error %v", err)
	}
}

func printResult(meta *[]dao.Meta, out io.Writer, cmd *cobra.Command) error {
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
	// convert list to runtime.Object
	for _, v := range *meta {
		byteJSon := []byte(v.Value)
		jsonMap := make(map[string]interface{})
		err := json.Unmarshal(byteJSon, &jsonMap)
		jsonMap["apiVersion"] = "v1"
		jsonMap["kind"] = "Node"

		byteJSon, err = json.Marshal(jsonMap)
		if err != nil {
			return err
		}

		converted, err := runtime.Decode(unstructured.UnstructuredJSONScheme, byteJSon)
		if err != nil {
			return err
		}

		list.Items = append(list.Items, runtime.RawExtension{
			Object: converted,
		})
	}

	// convert to

	switch of {
	case "":
		for _, v := range *meta {
			fmt.Fprintf(out, "%s\n", v.Key)
		}
	case "json":
		listData, err := json.Marshal(list)
		if err != nil {
			return err
		}
		content := string(listData)
		fmt.Fprintln(out, content)
	case "yaml":
		listData, err := yaml.Marshal(list)
		if err != nil {
			return err
		}
		content := string(listData)
		fmt.Fprintln(out, content)
	}
	return nil
}