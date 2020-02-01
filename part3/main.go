package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	portscanner "github.com/anvie/port-scanner"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	namespace   = "default"
	serviceName = "helloworld"
)

func main() {

	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "InCluster", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	var config *rest.Config
	var err error

	log.Println("Getting config")
	if *kubeconfig == "InCluster" {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Printf("Error while getting InCluster config: %s", err.Error())
			os.Exit(1)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Printf("Error while getting Out-of-Cluster config: %s", err.Error())
			os.Exit(1)
		}
	}

	log.Println("Creating K8s clientset")

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Error while creating clientset: %s", err.Error())
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Printf("Could not find services: %s", err.Error())
	}

	for _, pod := range podList.Items {
		log.Printf("Pod: %s - IP to scan: %s", pod.ObjectMeta.Name, pod.Status.PodIP)

		// scan Pod with a 2 second timeout per port in 5 concurrent threads
		ps := portscanner.NewPortScanner(pod.Status.PodIP, 2*time.Second, 5)

		// get opened port
		log.Printf("scanning port %d-%d...\n", 8000, 10000)

		openedPorts := ps.GetOpenedPort(8000, 10000)

		for i := 0; i < len(openedPorts); i++ {
			port := openedPorts[i]
			log.Print(" ", port, " [open]")
			log.Println("  -->  ", ps.DescribePort(port))
		}
	}

}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
