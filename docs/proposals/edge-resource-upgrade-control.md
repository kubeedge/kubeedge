---
title: Edge Resource Upgrade Control
status: implementable
authors:
  - "@fujitatomoya"
  - "@FengGaoCSC"
  - "@XmchxUp"
approvers:
  - "@WillardHu"
  - "@Shelley-BaoYue"
creation-date: 2025-06-11
last-updated: 2025-XX-XX
---

# Edge Resource Upgrade Control

- [Motivation / Background](#motivation--background)
- [Use cases](#use-cases)
- [Proposal Design](#proposal-design)
  - [Annotation](#annotation)
  - [MetaServer/MetaService API](#metaservermetaservice-api)
  - [`keadm ctl` Extension](#keadm-ctl-extension)
  - [Testing](#testing)
- [Consideration](#consideration)
- [Additional Note](#additional-note)

## Motivation / Background

For the use cases such as drones, robotics/robots and autonomous cars, the edge system application should be able to hold/release the lock for the resources (Pods, ConfigMap, Deployment and so on) updated at edge, so that the resources cannot be updated or upgraded unless it confirms the device status.
The user application (likely system application running in the host system where `edgecore` runs) can confirm and control the resource update timing to release the lock.
For example, when the drone is on the ground and only in the updating state, `edged` redeploys the pods and deployments with requesting to `containerd`.
The same requirement for robotics/robots, only when the system application determines that it is safe to update the application pods or any other resources, it can do so accordingly.
Otherwise, `edged` sends the request to `containerd` to recreate the resource such as pods and deployments once queued to `MetaManager` via `CloudCore(EdgeController)`.
Usually this would not be the problem for the use cases such as sensing data collection or completely stateless application, this is not going to be any problems because it can update the resources anytime requested by Kubernetes.
Probably just temporarily sensing data can be lost, but it does not really matter for those use cases.
The sensing data will be coming once the new application pods are deployed and started.

This sounds pretty much for common use case at edge application lifecycle management.
This feature enables us to control the update timing to confirm and release the lock once the device is ready to be updated at edge itself.

## Use cases

- Robotics / Robots
  An automatic resource update in the middle of the operation or actuation could interrupt motion, possibly causing actuator lock-up or crash, production halt, safety hazard to nearby human operators.
  Robot system signals when the actuators are idle and in safe pose, only then does `edged` apply the updated container to ensure updates occur during defined maintenance windows or pause states.

- Autonomous Car / AMR / AGV
  If a resource update restarts the perception or control module mid-navigation, vehicle may stop unexpectedly, possibly causing risk of collision or failure to navigate, loss of customer trust and service reliability.
  The local system inside the vehicle controls when it is parked or charging, it toggles the flag or sends a signal to enable the resource update only when the car system is ready to do so.
  This ensures zero disruption to the driving session.

- Drone / Aerospace
  If the update hits mid-flight, the pod restart disconnects the telemetry stream or flight control interface.
  This could possibly trigger emergency landing or flyaway condition, the worst case is crash down on the ground.
  Edge device onboard drone (e.g. PX4 companion computer) knows flight state always, then signals when landed or in safe altitude hold mode.
  Resource updates only can be applied to post-flight or during downtime.

## Proposal Design

We can use annotations like `edge.kubeedge.io/hold-upgrade: "true"` on Deployments, StatefulSets, DaemonSets, etc., to indicate that their Pod updates should be held at the edge unless the edge system or application releases the lock.
It propagates this state via a new PodCondition (e.g. `HoldUpgrade`) so the cloud knows that the update is deferred.
Add a new MetaServer API and `keadm ctl command` (`keadm ctl unhold-upgrade pod <pod>` and `keadm ctl unhold-upgrade node`) to release the hold and allow the update to apply.

### Annotation

We can define new annotation key, `edge.kubeedge.io/hold-upgrade: "true"`, then at cloud-side `EdgeController` can watch the events for Deployment/StatefulSet, detect annotation changes.
When they are detected, it can set PodCondition `"HeldUpgrade": "True"` on newly created Pods and resources.
And it should stop sending Pod UPDATE messages to edge for this resource group.

```yaml
status:
  conditions:
    - type: HeldUpgrade
      status: "True"
      reason: UpdateHoldActive
      message: Pod upgrade is currently held at the edge.
```

### MetaServer/MetaService API

It needs to extend KubeEdge metaserver with new endpoints, `POST /edge/unhold/pod/{podName}` — clears the hold annotation and resumes upgrades, and likewise for node-level holds.

### `keadm ctl` Extension

We can add the subcommand to `keadm ctl` as followings.

```bash
keadm ctl unhold-upgrade pod <pod-name>   ### release the specific pod upgrade only
keadm ctl unhold-upgrade node   ### release the node wide upgrade, all help-upgrade pods are restarted
```

These commands internally call on newly developed MetaServer handlers via MetaService APIs described above to send the unhold signal.

### Testing

- Corresponding test should be added align with `kubeedge/edge/pkg/metamanager/metaserver/handlerfactory/extend_confirm_upgrade_test.go`.

## Consideration

- What kinds of resource need to be managed and controlled by this new feature?
  Currently Pod, Deployments, StatefulSets, DaemonSets are in scope, but what about ConfigMaps and Secrets?
  Those are likely to be bound to the workloads, and it does not automatically rebound the configuration to the workloads unless the workloads are restarted and redeployed.
  So I would suggest that ConfigMaps and Secrets are out of scope for this feature at this moment.

  | Resource | Control Required | Description / Reason |
  | -------- | ---------------- | -------------------- |
  | **Pods**           | ✅ Yes    | Primary unit of runtime workload. Restart affects running application. |
  | **Deployments**    | ✅ Yes    | Controls rollout strategy. Might recreate Pods if spec changes. |
  | **StatefulSets**   | ✅ Yes    | Stateful services (like databases); restart or scale could be dangerous. |
  | **DaemonSets**     | ✅ Yes    | Often used in edge use-cases for agents, telemetry, etc. |
  | **ConfigMaps**     | ⚠️ *Maybe* | Apps *mount* or *env-inject* values; updates have no effect unless pod restarts. |
  | **Secrets**        | ⚠️ *Maybe* | Same behavior as ConfigMaps — no automatic propagation into running Pods. |
  | **CRDs / Volumes** | ❓ Maybe   | Depending on use case. Might not require gating, but may be referenced indirectly. |

## Additional Note

- N.A
