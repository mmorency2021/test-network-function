package diagnostic

import (
	"encoding/json"
	"path"
	"strings"
	"time"

	"github.com/test-network-function/test-network-function/test-network-function/common"
	"github.com/test-network-function/test-network-function/test-network-function/identifiers"
	"github.com/test-network-function/test-network-function/test-network-function/results"

	"github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega"
	"github.com/test-network-function/test-network-function/pkg/tnf"
	"github.com/test-network-function/test-network-function/pkg/tnf/handlers/generic"
	"github.com/test-network-function/test-network-function/pkg/tnf/handlers/nodedebug"
	"github.com/test-network-function/test-network-function/pkg/tnf/handlers/nodenames"
	"github.com/test-network-function/test-network-function/pkg/tnf/reel"
	"github.com/test-network-function/test-network-function/pkg/tnf/testcases"
)

const (
	// defaultTimeoutSeconds contains the default timeout in secons.
	defaultTimeoutSeconds = 20
)

var (
	// defaultTestTimeout is the timeout for the test.
	defaultTestTimeout = time.Duration(defaultTimeoutSeconds) * time.Second

	// nodeSummary stores the raw JSON output of `oc get nodes -o json`
	nodeSummary = make(map[string]interface{})

	cniPlugins = make([]CniPlugin, 0)

	nodesHwInfo = NodesHwInfo{}

	// csiDriver stores the csi driver JSON output of `oc get csidriver -o json`
	csiDriver = make(map[string]interface{})

	// nodesTestPath is the file location of the nodes.json test case relative to the project root.
	nodesTestPath = path.Join("pkg", "tnf", "handlers", "node", "nodes.json")

	// csiDriverTestPath is the file location of the csidriver.json test case relative to the project root.
	csiDriverTestPath = path.Join("pkg", "tnf", "handlers", "csidriver", "csidriver.json")

	// relativeCsiDriverTestPath is the relative path to the csidriver.json test case.
	relativeCsiDriverTestPath = path.Join(pathRelativeToRoot, csiDriverTestPath)

	// pathRelativeToRoot is used to calculate relative filepaths for the `test-network-function` executable entrypoint.
	pathRelativeToRoot = path.Join("..")

	// relativeNodesTestPath is the relative path to the nodes.json test case.
	relativeNodesTestPath = path.Join(pathRelativeToRoot, nodesTestPath)

	// relativeSchemaPath is the relative path to the generic-test.schema.json JSON schema.
	relativeSchemaPath = path.Join(pathRelativeToRoot, schemaPath)

	// schemaPath is the path to the generic-test.schema.json JSON schema relative to the project root.
	schemaPath = path.Join("schemas", "generic-test.schema.json")
)

var _ = ginkgo.Describe(common.DiagnosticTestKey, func() {
	if testcases.IsInFocus(ginkgoconfig.GinkgoConfig.FocusStrings, common.DiagnosticTestKey) {
		ginkgo.When("a cluster is set up and installed with OpenShift", func() {
			testID := identifiers.XformToGinkgoItIdentifier(identifiers.TestExtractNodeInformationIdentifier)
			ginkgo.It(testID, func() {
				defer results.RecordResult(identifiers.TestExtractNodeInformationIdentifier)
				context := common.GetContext()

				test, handlers, jsonParseResult, err := generic.NewGenericFromJSONFile(relativeNodesTestPath, relativeSchemaPath)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(jsonParseResult).ToNot(gomega.BeNil())
				gomega.Expect(jsonParseResult.Valid()).To(gomega.BeTrue())
				gomega.Expect(handlers).ToNot(gomega.BeNil())
				gomega.Expect(test).ToNot(gomega.BeNil())

				tester, err := tnf.NewTest(context.GetExpecter(), *test, handlers, context.GetErrorChannel())
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(tester).ToNot(gomega.BeNil())

				result, err := tester.Run()
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(result).To(gomega.Equal(tnf.SUCCESS))

				genericTest := (*test).(*generic.Generic)
				gomega.Expect(genericTest).ToNot(gomega.BeNil())
				matches := genericTest.Matches
				gomega.Expect(len(matches)).To(gomega.Equal(1))
				match := genericTest.GetMatches()[0]
				err = json.Unmarshal([]byte(match.Match), &nodeSummary)
				gomega.Expect(err).To(gomega.BeNil())
			})
			ginkgo.By("should report all CNI plugins")
			testID = identifiers.XformToGinkgoItIdentifier(identifiers.TestListCniPluginsIdentifier)
			ginkgo.It(testID, func() {
				defer results.RecordResult(identifiers.TestListCniPluginsIdentifier)
				testCniPlugins()
			})
			ginkgo.By("should report nodes HW info")
			testID = identifiers.XformToGinkgoItIdentifier(identifiers.TestNodesHwInfoIdentifier)
			ginkgo.It(testID, func() {
				defer results.RecordResult(identifiers.TestNodesHwInfoIdentifier)
				testNodesHwInfo()
			})
			ginkgo.By("should report cluster CSI driver info")
			testID = identifiers.XformToGinkgoItIdentifier(identifiers.TestClusterCsiInfoIdentifier)
			ginkgo.It(testID, func() {
				defer results.RecordResult(identifiers.TestClusterCsiInfoIdentifier)
				listClusterCSIInfo()
			})
		})
	}
})

