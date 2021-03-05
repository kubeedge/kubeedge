# EdgeMesh and EdgeCore decoupling design

## Motivation

```
EdgeMesh supports service mesh functions of the edge, including service discovery and traffic management. The current EdgeMesh functions are relatively independent and suitable for decoupling from EdgeCore to deploy as an application container, facilitating the subsequent upgrade and maintenance of EdgeMesh.
```

## Goals

- ```
  The EdgeMesh module is decoupled from EdgeCore and supports EdgeMesh to be deployed as an independent application container.
  ```

- ```
  The deployment methods also ensure forward compatibility, deployment methods supports users to choose EdgeMesh container deployment or built-in EdgeCore.
  ```

- *In the future, the new service mesh features will only be reflected on the EdgeMesh module decoupled from EdgeCore.*

## Proposal

```
The function of EdgeMesh is relatively independent, and it is suitable for decoupling from EdgeCore and deploying as an application container, which is convenient for subsequent upgrade and maintenance.
```

### Changes

1、Deployment Method：

- ```
  Support EdgeMesh to be independent of EdgeCore and be deployed as an application container.
  ```

- ```
  The deployment methods supports forward compatibility, retains the ability of the enable module EdgeMesh in EdgeCore. Users can choose container deployment or built-in EdgeCore deployment.
  ```

- ```
  The installation tool keadm supports users to choose the deployment method of EdgeMesh: EdgeMesh module built in EdgeCore or  is deployed as a container application. By default, the application deployment method is adopted.
  ```

2、Communication：

- ```
  If EdgeMesh is deployed as an independent application container, the message channels for the addition, deletion and modification of the "service" service object involved in EdgeMesh will be changed from the original "channel" communication to the "list-watch" communication . For the specific list-watch design, please refer to https://github.com/kubeedge/kubeedge/pull/2508.
  ```

