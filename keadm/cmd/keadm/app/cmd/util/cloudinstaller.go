package util

import (
	"context"
	"fmt"
	"os"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// KubeCloudInstTool embeds Common struct
// It implements ToolsInstaller interface
type KubeCloudInstTool struct {
	Common
	AdvertiseAddress string
	DNSName          string
	TarballPath      string
	CloudCoreRunMode string
	Replicas         int32
	NodeSelector     map[string]string
}

// InstallTools downloads KubeEdge for the specified version
// and makes the required configuration changes and initiates cloudcore.
func (cu *KubeCloudInstTool) InstallTools() error {
	cu.SetOSInterface(GetOSInterface())
	cu.SetKubeEdgeVersion(cu.ToolVersion)

	if cu.IsCloudCoreRunningAsContainer() {
		// install prerequisite resource: configmap, serviceaccount, and etc.
		err := cu.InstallCloudCoreResource()
		if err != nil {
			return err
		}

		err = cu.RunCloudCore()
		if err != nil {
			return err
		}
		fmt.Println("CloudCore started")
		return nil
	}

	opts := &types.InstallOptions{
		TarballPath:   cu.TarballPath,
		ComponentType: types.CloudCore,
	}

	err := cu.InstallKubeEdge(*opts)
	if err != nil {
		return err
	}

	//This makes sure the path is created, if it already exists also it is fine
	err = os.MkdirAll(KubeEdgeConfigDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeConfigDir)
	}

	cloudCoreConfig := v1alpha1.NewDefaultCloudCoreConfig()
	if cu.KubeConfig != "" {
		cloudCoreConfig.KubeAPIConfig.KubeConfig = cu.KubeConfig
	}

	if cu.Master != "" {
		cloudCoreConfig.KubeAPIConfig.Master = cu.Master
	}

	if cu.AdvertiseAddress != "" {
		cloudCoreConfig.Modules.CloudHub.AdvertiseAddress = strings.Split(cu.AdvertiseAddress, ",")
	}

	if cu.DNSName != "" {
		cloudCoreConfig.Modules.CloudHub.DNSNames = strings.Split(cu.DNSName, ",")
	}

	if err := types.Write2File(KubeEdgeCloudCoreNewYaml, cloudCoreConfig); err != nil {
		return err
	}

	err = cu.RunCloudCore()
	if err != nil {
		return err
	}
	fmt.Println("CloudCore started")

	return nil
}

// InstallCloudCoreResource install necessary resource: including namespace, serviceaccount, clusterrole, clusterrolebinding and configmap
func (cu *KubeCloudInstTool) InstallCloudCoreResource() error {
	var err error
	// if create resource failed, we need to clean all created resource
	defer func() {
		if err != nil {
			err1 := cleanNameSpace(constants.SystemNamespace, cu.KubeConfig)
			if err1 != nil {
				fmt.Println("failed to clean kubeedge namespace", err1)
			}
		}
	}()
	// create namespace
	err = cu.createNamespace()
	if err != nil {
		return err
	}

	// create serviceccount
	err = cu.createServiceAccount()
	if err != nil {
		return err
	}

	// create cluster role
	err = cu.createClusterRole()
	if err != nil {
		return err
	}

	// create cluster role binding
	if err = cu.createClusterRoleBinding(); err != nil {
		return err
	}

	// create config map
	if err = cu.createConfigmap(); err != nil {
		return err
	}

	return nil
}

func (cu *KubeCloudInstTool) createNamespace() error {
	fmt.Println("begin to create kubeedge namespace.")
	cli, err := KubeClient(cu.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubeedge",
			Labels: map[string]string{
				"k8s-app": "kubeedge",
			},
		},
	}
	option := metav1.CreateOptions{}
	_, err = cli.CoreV1().Namespaces().Create(context.Background(), ns, option)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	fmt.Println("create kubeedge namespace successfully.")
	return nil
}

func (cu *KubeCloudInstTool) createServiceAccount() error {
	fmt.Println("begin to create service account.")
	cli, err := KubeClient(cu.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}

	serviceAccount := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloudcore",
			Namespace: "kubeedge",
			Labels: map[string]string{
				"k8s-app":  "kubeedge",
				"kubeedge": "cloudcore",
			},
		},
	}
	option := metav1.CreateOptions{}
	_, err = cli.CoreV1().ServiceAccounts(serviceAccount.Namespace).Create(context.Background(), serviceAccount, option)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	fmt.Println("create service account successfully.")
	return nil
}

