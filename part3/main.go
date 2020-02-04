package main

import (
	"flag"
	"fmt"
	"strconv"
	"log"
	"os"
	"path/filepath"
	"time"

	portscanner "github.com/anvie/port-scanner"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")

var (
	portScansExecuted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "portscanner_scans_executed_total",
			Help: "The amount of total port scans executed",
		},
		[]string{"scanned_pod", "scanned_namespace"},
	)

	openHighPorts = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "portscanner_open_high_ports",
			Help: "The amount of currently detected open high ports",
		},
		[]string{"scanned_pod", "scanned_namespace", "scanned_port"},
	)

	openLowPorts = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "portscanner_open_low_ports",
			Help: "The amount of currently detected open low ports",
		},
		[]string{"scanned_pod", "scanned_namespace", "scanned_port"},
	)
)

func init() {
	prometheus.MustRegister(portScansExecuted)
	prometheus.MustRegister(openHighPorts)
	prometheus.MustRegister(openLowPorts)
}

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

	go func() {
		for {

			namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
			if err != nil {
				log.Printf("Could not find namespaces: %s", err.Error())
			}

			var podList []v1.Pod

			for _, namespace := range namespaces.Items {
				pods, err := clientset.CoreV1().Pods(namespace.ObjectMeta.Name).List(metav1.ListOptions{})
				if err != nil {
					log.Printf("Could not find pods: %s", err.Error())
				}
				podList = append(podList, pods.Items...)
			}

			for _, pod := range podList {

				log.Printf("Pod: %s in Namespace: %s - IP to scan: %s", pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, pod.Status.PodIP)

				// scan Pod with a 50 millisecond timeout per port in 10 concurrent threads
				// only if the Pod doesn't use hostNetwork: true
				if pod.Spec.HostNetwork {
					log.Printf("Not scanning Pod %s because it uses hostNetwork", pod.ObjectMeta.Name)
					continue
				}
				ps := portscanner.NewPortScanner(pod.Status.PodIP, 50*time.Millisecond, 100)

				// get opened port
				log.Printf("scanning port %d-%d...\n", 0, 65535)

				openedPorts := ps.GetOpenedPort(0, 65535)

				portScansExecuted.With(prometheus.Labels{"scanned_pod": pod.ObjectMeta.Name, "scanned_namespace": pod.ObjectMeta.Namespace}).Inc()

				for i := 0; i < len(openedPorts); i++ {
					port := openedPorts[i]
					if port < 1024 {
						openLowPorts.With(prometheus.Labels{"scanned_pod": pod.ObjectMeta.Name, "scanned_namespace": pod.ObjectMeta.Namespace, "scanned_port": strconv.Itoa(port)}).Set(float64(1))
					} else {
						openHighPorts.With(prometheus.Labels{"scanned_pod": pod.ObjectMeta.Name, "scanned_namespace": pod.ObjectMeta.Namespace, "scanned_port": strconv.Itoa(port)}).Set(float64(1))
					}
					log.Print(" ", port, " [open]")
				}

			}
			time.Sleep(300 * time.Second)
		}

	}()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "app is alive")
	})

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))
	log.Fatal(http.ListenAndServe(*addr, nil))

}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