// CniPlugin holds info about a CNI plugin
type CniPlugin struct {
	Name    string
	version string
}

// NodeHwInfo node HW info
type NodeHwInfo struct {
	NodeName string
	Lscpu    map[string]string   // lscpu output parsed as entry to value map
	Ifconfig map[string][]string // ifconfig output parsed as interface name to output lines map
	Lsblk    interface{}         // 'lsblk -J' output un-marshaled into an unknown type
	Lspci    []string            // lspci output parsed to individual lines
}

// NodesHwInfo one master one worker
type NodesHwInfo struct {
	Master NodeHwInfo
	Worker NodeHwInfo
}

// GetNodeSummary returns the result of running `oc get nodes -o json`.
func GetNodeSummary() map[string]interface{} {
	return nodeSummary
}

// GetCniPlugins return the found plugins
func GetCniPlugins() []CniPlugin {
	return cniPlugins
}

// GetNodesHwInfo returns an object with HW info of one master and one worker
func GetNodesHwInfo() NodesHwInfo {
	return nodesHwInfo
}

// GetCsiDriverInfo returns the CSI driver info of running `oc get csidriver -o json`.
func GetCsiDriverInfo() map[string]interface{} {
	return csiDriver
}

func getFirstNode(labelFilter map[string]*string) string {
	context := common.GetContext()
	tester := nodenames.NewNodeNames(defaultTestTimeout, labelFilter)
	test, err := tnf.NewTest(context.GetExpecter(), tester, []reel.Handler{tester}, context.GetErrorChannel())
	gomega.Expect(err).To(gomega.BeNil())
	testResult, err := test.Run()
	gomega.Expect(testResult).To(gomega.Equal(tnf.SUCCESS))
	gomega.Expect(err).To(gomega.BeNil())
	nodeNames := tester.GetNodeNames()
	gomega.Expect(nodeNames).NotTo(gomega.BeEmpty())
	return nodeNames[0]
}

func getMasterNodeName() string {
	const masterNodeLabel = "node-role.kubernetes.io/master"
	return getFirstNode(map[string]*string{masterNodeLabel: nil})
}

func getWorkerNodeName() string {
	const workerNodeLabel = "node-role.kubernetes.io/worker"
	return getFirstNode(map[string]*string{workerNodeLabel: nil})
}

func listNodeCniPlugins(nodeName string) []CniPlugin {
	const command = "jq -r .name,.cniVersion '/etc/cni/net.d/*'"
	result := []CniPlugin{}
	context := common.GetContext()
	tester := nodedebug.NewNodeDebug(defaultTestTimeout, nodeName, command, true, true)
	test, err := tnf.NewTest(context.GetExpecter(), tester, []reel.Handler{tester}, context.GetErrorChannel())
	gomega.Expect(err).To(gomega.BeNil())
	testResult, err := test.Run()
	gomega.Expect(testResult).To(gomega.Equal(tnf.SUCCESS))
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(len(tester.Processed)%2 == 0).To(gomega.BeTrue())
	for i := 0; i < len(tester.Processed); i += 2 {
		result = append(result, CniPlugin{
			tester.Processed[i],
			tester.Processed[i+1],
		})
	}
	return result
}

func testCniPlugins() {
	if common.IsMinikube() {
		ginkgo.Skip("can't use 'oc debug' in minikube")
	}
	// get name of a master node
	nodeName := getMasterNodeName()
	gomega.Expect(nodeName).ToNot(gomega.BeEmpty())
	// get CNI plugins from node
	cniPlugins = listNodeCniPlugins(nodeName)
	gomega.Expect(cniPlugins).ToNot(gomega.BeNil())
}

func testNodesHwInfo() {
	if common.IsMinikube() {
		ginkgo.Skip("can't use 'oc debug' in minikube")
	}

	masterNodeName := getMasterNodeName()
	gomega.Expect(masterNodeName).ToNot(gomega.BeEmpty())
	workerNodeName := getWorkerNodeName()
	gomega.Expect(workerNodeName).ToNot(gomega.BeEmpty())
	nodesHwInfo.Master.NodeName = masterNodeName
	nodesHwInfo.Master.Lscpu = getNodeLscpu(masterNodeName)
	nodesHwInfo.Master.Ifconfig = getNodeIfconfig(masterNodeName)
	nodesHwInfo.Master.Lsblk = getNodeLsblk(masterNodeName)
	nodesHwInfo.Master.Lspci = getNodeLspci(masterNodeName)
	nodesHwInfo.Worker.NodeName = workerNodeName
	nodesHwInfo.Worker.Lscpu = getNodeLscpu(workerNodeName)
	nodesHwInfo.Worker.Ifconfig = getNodeIfconfig(workerNodeName)
	nodesHwInfo.Worker.Lsblk = getNodeLsblk(workerNodeName)
	nodesHwInfo.Worker.Lspci = getNodeLspci(workerNodeName)
}

