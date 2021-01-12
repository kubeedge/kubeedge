---
title: Gateway support at edge
authors:
  - "@FengFees"
approvers:
  - "@kevin-wangzefeng"
  - "@anyushun"
  - "@fisherxu"
creation-date: 2020-11-20
last-updated: 2020-01-12
status: implementing
---

# Edge-Gateway support at edge

## Abstract

Gateway or edge service in a microservice system provide entry point for the entire application so that external applications can visit this application.

In the native kubernetes, [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) is provided to achieve the purpose. Ingress exposes HTTP routes from outside the cluster to services within the cluster. Using ingress the user can bring all their services under a single umbrella and manage them easily. 

In this proposal, we present a design for the gateway/edge service support at edge which is suitable for the edge computing scenario. The gateway/edge service support should continue to work even in off-line  scenarios i.e. even in the cases where connectivity between edge and cloud is lost/unreliable.

## Motivation

The cloud native and microservice architecture is becoming more and more popular and the edge node is becoming more and more powerful. In these scenarios, users might want to deploy their applications at the edge, which could need to be accessed by external users, through services. Using ingress we would be able to provide name based virtual hosting and load balance traffic.

This design document deals with enabling users to leverage powerful machines at the edge to provide external users (users of the application running as a pod within KubeEdge)access to the services which are running on different nodes. 

## Goals

1. Provide EdgeGateway as the management side of Ingress. When a user creates an Ingress to the edge, EdgeGateway responds and initiates service discovery and service registration. Through EdgeGateway, it creates an Ingress Controller to the edge and binds the service.
2. Support HTTP/HTTPS/TCP as the external request.
3. Provide name based virtual hosting and load balance traffic.
4. Provide ingress support even in offline scenarios.

- Alpha

  Support HTTP/TCP ingress, the entire cluster first deploys an Ingress Controller.

- Beta

  Support HTTPS ingress, realize edge to deploy Ingress Class, basic functions are the same as native Ingress Class.

## Non-Goals

1. The HA capability for the gateway itself. 



## Design Details

#### Architecture

![edgegateway-arc](..\images\gateway\edgegateway-arc.png)

In the current design the communication between external users and the services are handled by the nginx ingress controller. The communication between the services and their respective application pods will be handled by the edge mesh feature of KubeEdge. 

EdgeGateway includes Edge-discovery and Proxy API. Edge-discovery will trigger the initialization of EdgeGateway configuration process, continue to obtain resource objects in Metamanager, and trigger Proxy API to create Proxy Pod. The Ingress Proxy in the Proxy Pod will create the Nginx Ingress Controller. We are using nginx controller as our ingress controller to handle ingress resources as the default way. 

### Work Flow

Next, introduce the flow of control flow and data flow according to the above figure.

##### step1:

The user first has to deploy the Nginx ingress controller pod and service on the edge through the edge core (edged component to be specific).Once the pod and service have been created, the user can define an ingress resource, which contains the rules based on which the ingress controller forwards the requests from the user.

##### step2:

EdgeHub will trigger EdgeGateway to start, Edge-discovery will continue to monitor Metaanager resource object status, feedback to EdgeGateway in time, and send the message to Proxy API, and then to Ingress Controller pod under Proxy Pod for processing.

##### step3:

EdgeGateway starts the Edge-discovery and Proxy API processes. The Edge-discovery process initializes the configuration and starts to continuously monitor the resource objects in Metamanager. The Proxy API can create an Ingress Controller Pod to the Proxy Pod.

#### Data flow

##### step1:

The external users can access the Nginx service either through a load balancer or a Node port service which would forward requests to the nginx service. The Nginx service forwards the request to the Nginx Ingress Controller pod.

##### step2：

Based on the rules that have been defined in the ingress resource, the ingress controller forwards the request to the appropriate service.

##### step3:

The services then forward the request to the respective application pods through the help of edge mesh feature of KubeEdge.



#### Ingress Resource:

The Ingress resource is the kubernetes resource based on which the ingress controller decides where to forward the incoming traffic. It is basically used to set rules based on which the ingress controller can take action. An example ingress resource is shown below :

`apiVersion: networking.k8s.io/v1beta1` 

`kind: Ingress` 

​	`metadata:`  

​		`name: test-ingress        # name of the ingress resource`  

​		`annotations:`   

​			`nginx.ingress.kubernetes.io/rewrite-target: /` 

​		`spec:`  

​			`rules:`  

​			`- http:`    

​			    `paths:`    

​				`- path: /testpath   # the path specified in the API based on which the controller will route traffic`

​				    `backend:`      

​						`serviceName: test   # the kubernetes service name to which the incoming request needs to be forwarded`      

​						`servicePort: 80`
The ingress contains the information, which is required to configure a reverse proxy or a load balancer to forward requests to the server. It is also possible to set up the TLS negotiation in the ingress resource. It contains the rules against which all incoming requests are matched. Currently, ingress only supports rules for redirecting HTTP traffic.



#### Rules:

Each HTTP rule contains the following information:

- An optional host. In this example, no host is specified, so the rule applies to all inbound HTTP traffic through the IP address specified. If a host is provided (for example, foo.bar.com), the rules apply to that host.
- A list of paths (for example, /testpath), each of which has an associated backend defined with a serviceName and servicePort. Both the host and path must match the content of an incoming request before the load balancer directs traffic to the referenced Service.
- A backend is a combination of Service and port names as described in the Service doc. HTTP (and HTTPS) requests to the Ingress that matches the host and path of the rule are sent to the listed backend.
- A default backend is often configured in an Ingress controller to service any requests that do not match a path in the spec.

