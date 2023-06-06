package util

import (
	"reflect"
	"testing"

	api "github.com/ray-project/kuberay/proto/go_client"
	v1 "k8s.io/api/core/v1"
)

var testVolume = &api.Volume{
	Name:       "hdfs",
	VolumeType: api.Volume_HOST_PATH,
	Source:     "/opt/hdfs",
	MountPath:  "/mnt/hdfs",
	ReadOnly:   true,
}

// There is only an fake case for test both MountPropagationMode and file type
// in real case hostToContainer mode may only valid for directory
var testFileVolume = &api.Volume{
	Name:                 "test-file",
	VolumeType:           api.Volume_HOST_PATH,
	MountPropagationMode: api.Volume_HOSTTOCONTAINER,
	Source:               "/root/proc/stat",
	MountPath:            "/proc/stat",
	HostPathType:         api.Volume_FILE,
	ReadOnly:             true,
}

var testPVCVolume = &api.Volume{
	Name:       "test-pvc",
	VolumeType: api.Volume_PERSISTENT_VOLUME_CLAIM,
	MountPath:  "/pvc/dir",
	ReadOnly:   true,
}

// Spec for testing
var headGroup = api.HeadGroupSpec{
	ComputeTemplate: "foo",
	Image:           "bar",
	ServiceType:     "ClusterIP",
	RayStartParams: map[string]string{
		"dashboard-host":      "0.0.0.0",
		"metrics-export-port": "8080",
		"num-cpus":            "0",
	},
	Environment: map[string]string{
		"foo": "bar",
	},
	Annotations: map[string]string{
		"foo": "bar",
	},
	Labels: map[string]string{
		"foo": "bar",
	},
}

var workerGroup = api.WorkerGroupSpec{
	GroupName:       "wg",
	ComputeTemplate: "foo",
	Image:           "bar",
	Replicas:        5,
	MinReplicas:     5,
	MaxReplicas:     5,
	RayStartParams: map[string]string{
		"node-ip-address": "$MY_POD_IP",
	},
	Environment: map[string]string{
		"foo": "bar",
	},
	Annotations: map[string]string{
		"foo": "bar",
	},
	Labels: map[string]string{
		"foo": "bar",
	},
}

var template = api.ComputeTemplate{
	Name:      "",
	Namespace: "",
	Cpu:       2,
	Memory:    8,
	Tolerations: []*api.PodToleration{
		{
			Key:      "blah1",
			Operator: "Exists",
			Effect:   "NoExecute",
		},
	},
}

var expectedToleration = v1.Toleration{
	Key:      "blah1",
	Operator: "Exists",
	Effect:   "NoExecute",
}

var expectedLabels = map[string]string{
	"foo": "bar",
}

func TestBuildVolumes(t *testing.T) {
	targetVolume := v1.Volume{
		Name: testVolume.Name,
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: testVolume.Source,
				Type: newHostPathType(string(v1.HostPathDirectory)),
			},
		},
	}
	targetFileVolume := v1.Volume{
		Name: testFileVolume.Name,
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: testFileVolume.Source,
				Type: newHostPathType(string(v1.HostPathFile)),
			},
		},
	}

	targetPVCVolume := v1.Volume{
		Name: testPVCVolume.Name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: testPVCVolume.Name,
				ReadOnly:  testPVCVolume.ReadOnly,
			},
		},
	}
	tests := []struct {
		name      string
		apiVolume []*api.Volume
		expect    []v1.Volume
	}{
		{
			"normal test",
			[]*api.Volume{
				testVolume, testFileVolume,
			},
			[]v1.Volume{targetVolume, targetFileVolume},
		},
		{
			"pvc test",
			[]*api.Volume{testPVCVolume},
			[]v1.Volume{targetPVCVolume},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildVols(tt.apiVolume)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("failed for %s ..., got %v, expected %v", tt.name, got, tt.expect)
			}
		})
	}
}

