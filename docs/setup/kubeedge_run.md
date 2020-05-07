# Run KubeEdge

**Note:** This step is only required if KubeEdge is installed using Source. If KubeEdge is installed using `keadm` ignore this step.
***

## Cloudcore on Cloud side

If you have copied the cloudcore binary in a folder and the configuration (conf) are stored in the same folder

```shell
cd ~/kubeedge/
nohup ./cloudcore &
```

or

 ```shell
nohup ./cloudcore > cloudcore.log 2>&1 &
 ```

If you have setup using the systemctl

Run cloudcore with systemd

It is also possible to start the cloudcore with systemd. If you want, you could use the example systemd-unit-file. The following command will show you how to setup this:

```shell
sudo ln build/tools/cloudcore.service /etc/systemd/system/cloudcore.service

sudo systemctl daemon-reload
sudo systemctl start cloudcore
```

**Note:** Please fix __ExecStart__ path in cloudcore.service. Do __NOT__ use relative path, use absolute path instead.

If you also want also an autostart, you have to execute this, too:

```shell
 sudo systemctl enable cloudcore
 ```

```shell
sudo systemctl start cloudcore
```

## Run Edgecore on Edge side

```shell
cd ~/kubeedge
./edgecore
 ```

 or

 ```shell
nohup ./edgecore > edgecore.log 2>&1 &
 ```

If you have setup using the systemctl
 
Run edgecore with systemd
 
It is also possible to start the edgecore with systemd. If you want, you could use the example systemd-unit-file.
 
```shell
sudo ln build/tools/edgecore.service /etc/systemd/system/edgecore.service
sudo systemctl daemon-reload
sudo systemctl start edgecore
```
 
**Note:** Please fix __ExecStart__ path in edgecore.service. Do __NOT__ use relative path, use absolute path instead.
 
If you also want also an autostart, you have to execute this, too:
 
```shell
sudo systemctl enable edgecore
```

```shell
sudo systemctl start edgecore
```

**Note:** Please run edgecore using the users who have root permission.
