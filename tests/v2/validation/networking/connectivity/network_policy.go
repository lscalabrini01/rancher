package connectivity

import (
	"errors"

	"github.com/rancher/shepherd/clients/rancher"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/sshkeys"
	"github.com/rancher/shepherd/extensions/workloads"
	"github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
)

const (
	pingPodProjectName = "ping-project"
	containerName      = "test1"
	containerImage     = "ranchertest/mytestcontainer"
	//Ping once
	pingCmd        = "ping -c 1"
	successfulPing = "0% packet loss"
)

type resourceNames struct {
	core   map[string]string
	random map[string]string
}

// newNames returns a new resourceNames struct
// it creates a random names with random suffix for each resource by using core and coreWithSuffix names
func newNames() *resourceNames {
	const (
		projectName             = "upgrade-wl-project"
		namespaceName           = "namespace"
		deploymentName          = "deployment"
		daemonsetName           = "daemonset"
		secretName              = "secret"
		serviceName             = "service"
		ingressName             = "ingress"
		defaultRandStringLength = 3
	)

	names := &resourceNames{
		core: map[string]string{
			"projectName":    projectName,
			"namespaceName":  namespaceName,
			"deploymentName": deploymentName,
			"daemonsetName":  daemonsetName,
			"secretName":     secretName,
			"serviceName":    serviceName,
			"ingressName":    ingressName,
		},
	}

	names.random = map[string]string{}
	for k, v := range names.core {
		names.random[k] = v + "-" + namegenerator.RandStringLower(defaultRandStringLength)
	}

	return names
}

// newPodTemplateWithTestContainer is a private constructor that returns pod template spec for workload creations
func newPodTemplateWithTestContainer() corev1.PodTemplateSpec {
	testContainer := newTestContainerMinimal()
	containers := []corev1.Container{testContainer}
	return workloads.NewPodTemplate(containers, nil, []corev1.LocalObjectReference{}, nil, nil)
}

// newTestContainerMinimal is a private constructor that returns container for minimal workload creations
func newTestContainerMinimal() corev1.Container {
	pullPolicy := corev1.PullAlways
	return workloads.NewContainer(containerName, containerImage, pullPolicy, nil, nil, nil, nil, nil)
}

func pingCommand(client *rancher.Client, clusterName string, namespace string, podIP string, machine v1.SteveAPIObject) (string, error) {
	_, stevecluster, err := clusters.GetProvisioningClusterByName(client, clusterName, namespace)
	if err != nil {
		return "", err
	}

	sshUser, err := sshkeys.GetSSHUser(client, stevecluster)
	if err != nil {
		return "", err
	}

	sshNode, err := sshkeys.GetSSHNodeFromMachine(client, sshUser, &machine)
	if err != nil {
		return "", err
	}

	pingExecCmd := pingCmd + " " + podIP
	excmdLog, err := sshNode.ExecuteCommand(pingExecCmd)
	if err != nil && !errors.Is(err, &ssh.ExitMissingError{}) {
		return pingExecCmd, err
	}

	logrus.Infof("Log of the ping command {%v}", excmdLog)
	return pingExecCmd, nil
}