func TestBuildVolumeMounts(t *testing.T) {
	hostToContainer := v1.MountPropagationHostToContainer
	targetVolumeMount := v1.VolumeMount{
		Name:      testVolume.Name,
		ReadOnly:  testVolume.ReadOnly,
		MountPath: testVolume.MountPath,
	}
	targetFileVolumeMount := v1.VolumeMount{
		Name:             testFileVolume.Name,
		ReadOnly:         testFileVolume.ReadOnly,
		MountPath:        testFileVolume.MountPath,
		MountPropagation: &hostToContainer,
	}
	targetPVCVolumeMount := v1.VolumeMount{
		Name:      testPVCVolume.Name,
		ReadOnly:  testPVCVolume.ReadOnly,
		MountPath: testPVCVolume.MountPath,
	}
	tests := []struct {
		name      string
		apiVolume []*api.Volume
		expect    []v1.VolumeMount
	}{
		{
			"normal test",
			[]*api.Volume{
				testVolume,
				testFileVolume,
			},
			[]v1.VolumeMount{
				targetVolumeMount,
				targetFileVolumeMount,
			},
		},
		{
			"pvc test",
			[]*api.Volume{testPVCVolume},
			[]v1.VolumeMount{targetPVCVolumeMount},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildVolumeMounts(tt.apiVolume)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("failed for %s ..., got %v, expected %v", tt.name, got, tt.expect)
			}
		})
	}
}

func TestBuildHeadPodTemplate(t *testing.T) {
	podSpec := buildHeadPodTemplate("2.4", make(map[string]string), &headGroup, &template)
	if !containsEnv(podSpec.Spec.Containers[0].Env, "foo", "bar") {
		t.Errorf("failed to propagate environment")
	}
	if len(podSpec.Spec.Tolerations) != 1 {
		t.Errorf("failed to propagate tolerations, expected 1, got %d", len(podSpec.Spec.Tolerations))
	}
	if !reflect.DeepEqual(podSpec.Spec.Tolerations[0], expectedToleration) {
		t.Errorf("failed to propagate annotations, got %v, expected %v", tolerationToString(&podSpec.Spec.Tolerations[0]),
			tolerationToString(&expectedToleration))
	}
	if val, exists := podSpec.Annotations["foo"]; !exists || val != "bar" {
		t.Errorf("failed to convert annotations")
	}
	if !reflect.DeepEqual(podSpec.Labels, expectedLabels) {
		t.Errorf("failed to convert labels, got %v, expected %v", podSpec.Labels, expectedLabels)
	}
}

func TestBuilWorkerPodTemplate(t *testing.T) {
	podSpec := buildWorkerPodTemplate("2.4", make(map[string]string), &workerGroup, &template)
	if !containsEnv(podSpec.Spec.Containers[0].Env, "foo", "bar") {
		t.Errorf("failed to propagate environment")
	}
	if len(podSpec.Spec.Tolerations) != 1 {
		t.Errorf("failed to propagate tolerations, expected 1, got %d", len(podSpec.Spec.Tolerations))
	}
	if !reflect.DeepEqual(podSpec.Spec.Tolerations[0], expectedToleration) {
		t.Errorf("failed to propagate annotations, got %v, expected %v", tolerationToString(&podSpec.Spec.Tolerations[0]),
			tolerationToString(&expectedToleration))
	}
	if val, exists := podSpec.Annotations["foo"]; !exists || val != "bar" {
		t.Errorf("failed to convert annotations")
	}
	if !reflect.DeepEqual(podSpec.Labels, expectedLabels) {
		t.Errorf("failed to convert labels, got %v, expected %v", podSpec.Labels, expectedLabels)
	}
}

func containsEnv(envs []v1.EnvVar, key string, val string) bool {
	for _, env := range envs {
		if env.Name == key && env.Value == val {
			return true
		}
	}
	return false
}

func tolerationToString(toleration *v1.Toleration) string {
	return "Key: " + toleration.Key + " Operator: " + string(toleration.Operator) + " Effect: " + string(toleration.Effect)
}
