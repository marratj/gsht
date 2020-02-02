# Hiring Task

## Part 1

There is a default 'restricted' PodSecurityPolicy, which prevented the 'helloworld' Pods from starting as the PSP states it MustRunAsNonRoot, however there was no Security Context defined in the Pod Template 
 
-> configured securityContext.runAsUser: 2 (which is the user 'daemon' in the alpine image the helloworld image is based off / Dockerfile found via Docker Hub) 
 
Second, the Ingress configuration was configured to use a wrong host (not the wildcard DNS entry I was provided with). Also, the Service Port was misconfigured in the Ingress Path, instead of 8080 it was configured as 8081. 
 
This didn’t fix the issue, so I looked at the configured Service. This didn't have any endpoints. The label  selector had a typo, writing 'helloworld' with only one 'L' instead of two.

Fixed manifests under [part1](part1).

## Part 2

Used kube-prometheus manifests as template. 
 
Prometheus StatefulSet errors because of restricted PSP which is not allowing emptyDir volumes 
--> Added additional prometheus PSP (used prometheus-operator helm chart as template) 
 
"Services in the cluster should be scraped for metrics" 
-> I was not entirely sure if ALL the services were meant with this, so I didn't run a full kube-prometheus installation that also installs kube-state-metrics and node-exporters, but instead only a "barebones" prometheus-operator and Prometheus with ServiceMonitors scraping the monitoring namespace and the default namespace. 
 
I also modified the helloworld app to be scraped, however this doesn't have a /metrics endpoint, so it's showing as DOWN, but at least it's shown in Prometheus' service discovery.

Required manifests under [part2/manifests](part2/manifests), mostly taken from [coreos/kube-prometheus](https://github.com/coreos/kube-prometheus), however without addons like alertmanager and grafana (will be installed separately).

## Part 3

Code and Helm Chart under [part3](part3)

I used Helm 3 for installation so I won't need tiller.

```bash
# Install port-scan-exporter Chart
./helm install port-scan-exporter --namespace monitoring chart

# Install Grafana in default namespace to be scanned by the exporter
./helm install grafana stable/grafana --namespace default --set rbac.pspEnabled=true --set rbac.pspUseAppArmor=false

# Install Dex in default namespace to be scanned by the exporter
./helm install dex stable/dex --namespace default

# Install GoCD in default namespace to be scanned by the exporter
./helm install gocd stable/gocd --namespace default --set server.persistence.enabled=false --set server.securityContext.runAsGroup=1000 --set server.securityContext.fsGroup=1000 --set agent.securityContext.runAsGroup=1000 --set agent.securityContext.fsGroup=1000
```