func getNodeLscpu(nodeName string) map[string]string {
	const command = "lscpu"
	result := map[string]string{}
	context := common.GetContext()
	tester := nodedebug.NewNodeDebug(defaultTestTimeout, nodeName, command, true, true)
	test, err := tnf.NewTest(context.GetExpecter(), tester, []reel.Handler{tester}, context.GetErrorChannel())
	gomega.Expect(err).To(gomega.BeNil())
	testResult, err := test.Run()
	gomega.Expect(testResult).To(gomega.Equal(tnf.SUCCESS))
	gomega.Expect(err).To(gomega.BeNil())
	for _, line := range tester.Processed {
		fields := strings.SplitN(line, ":", 2)
		result[fields[0]] = strings.TrimSpace(fields[1])
	}
	return result
}

func getNodeIfconfig(nodeName string) map[string][]string {
	const command = "ifconfig"
	result := map[string][]string{}
	context := common.GetContext()
	tester := nodedebug.NewNodeDebug(defaultTestTimeout, nodeName, command, true, true)
	test, err := tnf.NewTest(context.GetExpecter(), tester, []reel.Handler{tester}, context.GetErrorChannel())
	gomega.Expect(err).To(gomega.BeNil())
	testResult, err := test.Run()
	gomega.Expect(testResult).To(gomega.Equal(tnf.SUCCESS))
	gomega.Expect(err).To(gomega.BeNil())
	deviceName := ""
	for _, line := range tester.Processed {
		if line == "" {
			continue
		}
		if line[0] != ' ' {
			fields := strings.SplitN(line, ":", 2)
			deviceName = fields[0]
			line = fields[1]
		}
		result[deviceName] = append(result[deviceName], strings.TrimSpace(line))
	}
	return result
}

func getNodeLsblk(nodeName string) interface{} {
	const command = "lsblk -J"
	context := common.GetContext()
	tester := nodedebug.NewNodeDebug(defaultTestTimeout, nodeName, command, false, false)
	test, err := tnf.NewTest(context.GetExpecter(), tester, []reel.Handler{tester}, context.GetErrorChannel())
	gomega.Expect(err).To(gomega.BeNil())
	testResult, err := test.Run()
	gomega.Expect(testResult).To(gomega.Equal(tnf.SUCCESS))
	gomega.Expect(err).To(gomega.BeNil())
	result := map[string]interface{}{}
	err = json.Unmarshal([]byte(tester.Raw), &result)
	gomega.Expect(err).To(gomega.BeNil())
	return result
}

func getNodeLspci(nodeName string) []string {
	const command = "lspci"
	context := common.GetContext()
	tester := nodedebug.NewNodeDebug(defaultTestTimeout, nodeName, command, true, true)
	test, err := tnf.NewTest(context.GetExpecter(), tester, []reel.Handler{tester}, context.GetErrorChannel())
	gomega.Expect(err).To(gomega.BeNil())
	testResult, err := test.Run()
	gomega.Expect(testResult).To(gomega.Equal(tnf.SUCCESS))
	gomega.Expect(err).To(gomega.BeNil())
	return tester.Processed
}

// check CSI driver info in cluster
func listClusterCSIInfo() {
	if common.IsMinikube() {
		ginkgo.Skip("CSI is not checked in minikube")
	}
	context := common.GetContext()
	test, handlers, result, err := generic.NewGenericFromJSONFile(relativeCsiDriverTestPath, common.RelativeSchemaPath)
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(result).ToNot(gomega.BeNil())
	gomega.Expect(result.Valid()).To(gomega.BeTrue())
	gomega.Expect(handlers).ToNot(gomega.BeNil())
	gomega.Expect(len(handlers)).To(gomega.Equal(1))
	gomega.Expect(test).ToNot(gomega.BeNil())
	tester, err := tnf.NewTest(context.GetExpecter(), *test, handlers, context.GetErrorChannel())
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(tester).ToNot(gomega.BeNil())
	testResult, err := tester.Run()
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(testResult).To(gomega.Equal(tnf.SUCCESS))
	genericTest := (*test).(*generic.Generic)
	gomega.Expect(genericTest).ToNot(gomega.BeNil())
	matches := genericTest.Matches
	gomega.Expect(len(matches)).To(gomega.Equal(1))
	match := genericTest.GetMatches()[0]
	err = json.Unmarshal([]byte(match.Match), &csiDriver)
	gomega.Expect(err).To(gomega.BeNil())
}
