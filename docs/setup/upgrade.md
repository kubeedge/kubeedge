# Upgrading KubeEdge

Please refer to following guide to upgrade your KubeEdge cluster.

## Backup

### Database

Backup edgecore database at each edge node:

```
$ mkdir -p /tmp/kubeedge_backup
$ cp /var/lib/kubeedge/edgecore.db /tmp/kubeedge_backup/
```

### Config(Optional)

You can keep old config to save some custom changes as you wish.

*Note*:

After upgrading, some options may be deleted and some may be added, please don't use old config directly.

### Device related(Optional)

If you upgrade from 1.3 to 1.4, please note that we upgrade device API from v1alpha1 to v1alpha2.

You need to install [Device v1alpha2](https://github.com/kubeedge/kubeedge/blob/release-1.4/build/crds/devices/devices_v1alpha2_device.yaml)
and [DeviceModel v1alpha2](https://github.com/kubeedge/kubeedge/blob/release-1.4/build/crds/devices/devices_v1alpha2_devicemodel.yaml),
and manually convert their existing custom resources from v1alpha1 to v1alpha2.

It's recommended to keep v1alpha1 CRD and custom resources in the cluster or exported somewhere, in case any rollback is needed.

## Stop Processes

Stop edgecore processes one by one, after ensuring all edgecore processes are stopped, stop cloudcore.

The way to stop depends on how you deploy:
- for binary or "keadm": use `kill`
- for "systemd": use `systemctl`

## Clean up

```
$ rm -rf /var/lib/kubeedge /etc/kubeedge
```

## Restore Database

Restore database at each edge node:

```
$ mkdir -p /var/lib/kubeedge
$ mv /tmp/kubeedge_backup/edgecore.db /var/lib/kubeedge/
```

## Deploy

Read the [setup](./keadm.md) for deployment.
