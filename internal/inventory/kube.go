package inventory

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SamuelSchutz13/orion-logger/internal/logs"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeSource lê o contexto atual do kubeconfig (sem precisar do kubectl).
type KubeSource struct {
	mu  sync.Mutex
	cs  *kubernetes.Clientset
	key string // contexto + servidor do cliente atual
	ctx string
}

func NewKubeSource() *KubeSource { return &KubeSource{} }

func (s *KubeSource) Name() string { return "k8s" }

// client relê o kubeconfig a cada chamada (é só um arquivo local) e recria o cliente
// quando o contexto ou o servidor mudam, como depois de `kubectl config use-context`.
func (s *KubeSource) client() (*kubernetes.Clientset, error) {
	raw, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, err
	}
	if len(raw.Contexts) == 0 {
		return nil, errors.New("sem kubeconfig")
	}
	key := raw.CurrentContext
	if c := raw.Contexts[raw.CurrentContext]; c != nil {
		if cl := raw.Clusters[c.Cluster]; cl != nil {
			key += "|" + cl.Server
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cs != nil && s.key == key {
		return s.cs, nil
	}
	cfg, err := clientcmd.NewDefaultClientConfig(*raw, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}
	cfg.Timeout = 3 * time.Second
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	s.cs, s.key, s.ctx = cs, key, raw.CurrentContext
	return cs, nil
}

func (s *KubeSource) List(ctx context.Context) ([]Item, error) {
	cs, err := s.client()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	all := metav1.ListOptions{}
	pods, err := cs.CoreV1().Pods("").List(ctx, all)
	if err != nil {
		return nil, shortKubeErr(err)
	}
	svcs, err := cs.CoreV1().Services("").List(ctx, all)
	if err != nil {
		return nil, shortKubeErr(err)
	}
	deps, err := cs.AppsV1().Deployments("").List(ctx, all)
	if err != nil {
		return nil, shortKubeErr(err)
	}
	items := make([]Item, 0, len(pods.Items)+len(svcs.Items)+len(deps.Items))
	for _, p := range pods.Items {
		items = append(items, podItem(p))
	}
	for _, sv := range svcs.Items {
		items = append(items, svcItem(sv))
	}
	for _, d := range deps.Items {
		items = append(items, deployItem(d))
	}
	s.mu.Lock()
	ctxName := s.ctx
	s.mu.Unlock()
	for i := range items {
		items[i].Detail["context"] = ctxName
	}
	return items, nil
}

func shortKubeErr(err error) error {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i > 0 && len(msg) > 120 {
		msg = msg[i+2:]
	}
	return fmt.Errorf("cluster inacessível: %s", msg)
}

var systemNamespaces = map[string]bool{"kube-system": true, "kube-public": true, "kube-node-lease": true, "local-path-storage": true}

func IsSystemNamespace(ns string) bool { return systemNamespaces[ns] }

func podItem(p corev1.Pod) Item {
	restarts := int32(0)
	problem := ""
	ready := len(p.Status.ContainerStatuses) > 0
	for _, c := range p.Status.ContainerStatuses {
		restarts += c.RestartCount
		ready = ready && c.Ready
		if c.State.Waiting != nil && c.State.Waiting.Reason != "" && problem == "" {
			problem = c.State.Waiting.Reason
		}
	}
	it := Item{
		ID: "k8s:pod:" + p.Namespace + "/" + p.Name, Kind: KindPod, Name: p.Name, Namespace: p.Namespace,
		Ports: []int{}, HasLogs: true, Actions: []string{"delete"}, Hidden: IsSystemNamespace(p.Namespace),
		Detail: map[string]string{"phase": string(p.Status.Phase), "restarts": strconv.Itoa(int(restarts)), "node": p.Spec.NodeName},
	}
	if problem != "" {
		it.Detail["reason"] = problem
	}
	switch {
	case p.Status.Phase == corev1.PodFailed || p.Status.Phase == corev1.PodSucceeded:
		it.Status = StatusDown
	case p.Status.Phase == corev1.PodRunning && ready && restarts == 0 && problem == "" && p.DeletionTimestamp == nil:
		it.Status = StatusOK
	default:
		it.Status = StatusWarn
	}
	return it
}

func svcItem(s corev1.Service) Item {
	ports := []int{}
	for _, p := range s.Spec.Ports {
		n := int(p.Port)
		if p.NodePort != 0 {
			n = int(p.NodePort)
		}
		if !slices.Contains(ports, n) {
			ports = append(ports, n)
		}
	}
	slices.Sort(ports)
	return Item{
		ID: "k8s:svc:" + s.Namespace + "/" + s.Name, Kind: KindService, Name: s.Name, Namespace: s.Namespace,
		Status: StatusOK, Ports: ports, Actions: []string{}, Hidden: IsSystemNamespace(s.Namespace),
		Detail: map[string]string{"type": string(s.Spec.Type), "clusterIP": s.Spec.ClusterIP},
	}
}

func deployItem(d appsv1.Deployment) Item {
	want := int32(1)
	if d.Spec.Replicas != nil {
		want = *d.Spec.Replicas
	}
	st := StatusOK
	if d.Status.AvailableReplicas < want {
		st = StatusWarn
	}
	return Item{
		ID: "k8s:deploy:" + d.Namespace + "/" + d.Name, Kind: KindDeploy, Name: d.Name, Namespace: d.Namespace,
		Status: st, Ports: []int{}, Actions: []string{}, Hidden: IsSystemNamespace(d.Namespace),
		Detail: map[string]string{"replicas": fmt.Sprintf("%d/%d", d.Status.AvailableReplicas, want)},
	}
}

func (s *KubeSource) DeletePod(ctx context.Context, ns, name string) error {
	cs, err := s.client()
	if err != nil {
		return err
	}
	return cs.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// StreamLogs equivale a `kubectl logs -f --tail 500`; com vários containers, prefixa cada linha.
func (s *KubeSource) StreamLogs(ctx context.Context, ns, name string, emit logs.Emit) error {
	cs, err := s.client()
	if err != nil {
		return err
	}
	p, err := cs.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	tail := int64(500)
	var wg sync.WaitGroup
	errs := make(chan error, len(p.Spec.Containers))
	for _, c := range p.Spec.Containers {
		e := emit
		if len(p.Spec.Containers) > 1 {
			prefix := "[" + c.Name + "] "
			e = func(stream, text string) { emit(stream, prefix+text) }
		}
		wg.Add(1)
		go func(container string) {
			defer wg.Done()
			rc, err := cs.CoreV1().Pods(ns).GetLogs(name, &corev1.PodLogOptions{Container: container, Follow: true, TailLines: &tail}).Stream(ctx)
			if err != nil {
				errs <- err
				return
			}
			defer rc.Close()
			w := logs.LineWriter("out", e)
			defer w.Close()
			_, err = io.Copy(w, rc)
			errs <- err
		}(c.Name)
	}
	wg.Wait()
	close(errs)
	return <-errs
}