func (cu *KubeCloudInstTool) createClusterRole() error {
	fmt.Println("begin to create cluster role.")
	cli, err := KubeClient(cu.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}

	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cloudcore",
			Labels: map[string]string{
				"k8s-app":  "kubeedge",
				"kubeedge": "cloudcore",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes", "nodes/status", "configmaps", "pods", "pods/status", "secrets", "endpoints", "services"},
				Verbs:     []string{"get", "list", "watch", "create", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/status"},
				Verbs:     []string{"patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"delete"},
			},
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"get", "list", "watch", "create", "update"},
			},
			{
				APIGroups: []string{"devices.kubeedge.io"},
				Resources: []string{"devices", "devicemodels", "devices/status", "devicemodels/status"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{"reliablesyncs.kubeedge.io"},
				Resources: []string{"objectsyncs", "clusterobjectsyncs", "objectsyncs/status", "clusterobjectsyncs/status"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{"rules.kubeedge.io"},
				Resources: []string{"rules", "ruleendpoints", "rules/status", "ruleendpoints/status"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		},
	}
	option := metav1.CreateOptions{}
	_, err = cli.RbacV1().ClusterRoles().Create(context.Background(), role, option)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	fmt.Println("create cluster role successfully.")
	return nil
}

func (cu *KubeCloudInstTool) createClusterRoleBinding() error {
	fmt.Println("begin create cluster role binding")
	cli, err := KubeClient(cu.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}

	roleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cloudcore",
			Labels: map[string]string{
				"k8s-app":  "kubeedge",
				"kubeedge": "cloudcore",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cloudcore",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "cloudcore",
				Namespace: "kubeedge",
			},
		},
	}

	option := metav1.CreateOptions{}
	_, err = cli.RbacV1().ClusterRoleBindings().Create(context.Background(), roleBinding, option)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	fmt.Println("create cluster role binding successfully.")
	return nil
}

func (cu *KubeCloudInstTool) createConfigmap() error {
	fmt.Println("begin to create configmap.")
	cli, err := KubeClient(cu.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}

	config := v1alpha1.NewDefaultCloudCoreConfig()
	if cu.KubeConfig != "" {
		config.KubeAPIConfig.KubeConfig = cu.KubeConfig
	}

	if cu.Master != "" {
		config.KubeAPIConfig.Master = cu.Master
	}

	if cu.AdvertiseAddress != "" {
		config.Modules.CloudHub.AdvertiseAddress = strings.Split(cu.AdvertiseAddress, ",")
	}

	if cu.DNSName != "" {
		config.Modules.CloudHub.DNSNames = strings.Split(cu.DNSName, ",")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config yaml, error: %v", err)
	}

	fmt.Println("config is \n", string(data))

	configmap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloudcore",
			Namespace: "kubeedge",
			Labels: map[string]string{
				"k8s-app":  "kubeedge",
				"kubeedge": "cloudcore",
			},
		},
		Data: map[string]string{
			"cloudcore.yaml": string(data),
		},
	}

	option := metav1.CreateOptions{}
	_, err = cli.CoreV1().ConfigMaps(configmap.Namespace).Create(context.Background(), configmap, option)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	fmt.Println("create configmap successfully.")
	return nil
}

func (cu *KubeCloudInstTool) deployCloudCore() error {
	fmt.Println("begin to create deployment.")
	cli, err := KubeClient(cu.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}

	privileged := true
	hostPathFile := v1.HostPathFile

	image := "kubeedge/cloudcore:v" + cu.ToolVersion.String()

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloudcore",
			Namespace: "kubeedge",
			Labels: map[string]string{
				"k8s-app":  "kubeedge",
				"kubeedge": "cloudcore",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &cu.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"k8s-app":  "kubeedge",
					"kubeedge": "cloudcore",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k8s-app":  "kubeedge",
						"kubeedge": "cloudcore",
					},
				},
				Spec: v1.PodSpec{
					// configure the nodeSelector here to directly schedule pods to specific nodes
					NodeSelector: cu.NodeSelector,
					ReadinessGates: []v1.PodReadinessGate{
						{
							ConditionType: "kubeedge.io/CloudCoreIsLeader",
						},
					},
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/edge",
												Operator: v1.NodeSelectorOpDoesNotExist,
											},
										},
									},
								},
							},
						},
						PodAffinity: &v1.PodAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "kubeedge",
												Operator: metav1.LabelSelectorOpIn,
												Values: []string{
													"cloudcore",
												},
											},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					HostNetwork: true,
					Containers: []v1.Container{
						{
							Name:            "cloudcore",
							Image:           image,
							ImagePullPolicy: v1.PullIfNotPresent,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 10000,
									Name:          "cloudhub",
									Protocol:      v1.ProtocolTCP,
								},
								{
									ContainerPort: 10002,
									Name:          "certandreadyz",
									Protocol:      v1.ProtocolTCP,
								},
							},
							Resources: v1.ResourceRequirements{
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("200m"),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("100m"),
									v1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "conf",
									MountPath: "/etc/kubeedge/config",
								},
								{
									Name:      "kubeconfig",
									MountPath: cu.KubeConfig,
								},
							},
							Env: []v1.EnvVar{
								{
									Name: "CLOUDCORE_POD_NAME",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "CLOUDCORE_POD_NAMESPACE",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
							},
						},
					},
					RestartPolicy:            v1.RestartPolicyAlways,
					DeprecatedServiceAccount: "cloudcore",
					ServiceAccountName:       "cloudcore",
					Volumes: []v1.Volume{
						{
							Name: "conf",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "cloudcore",
									},
								},
							},
						},
						{
							Name: "kubeconfig",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: cu.KubeConfig,
									Type: &hostPathFile,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = cli.AppsV1().Deployments(deployment.Namespace).Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	fmt.Println("create deployment successfully.")
	return nil
}

