**Project status: whiteboard**

This project is in a conceptual stage.
It is liable to be missing key features, and break unexpectedly.

# Kubernetes rollout API

The Kubernetes rollout (`rollout.zeitgeistlabs.io`) API is an API for deploying resources on Kubernetes.
[[Blog]](https://timewitch.net/post/2019-12-30-pinneddeployments/)

[![asciicast](https://asciinema.org/a/291109.svg)](https://asciinema.org/a/291109)

## Objects

### PinnedDeployment

A PinnedDeployment behaves similarly to a Deployment,
except it has explicit previous and next versions (Pod specs).
A PinnedDeployment will maintain the specified percent of old and new pods.

```
# Example PinnedDeployment
apiVersion: rollout.zeitgeistlabs.io/v1alpha1
kind: PinnedDeployment
metadata:
  name: example
spec:
  selector:
    matchLabels:
      app: example
  replicas: 5
  replicasPercentNext: 20
  replicasRoundingStrategy: "Nearest"
  templates:
    previous:
      metadata:
        labels:
          app: example
      spec:
        containers:
          - name: webserver
            image: nginx:1.16
            ports:
              - containerPort: 80
    next:
      metadata:
        labels:
          app: example
      spec:
        containers:
          - name: webserver
            image: nginx:1.17
            ports:
              - containerPort: 80
```

PinnedDeployments allow Kubernetes users to easily roll out gradual deployments to a service.

## Repo Contents

* rollout API controller
* rollout API admission webhooks
* rollout API CRD

# Installing

(Note: don't use this in a non-development environment yet!)

Installation steps are still rough.
In particular,
`kubectl apply` does not work for installation/updates of the CRD,
as it is larger than kubectl supports.
Note that deleting the CRD (to re-create it) will delete all PinnedDeployments.

Tested version: Kubernetes 1.16.

Local dependencies:
* kubectl, with admin access to a cluster
* kustomize (if making changes)

1. Deploy cert-manager to Kubernetes (for admission webhook certificate management).
    1. `kubectl create namespace cert-manager && kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v0.11.1/cert-manager.yaml`
    1. See https://docs.cert-manager.io/en/release-0.11/getting-started/install/kubernetes.html
1. Build the admission webhook images.
    1. `make docker-build`
1. Push the admission webhook images to a container registry, and update the image path in the Makefile.
    1. If using `kind`, load the image with `kind load docker-image --name <cluster> controller:latest`
1. Create the CRD and associated resources.
    1. `make deploy`
    1. OR, if you haven't changed anything, `kubectl create -f ./kustomize_output.yaml`.