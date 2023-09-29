/*
Copyright 2018 The Kubernetes Authors.

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

package e2e

import (
	"fmt"
	ebscsidriver "github.com/kubernetes-sigs/aws-ebs-csi-driver/pkg/driver"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/tests/e2e/driver"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/tests/e2e/testsuites"
	. "github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
)

const (
	blockSizeTestValue      = "1024"
	inodeSizeTestValue      = "512"
	bytesPerInodeTestValue  = "8192"
	numberOfInodesTestValue = "200192"

	expectedBytesPerInodeTestResult = "131072" // TODO having this here is code smell. Hardcode? Test case from original inode PR #1661 https://github.com/kubernetes-sigs/aws-ebs-csi-driver/pull/1661
)

var (
	formatOptionTests = []testsuites.FormatOptionTest{
		{
			CreateVolumeParameterKey:        ebscsidriver.BlockSizeKey,
			CreateVolumeParameterValue:      blockSizeTestValue,
			ExpectedFilesystemInfoParamName: "Block size",
			ExpectedFilesystemInfoParamVal:  blockSizeTestValue,
		},
		{
			CreateVolumeParameterKey:        ebscsidriver.INodeSizeKey,
			CreateVolumeParameterValue:      inodeSizeTestValue,
			ExpectedFilesystemInfoParamName: "Inode size",
			ExpectedFilesystemInfoParamVal:  inodeSizeTestValue},
		{
			CreateVolumeParameterKey:        ebscsidriver.BytesPerINodeKey,
			CreateVolumeParameterValue:      bytesPerInodeTestValue,
			ExpectedFilesystemInfoParamName: "Inode count",
			ExpectedFilesystemInfoParamVal:  expectedBytesPerInodeTestResult,
		},
		{
			CreateVolumeParameterKey:        ebscsidriver.NumberOfINodesKey,
			CreateVolumeParameterValue:      numberOfInodesTestValue,
			ExpectedFilesystemInfoParamName: "Inode count",
			ExpectedFilesystemInfoParamVal:  numberOfInodesTestValue,
		},
	}
)

var _ = Describe("[ebs-csi-e2e] [single-az] [format-options] Formatting a volume", func() {
	f := framework.NewDefaultFramework("ebs")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged // TODO Maybe don't need this if Connor big brain pulls thru

	var (
		cs        clientset.Interface
		ns        *v1.Namespace
		ebsDriver driver.PVTestDriver

		testedFsTypes = []string{ebscsidriver.FSTypeExt4} // TODO Is this right place for this?
	)

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		ebsDriver = driver.InitEbsCSIDriver()
	})

	for _, fsType := range testedFsTypes {
		Context(fmt.Sprintf("with an %s filesystem", fsType), func() {
			for _, formatOptionTestCase := range formatOptionTests {
				formatOptionTestCase := formatOptionTestCase // Go trap
				if fsTypeDoesNotSupportFormatOptionParameter(fsType, formatOptionTestCase.CreateVolumeParameterKey) {
					continue
				}

				Context(fmt.Sprintf("with a custom %s parameter", formatOptionTestCase.CreateVolumeParameterKey), func() {
					It("successfully mounts and is resizable", func() {
						formatOptionTestCase.Run(cs, ns, ebsDriver, fsType)
					})
				})
			}
		})
	}
})

func fsTypeDoesNotSupportFormatOptionParameter(fsType string, createVolumeParameterKey string) bool {
	_, paramNotSupported := ebscsidriver.FileSystemConfigs[fsType].NotSupportedParams[createVolumeParameterKey]
	return paramNotSupported
}