func (cu *KubeCloudInstTool) RunCloudCoreContainer() error {
	var err error
	defer func() {
		if err != nil {
			if err := cleanNameSpace(constants.SystemName, cu.KubeConfig); err != nil {
				fmt.Println("failed to clean all kubeedge namespace resource: ", err)
			}
		}
	}()
	// deployment.yaml
	if err = cu.deployCloudCore(); err != nil {
		return err
	}

	return nil
}

func (cu *KubeCloudInstTool) RunCloudCoreBinary() error {
	// create the log dir for kubeedge
	err := os.MkdirAll(KubeEdgeLogPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeLogPath)
	}

	if err := os.MkdirAll(KubeEdgeUsrBinPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create %s folder path", KubeEdgeUsrBinPath)
	}

	// add +x for cloudcore
	command := fmt.Sprintf("chmod +x %s/%s", KubeEdgeUsrBinPath, KubeCloudBinaryName)
	cmd := NewCommand(command)
	if err := cmd.Exec(); err != nil {
		return err
	}

	// start cloudcore
	command = fmt.Sprintf("%s/%s > %s/%s.log 2>&1 &", KubeEdgeUsrBinPath, KubeCloudBinaryName, KubeEdgeLogPath, KubeCloudBinaryName)

	cmd = NewCommand(command)

	if err := cmd.Exec(); err != nil {
		return err
	}

	fmt.Println(cmd.GetStdOut())

	fmt.Println("KubeEdge cloudcore is running, For logs visit: ", KubeEdgeLogPath+KubeCloudBinaryName+".log")

	return nil
}

//RunCloudCore starts cloudcore process
func (cu *KubeCloudInstTool) RunCloudCore() error {
	if cu.IsCloudCoreRunningAsContainer() {
		return cu.RunCloudCoreContainer()
	}
	return cu.RunCloudCoreBinary()
}

// IsCloudCoreRunningAsContainer determine whether cloudcore running in the container mode
func (cu *KubeCloudInstTool) IsCloudCoreRunningAsContainer() bool {
	return cu.CloudCoreRunMode == types.CloudCoreContainerRunMode
}

//TearDown method will remove the edge node from api-server and stop cloudcore process
func (cu *KubeCloudInstTool) TearDown() error {
	cu.SetOSInterface(GetOSInterface())
	cu.SetKubeEdgeVersion(cu.ToolVersion)

	fmt.Println("begin to teardown cloudcore.")
	// in container mode, no need to kill process, only need to clean all resources in namespace kubeedge
	if !cu.IsCloudCoreRunningAsContainer() {
		fmt.Println("begin to kill process.")
		//Kill cloudcore process
		if err := cu.KillKubeEdgeBinary(KubeCloudBinaryName); err != nil {
			return err
		}
		fmt.Println("kill process successfully.")
	}

	fmt.Println("begin to clean all resouces in kubeedge namespace.")
	// clean kubeedge namespace
	err := cleanNameSpace(constants.SystemNamespace, cu.KubeConfig)
	if err != nil {
		return fmt.Errorf("fail to clean kubeedge namespace, err:%v", err)
	}
	fmt.Println("clean all resouces in kubeedge namespace successfully.")
	fmt.Println("teardown cloudcore successfully.")
	return nil
}
