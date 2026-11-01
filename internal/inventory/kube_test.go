package inventory

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pod(phase corev1.PodPhase, cs ...corev1.ContainerStatus) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: phase, ContainerStatuses: cs},
	}
}

func TestPodRunning(t *testing.T) {
	it := podItem(pod(corev1.PodRunning, corev1.ContainerStatus{Ready: true}))
	if it.Status != StatusOK || !reflect.DeepEqual(it.Actions, []string{"delete"}) || !it.HasLogs ||
		it.ID != "k8s:pod:default/web-1" || it.Namespace != "default" || it.Kind != KindPod {
		t.Fatalf("%+v", it)
	}
}

func TestPodCrashLoop(t *testing.T) {
	it := podItem(pod(corev1.PodRunning, corev1.ContainerStatus{RestartCount: 4,
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}}))
	if it.Status != StatusWarn || it.Detail["restarts"] != "4" {
		t.Fatalf("%+v", it)
	}
	if it := podItem(pod(corev1.PodRunning, corev1.ContainerStatus{Ready: true, RestartCount: 1})); it.Status != StatusWarn {
		t.Fatalf("restarts>0 should warn: %+v", it)
	}
}

func TestPodFailed(t *testing.T) {
	for _, ph := range []corev1.PodPhase{corev1.PodFailed, corev1.PodSucceeded} {
		if it := podItem(pod(ph)); it.Status != StatusDown {
			t.Fatalf("%s: %+v", ph, it)
		}
	}
}

func TestSystemNamespacesHidden(t *testing.T) {
	for _, ns := range []string{"kube-system", "kube-public", "kube-node-lease", "local-path-storage"} {
		p := pod(corev1.PodRunning)
		p.Namespace = ns
		if !podItem(p).Hidden || !IsSystemNamespace(ns) {
			t.Fatalf("%s should be hidden", ns)
		}
	}
	if podItem(pod(corev1.PodRunning)).Hidden {
		t.Fatal("default should be visible")
	}
}

func TestDeployReplicas(t *testing.T) {
	two := int32(2)
	d := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{Replicas: &two}, Status: appsv1.DeploymentStatus{AvailableReplicas: 1}}
	if it := deployItem(d); it.Status != StatusWarn || len(it.Actions) != 0 || it.ID != "k8s:deploy:default/api" {
		t.Fatalf("%+v", it)
	}
	d.Status.AvailableReplicas = 2
	if it := deployItem(d); it.Status != StatusOK {
		t.Fatalf("%+v", it)
	}
}

func TestServicePorts(t *testing.T) {
	s := corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
		Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80, NodePort: 30080}, {Port: 443}}}}
	if it := svcItem(s); !reflect.DeepEqual(it.Ports, []int{443, 30080}) || it.Kind != KindService {
		t.Fatalf("%+v", it)
	}
}
