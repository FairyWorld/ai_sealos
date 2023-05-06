/*
Copyright 2023 cuisongliu@qq.com.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	proxy "k8s.io/kube-proxy/config/v1alpha1"
	kubelet "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	"github.com/labring/sealos/pkg/utils/logger"
	"github.com/labring/sealos/test/e2e/testhelper"
	"github.com/labring/sealos/test/e2e/testhelper/cmd"
)

type Interface interface {
	Verify() error
}

type fakeClient struct {
	InitConfiguration      kubeadm.InitConfiguration
	ClusterConfiguration   kubeadm.ClusterConfiguration
	JoinConfiguration      kubeadm.JoinConfiguration
	KubeProxyConfiguration proxy.KubeProxyConfiguration
	KubeletConfiguration   kubelet.KubeletConfiguration
	clusterName            string
	cmd                    cmd.Interface
}

type FakeClientGroup struct {
	clients []Interface
}

type FakeOpts struct {
	Socket      string
	Cgroup      string
	PodCIDR     string
	ServiceCIDR string
	CertSan     string
	Taints      map[string]string
	Images      []string
	Etcd        []string
}

func NewFakeGroupClient(name string, opt *FakeOpts) (*FakeClientGroup, error) {
	client := &fakeClient{clusterName: name, cmd: &cmd.LocalCmd{}}
	err := client.loadYaml()
	if err != nil {
		return nil, err
	}
	if opt == nil {
		opt = &FakeOpts{}
	}

	if opt.Socket == "" {
		opt.Socket = "/run/containerd/containerd.sock"
	}
	if opt.Cgroup == "" {
		opt.Cgroup = "systemd"
	}
	if opt.PodCIDR == "" {
		opt.PodCIDR = "100.64.0.0/10"
	}
	if opt.ServiceCIDR == "" {
		opt.ServiceCIDR = "10.96.0.0/22"
	}
	if opt.Etcd == nil {
		opt.Etcd = make([]string, 0)
	}

	return &FakeClientGroup{clients: []Interface{
		&fakeSocketClient{fakeClient: client, data: opt.Socket},
		&fakeCgroupClient{fakeClient: client, data: opt.Cgroup},
		&fakePodCIDRClient{fakeClient: client, data: opt.PodCIDR},
		&fakeServiceCIDRClient{fakeClient: client, data: opt.ServiceCIDR},
		&fakeCertSansClient{fakeClient: client, data: opt.CertSan},
		&fakeSingleTaintsClient{fakeClient: client},
		&fakeTaintsClient{fakeClient: client, data: opt.Taints},
		&fakeImageClient{fakeClient: client, data: opt.Images},
		&fakeEtcdClient{fakeClient: client, etcd: opt.Etcd},
	}}, nil
}

func (f *FakeClientGroup) Verify() error {
	for _, client := range f.clients {
		err := client.Verify()
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeClient) loadYaml() error {
	logger.Info("verify default cluster info")
	initFile := fmt.Sprintf("/root/.sealos/%s/etc/kubeadm-init.yaml", f.clusterName)
	if !testhelper.IsFileExist(initFile) {
		return fmt.Errorf("file %s not exist", initFile)
	}
	data, err := os.ReadFile(filepath.Clean(initFile))
	if err != nil {
		return err
	}
	yamls := testhelper.ToYalms(string(data))
	for _, yamlString := range yamls {
		obj, _ := testhelper.UnmarshalData([]byte(yamlString))
		kind, _, _ := unstructured.NestedString(obj, "kind")
		switch kind {
		case "InitConfiguration":
			_ = yaml.Unmarshal([]byte(yamlString), &f.InitConfiguration)
		case "ClusterConfiguration":
			_ = yaml.Unmarshal([]byte(yamlString), &f.ClusterConfiguration)
		case "JoinConfiguration":
			_ = yaml.Unmarshal([]byte(yamlString), &f.JoinConfiguration)
		case "KubeProxyConfiguration":
			_ = yaml.Unmarshal([]byte(yamlString), &f.KubeProxyConfiguration)
		case "KubeletConfiguration":
			_ = yaml.Unmarshal([]byte(yamlString), &f.KubeletConfiguration)
		}
	}
	return nil
}
