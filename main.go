package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"
	klog "k8s.io/klog/v2"

	//"k8s.io/client-go/pkg/api/v1"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/logs"
)

var kubeconfig string
var namespace string
var interval int
var restartSecondsKey = "k8srestart.stenic.io/restartAfter"
var lastRestartKey = "k8srestart.stenic.io/lastRestart"

func init() {
	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&namespace, "namespace", "", "namespace")
	flag.IntVar(&interval, "interval", 30, "readiness poll interval")
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	logs.InitLogs()
	defer logs.FlushLogs()

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			klog.V(1).Info("Checking")
			now := time.Now()
			checkPods(clientset, namespace, now)
			checkDeployments(clientset, namespace, now)
		case <-quit:
			ticker.Stop()
			return
		}
	}

}

func checkPods(clientset *kubernetes.Clientset, ns string, now time.Time) {
	pods, err := clientset.CoreV1().Pods(ns).List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, pod := range pods.Items {
		if v1.HasAnnotation(pod.ObjectMeta, restartSecondsKey) {
			seconds := pod.ObjectMeta.Annotations[restartSecondsKey]
			secondsInt, _ := strconv.ParseInt(seconds, 10, 64)
			before := v1.NewTime(now.Add(-time.Duration(secondsInt) * time.Second))
			if pod.Status.StartTime.Before(&before) {
				klog.Infof(
					"Pod %s/%s running longer than %d seconds",
					pod.Namespace,
					pod.Name,
					secondsInt,
				)
				deletePolicy := v1.DeletePropagationForeground
				if err := clientset.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, v1.DeleteOptions{
					PropagationPolicy: &deletePolicy,
				}); err != nil {
					klog.Error(err)
				}
			}
		}
	}
}

func checkDeployments(clientset *kubernetes.Clientset, ns string, now time.Time) {
	deploys, err := clientset.AppsV1().Deployments(ns).List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, deploy := range deploys.Items {
		if v1.HasAnnotation(deploy.ObjectMeta, restartSecondsKey) {

			seconds := deploy.ObjectMeta.Annotations[restartSecondsKey]
			secondsInt, _ := strconv.ParseInt(seconds, 10, 64)
			before := now.Add(-time.Duration(secondsInt) * time.Second)

			last, dateParseErr := time.Parse(time.RFC822, deploy.ObjectMeta.Annotations[lastRestartKey])
			if dateParseErr != nil || last.Before(before) {
				klog.Infof(
					"Deployment %s/%s running longer than %d seconds",
					deploy.Namespace,
					deploy.Name,
					secondsInt,
				)
				data := fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}},"spec":{"template":{"metadata":{"annotations":{"%s":"%s"}}}}}`, lastRestartKey, time.Now().Format(time.RFC822), lastRestartKey, time.Now().Format(time.RFC822))
				_, err = clientset.AppsV1().Deployments(deploy.Namespace).Patch(
					context.Background(),
					deploy.Name,
					types.StrategicMergePatchType,
					[]byte(data),
					v1.PatchOptions{FieldManager: "kubectl-rollout"},
				)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	}
}
