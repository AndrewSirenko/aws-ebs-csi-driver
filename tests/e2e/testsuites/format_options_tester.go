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

package testsuites

import (
	"fmt"
	awscloud "github.com/kubernetes-sigs/aws-ebs-csi-driver/pkg/cloud"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/tests/e2e/driver"
	"github.com/onsi/gomega/format"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
)

// FormatOptionTest will provision required StorageClass(es), PVC(s) and Pod(s) TODO
// Waiting for the PV provisioner to create a new PV
// Update pvc storage size
// Waiting for new PVC and PV to be ready
// And finally attach pvc to the pod and wait for pod to be ready.
type FormatOptionTest struct {
	CreateVolumeParameterKey        string
	CreateVolumeParameterValue      string
	ExpectedFilesystemInfoParamName string
	ExpectedFilesystemInfoParamVal  string
}

const (
	volumeSizeIncreaseAmtGi = 1
	volumeMountPath         = "/mnt/test-format-option" // TODO should I keep this as mnt/test-1, and refactor to be `DefaultMountPath` globally in testsuites?
)

var (
	podCmdGetFsInfo     = fmt.Sprintf("tune2fs -l $(df -k '%s'| tail -1 | awk '{ print $1 }')", volumeMountPath)                               // Gets the filesystem info for the mounted volume
	podCmdWriteToVolume = fmt.Sprintf("echo 'hello world' >> %s/data && grep 'hello world' %s/data && sync", volumeMountPath, volumeMountPath) // TODO Debt: All the dynamic provisioning tests use this same cmd. Should we refactor out into exported constant?

)

func (t *FormatOptionTest) Run(client clientset.Interface, namespace *v1.Namespace, ebsDriver driver.PVTestDriver, fsType string) {
	By("setting up pvc")
	volumeDetails := createFormatOptionVolumeDetails(fsType, volumeMountPath, t)
	testPvc, _ := volumeDetails.SetupDynamicPersistentVolumeClaim(client, namespace, ebsDriver)
	defer testPvc.Cleanup()

	By("deploying pod with custom format option")
	getFsInfoTestPod := createPodWithVolume(client, namespace, podCmdGetFsInfo, testPvc, volumeDetails)
	defer getFsInfoTestPod.Cleanup()
	getFsInfoTestPod.WaitForSuccess() // TODO e2e test implementation defaults to a 15 min wait instead of 5 min one... Is that fine or refactor worthy?

	By("confirming custom format option was applied")
	fsInfoSearchRegexp := fmt.Sprintf(`%s:\s+%s`, t.ExpectedFilesystemInfoParamName, t.ExpectedFilesystemInfoParamVal)
	if isFormatOptionApplied := FindRegexpInPodLogs(fsInfoSearchRegexp, getFsInfoTestPod); !isFormatOptionApplied {
		framework.Failf("Did not find expected %s value of %s in filesystem info", t.ExpectedFilesystemInfoParamName, t.ExpectedFilesystemInfoParamVal)
	}

	By("testing that pvc is able to be resized")
	ResizeTestPvc(client, namespace, testPvc, volumeSizeIncreaseAmtGi)

	By("validating resized pvc by deploying new pod")
	resizeTestPod := createPodWithVolume(client, namespace, podCmdWriteToVolume, testPvc, volumeDetails)
	defer resizeTestPod.Cleanup()

	By("confirming new pod can write to resized volume")
	resizeTestPod.WaitForSuccess()
}

// TODO should we improve this across e2e tests via builder design pattern? Or is that not go-like?
func createFormatOptionVolumeDetails(fsType string, volumeMountPath string, t *FormatOptionTest) *VolumeDetails {
	allowVolumeExpansion := true

	volume := VolumeDetails{
		VolumeType:   awscloud.VolumeTypeGP2,
		FSType:       fsType,
		MountOptions: []string{"rw"},
		ClaimSize:    driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
		VolumeMount: VolumeMountDetails{
			NameGenerate:      "test-volume-format-option",
			MountPathGenerate: volumeMountPath,
		},
		AllowVolumeExpansion: &allowVolumeExpansion,
		AdditionalParameters: map[string]string{
			t.CreateVolumeParameterKey: t.CreateVolumeParameterValue,
		},
	}

	return &volume
}

// TODO putting this in function may be overkill? In an ideal world we refactor out TestEverything objects so testPod.SetupVolume isn't gross.
func createPodWithVolume(client clientset.Interface, namespace *v1.Namespace, cmd string, testPvc *TestPersistentVolumeClaim, volumeDetails *VolumeDetails) *TestPod {
	testPod := NewTestPod(client, namespace, cmd)
	testPod.SetupVolume(testPvc.persistentVolumeClaim, volumeDetails.VolumeMount.NameGenerate, volumeDetails.VolumeMount.MountPathGenerate, volumeDetails.VolumeMount.ReadOnly)
	testPod.Create()

	return testPod
}

// TODO should I move this to testsuites.go ?

// FindRegexpInPodLogs searches given testPod's logs for a given regular expression. Returns `true` if found.
func FindRegexpInPodLogs(regexpPattern string, testPod *TestPod) bool {
	By(fmt.Sprintf("Searching for matching regexp '%s' in logs of pod", regexpPattern))
	podLogs, err := testPod.Logs()
	framework.ExpectNoError(err, "tried getting logs for pod %s", format.Object(testPod, 2))

	var expectedLine = regexp.MustCompile(regexpPattern)

	res := expectedLine.Find(podLogs)
	framework.Logf("result of regexp search through pod logs: '%s'", string(res))
	return res != nil
}
