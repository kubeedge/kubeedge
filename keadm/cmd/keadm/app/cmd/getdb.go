package cmd

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

// NewCmdGetDb represents the getdb command
func NewCmdGetDb(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getdb",
		Short: "get format output of edgecore.db",
		Run: func(cmd *cobra.Command, args []string) {

			for i, s := range args {
				fmt.Printf("i=%d, s=%s\n", i, s)
			}

			keyData, valueData, err := getRowsData(out, cmd)
			CheckErr(err, fatal)

			err = runOutput(keyData, valueData, out, cmd)
			CheckErr(err, fatal)
		},
	}

	cmd.Flags().StringP("path", "p", "/var/lib/kubeedge/edgecore.db", "Path of db file; default: /var/lib/kubeedge/edgecore.db")
	cmd.Flags().StringP("output", "o", "", "Output format; available options are 'yaml', 'json'")

	return cmd
}

func getRowsData(out io.Writer, cmd *cobra.Command) (*[]string, *[]string, error) {
	const flag = "path"
	dbPath, err := cmd.Flags().GetString(flag)
	if err != nil {
		klog.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, nil, err
	}

	stmt := fmt.Sprintf("select key, type, value from %s where type='pod';", dao.MetaTableName)

	rows, err := db.Query(stmt)
	if err != nil {
		return nil, nil, err
	}

	keyData := make([]string, 0)
	valueData := make([]string, 0)

	defer rows.Close()
	for rows.Next() {
		var _key string
		var _type string
		var _value string
		err = rows.Scan(&_key, &_type, &_value)
		if err != nil {
			klog.Fatal(err)
		}
		keyData = append(keyData, _key)
		valueData = append(valueData, _value)
	}

	err = rows.Err()
	if err != nil {
		return nil, nil, err
	}

	return &keyData, &valueData, nil
}

func runOutput(keyData *[]string, valueData *[]string, out io.Writer, cmd *cobra.Command) error {
	const flag = "output"
	of, err := cmd.Flags().GetString(flag)
	if err != nil {
		klog.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}

	switch of {
	case "":
		fmt.Fprintf(out, "KEY\n%s\n", strings.Join(*keyData, "\n"))
		return nil
	case "json":
		for i, v := range *valueData {
			var bytesContent bytes.Buffer
			err := json.Indent(&bytesContent, []byte(v), "", "\t")
			if err != nil {
				return err
			}
			content := bytesContent.String()
			content = fmt.Sprintf("---------------%s---------------\n%s\n", (*keyData)[i], content)

			fmt.Fprint(out, content)
		}

		return nil
	case "yaml":
		return nil
	}
	return nil
}
