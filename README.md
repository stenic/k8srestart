# K8sRestart

[![Last release](https://github.com/stenic/k8srestart/actions/workflows/release.yaml/badge.svg)](https://github.com/stenic/k8srestart/actions/workflows/release.yaml)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/k8srestart)](https://artifacthub.io/packages/search?repo=k8srestart)


k8srestart restarts pods and deployments after a set amount of time.

## Behaviour

`pods` will be deleted once the time has past.
`deployments` will be triggered as a `rollout` and will restart following the deployment specifications.


## Installation

```sh
helm repo add k8srestart https://stenic.github.io/k8srestart/
helm install k8srestart --namespace mynamespace k8srestart/k8srestart
```


## Annotations

You can add these Kubernetes annotations to specific service objects to customize k8srestart's behaviour.

`k8srestart.stenic.io/restart-seconds`
(string) Amount of seconds after which the pod/deployment will restart.


## Build & run

```
docker build -t k8srestart .
docker run -ti -p 8080:8080 -v ~/.kube:/home/nonroot/.kube k8srestart --interval=5 -v=2
```