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
- [Requirement](#requirement)
- [Proposal Design](#proposal-design)
  - [API / Interfaces](#api--interfaces)
  - [Implementation](#implementation)
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

## Requirement

- **T.B.D**

## Proposal Design

- **T.B.D**

### API / Interfaces

- **T.B.D**

### Implementation

- **T.B.D**

### Testing

- **T.B.D**

## Consideration

- **T.B.D**

## Additional Note

- **T.B.D**
