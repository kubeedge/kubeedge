# Run KubeEdge

## Cloudcore on Cloud side

If you have copied the cloudcore binary in a folder and the configuration (conf) are stored in the same folder

```shell
 cd ~/folder/
 nohup ./cloudcore &
```

or

If you have setup using the systemctl

```shell
 sudo systemctl start cloudcore
```

## Run Edgecore on Edge side

```shell
 cp $GOPATH/src/github.com/kubeedge/kubeedge/edge/edgecore ~/cmd/
 cd ~/cmd
 ./edgecore
 ```
 or

 ```shell
 nohup ./edgecore > edgecore.log 2>&1 &
 ```

 If you have setup using the systemctl

```
 sudo systemctl start edgecore
```

**Note:** Please run edgecore using the users who have root permission.
