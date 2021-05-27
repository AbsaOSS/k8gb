package test

import (
	"fmt"
	"github.com/AbsaOSS/gopkg/dns"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/shell"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Workflow struct {
	error      error
	namespace  string
	k8sOptions *k8s.KubectlOptions
	t          *testing.T
	settings   struct {
		ingressResourcePath 	string
		gslbResourcePath 		string
		ingressName				string
	}
	state struct {
		namespaceCreated		bool
		testApp struct {
			name 		string
			isRunning 	bool
			isInstalled bool
		}
		gslb struct {
			name		string
			host		string
			port		int
			isInstalled	bool
		}
	}
}

type Instance struct {
	w *Workflow
}

func NewWorkflow(t *testing.T, cluster string, port int) *Workflow {
	var err error
	if cluster == "" {
		err = fmt.Errorf("empty cluster")
	}
	if port < 1000 {
		err = fmt.Errorf("invalid port")
	}
	w :=  new(Workflow)
	w.namespace = fmt.Sprintf("k8gb-test-%s", strings.ToLower(random.UniqueId()))
	w.k8sOptions = k8s.NewKubectlOptions(cluster, "", w.namespace)
	w.t = t
	w.state.gslb.port = port
	w.error = err
	return w
}

func (w *Workflow) WithIngress(path string) *Workflow {
	if path == "" {
		w.error = fmt.Errorf("empty ingress resource path")
	}
	w.settings.ingressResourcePath = path
	return w
}

func (w *Workflow) WithGslb(path, host string) *Workflow {
	var err error
	if host == "" {
		w.error = fmt.Errorf("empty gslb host")
	}
	if path == "" {
		w.error = fmt.Errorf("empty gslb resource path")
	}
	w.settings.gslbResourcePath, err = filepath.Abs(path)
	if err != nil {
		w.error = fmt.Errorf("reading %s; %s",path, err)
	}
	w.state.gslb.name, err =  w.getManifestName(w.settings.gslbResourcePath)
	if err != nil {
		w.error = err
	}
	w.state.gslb.host = host
	if err != nil {
		w.error = err
	}
	return w
}

func (w *Workflow) WithTestApp() *Workflow {
	w.state.testApp.isInstalled = true
	w.state.testApp.name = "frontend-podinfo"
	return w
}

func (w *Workflow) Start() (*Instance, error) {
	if w.error != nil {
		return nil, w.error
	}

	// namespace
	w.t.Logf("Create namespace %s", w.namespace)
	k8s.CreateNamespace(w.t, w.k8sOptions, w.namespace)
	w.state.namespaceCreated = true

	// gslb
	if w.settings.gslbResourcePath != "" {
		w.t.Logf("Create ingress %s from %s", w.state.gslb.name, w.settings.gslbResourcePath)
		k8s.KubectlApply(w.t, w.k8sOptions, w.settings.gslbResourcePath)
		k8s.WaitUntilIngressAvailable(w.t, w.k8sOptions, w.state.gslb.name, 60, 1*time.Second)
		ingress := k8s.GetIngress(w.t, w.k8sOptions, w.state.gslb.name)
		require.Equal(w.t, ingress.Name, w.state.gslb.name)
		w.settings.ingressName = w.state.gslb.name
	}

	// ingress
	if w.settings.ingressResourcePath != "" {

	}

	// app
	if w.state.testApp.isInstalled {
		const app = "https://stefanprodan.github.io/podinfo"
		w.t.Logf("Create test application %s", app)
		helmRepoAdd := shell.Command{
			Command: "helm",
			Args:    []string{"repo", "add", "--force-update", "podinfo", app},
		}
		helmRepoUpdate := shell.Command{
			Command: "helm",
			Args:    []string{"repo", "update"},
		}
		shell.RunCommand(w.t, helmRepoAdd)
		shell.RunCommand(w.t, helmRepoUpdate)
		helmOptions := helm.Options{
			KubectlOptions: w.k8sOptions,
			Version:        "4.0.6",
		}
		helm.Install(w.t, &helmOptions, "podinfo/podinfo", "frontend")
		testAppFilter := metav1.ListOptions{
			LabelSelector: "app="+w.state.testApp.name,
		}
		k8s.WaitUntilNumPodsCreated(w.t, w.k8sOptions, testAppFilter, 1, 60, 1*time.Second)
		var testAppPods []corev1.Pod
		testAppPods = k8s.ListPods(w.t, w.k8sOptions, testAppFilter)
		for _, pod := range testAppPods {
			k8s.WaitUntilPodAvailable(w.t, w.k8sOptions, pod.Name, 60, 1*time.Second)
		}
		k8s.WaitUntilServiceAvailable(w.t, w.k8sOptions, w.state.testApp.name, 60, 1*time.Second)
		w.state.testApp.isRunning = true
	}
	return &Instance{
		w: w,
	}, nil
}

func (w *Workflow) getManifestName(path string) (string, error) {
	m := struct {
		Metadata struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
	}{}

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("parse %s; %s",path, err)
	}
	err = yaml.Unmarshal(yamlFile, &m)
	if err != nil {
		return "", fmt.Errorf("unmarshall %s; %s",path, err)
	}
	return m.Metadata.Name, nil
}

func (i *Instance) Kill(){
	i.w.t.Logf("killing %s",i)
	if i.w.state.namespaceCreated {
		k8s.DeleteNamespace(i.w.t, i.w.k8sOptions, i.w.namespace)
	}
}

func (i *Instance) GetIngressIPs() []string {
	var ingressIPs []string
	ingress := k8s.GetIngress(i.w.t, i.w.k8sOptions, i.w.settings.ingressName)
	for _, ip := range ingress.Status.LoadBalancer.Ingress {
		ingressIPs = append(ingressIPs, ip.IP)
	}
	return ingressIPs
}

func (i *Instance) StopTestApp() {
	k8s.RunKubectl(i.w.t, i.w.k8sOptions, "scale", "deploy", i.w.state.testApp.name, "--replicas=0")
	assertGslbStatus(i.w.t, i.w.k8sOptions, i.w.state.gslb.name,  i.w.state.gslb.host+":Unhealthy")
	i.w.state.testApp.isRunning = false
}

func (i *Instance) StartTestApp() {
	k8s.RunKubectl(i.w.t, i.w.k8sOptions, "scale", "deploy", i.w.state.testApp.name, "--replicas=1")
	assertGslbStatus(i.w.t, i.w.k8sOptions, i.w.state.gslb.name,  i.w.state.gslb.host+":Healthy")
	i.w.state.testApp.isRunning = true
}

// WaitForGSLB returns IP address list
func (i *Instance) WaitForGSLB(instances... *Instance) ([]string, error){
	var expectedIPs []string
	instances = append(instances,i)
	for _,in := range instances {
		// add expected ip's only if app is running
		if in.w.state.testApp.isRunning {
			ip := in.GetIngressIPs()
			expectedIPs = append(expectedIPs, ip...)
		}
	}
	sort.Strings(expectedIPs)
	return waitForLocalGSLBNew(i.w.t, i.w.state.gslb.host, i.w.state.gslb.port, expectedIPs)
}

func (i *Instance) String() string {
	return fmt.Sprintf("Instance: %s", i.w.namespace)
}


func waitForLocalGSLBNew(t *testing.T, host string, port int, expectedResult []string) (output []string, err error) {
	return DoWithRetryWaitingForValueE(
		t,
		"Wait for failover to happen and coredns to pickup new values...",
		100,
		time.Second*1,
		func() ([]string, error) { return dns.Dig("localhost:"+strconv.Itoa(port), host) },
		expectedResult)
}