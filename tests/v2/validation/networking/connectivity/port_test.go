//go:build (validation || infra.rke2k3s || cluster.any || sanity) && !stress && !extended

package connectivity

import (
	"testing"

	projectsapi "github.com/rancher/rancher/tests/v2/actions/projects"
	"github.com/rancher/rancher/tests/v2/actions/workloads"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/charts"
	"github.com/rancher/shepherd/extensions/clusters"
	shepworkloads "github.com/rancher/shepherd/extensions/workloads"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PortTestSuite struct {
	suite.Suite
	client      *rancher.Client
	session     *session.Session
	cluster     *management.Cluster
	clusterName string
}

func (p *PortTestSuite) TearDownSuite() {
	p.session.Cleanup()
}

func (p *PortTestSuite) SetupSuite() {
	p.session = session.NewSession()

	client, err := rancher.NewClient("", p.session)
	require.NoError(p.T(), err)

	p.client = client

	log.Info("Getting cluster name from the config file and append cluster details in connection")
	p.clusterName = client.RancherConfig.ClusterName
	require.NotEmptyf(p.T(), p.clusterName, "Cluster name to install should be set")

	clusterID, err := clusters.GetClusterIDByName(p.client, p.clusterName)
	require.NoError(p.T(), err, "Error getting cluster ID")

	p.cluster, err = p.client.Management.Cluster.ByID(clusterID)
	require.NoError(p.T(), err)
}

func (p *PortTestSuite) TestHostPort() {
	subSession := p.session.NewSession()
	defer subSession.Cleanup()

	log.Info("Creating new project and namespace")
	_, namespace, err := projectsapi.CreateProjectAndNamespace(p.client, p.cluster.ID)
	require.NoError(p.T(), err)

	steveClient, err := p.client.Steve.ProxyDownstream(p.cluster.ID)
	require.NoError(p.T(), err)

	//Make it random
	hostPort := 9999

	testContainerPodTemplate := newPodTemplateWithTestContainer()
	testContainerPodTemplate.Spec.Containers[0].Ports = append(testContainerPodTemplate.Spec.Containers[0].Ports, corev1.ContainerPort{
		HostPort:      int32(hostPort),
		ContainerPort: 80,
		Protocol:      corev1.ProtocolTCP,
	})

	daemonsetName := namegen.AppendRandomString("testdaemonset")

	p.T().Logf("Creating a daemonset with the test container with name [%v]", daemonsetName)
	daemonsetTemplate := shepworkloads.NewDaemonSetTemplate(daemonsetName, namespace.Name, testContainerPodTemplate, true, nil)
	createdDaemonSet, err := steveClient.SteveType(workloads.DaemonsetSteveType).Create(daemonsetTemplate)
	require.NoError(p.T(), err)
	assert.Equal(p.T(), createdDaemonSet.Name, daemonsetName)

	p.T().Logf("Waiting for daemonset [%v] to have expected number of available replicas", daemonsetName)
	err = charts.WatchAndWaitDaemonSets(p.client, p.cluster.ID, namespace.Name, metav1.ListOptions{})
	require.NoError(p.T(), err)
}

func TestPortTestSuite(t *testing.T) {
	suite.Run(t, new(PortTestSuite))
}
