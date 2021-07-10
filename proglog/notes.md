
pg.195

Use Kind for Local Development and Continuous
Integration
Kind 3 (an acronym for Kubernetes IN Docker) is a tool developed by the
Kubernetes team to run local Kubernetes clusters using Docker containers
as nodes. It’s the easiest way to run your own Kubernetes cluster, and it’s
great for local development, testing, and continuous integration.
To install Kind, run the following:
$ curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.8.1/kind-$(uname)-amd64
$ chmod +x ./kind
$ mv ./kind /usr/local/bin/kind
To use Kind, you’ll need to install Docker. 4 See Docker’s dedicated install
instructions for your operation system.
With Docker running, you can create a Kind cluster by running:
$ kind create cluster
You can then verify that Kind created your cluster and configured kubectl to
use it by running the following:
$ kubectl cluster-info
> Kubernetes master is running at https://127.0.0.1:46023
KubeDNS is running at \
https://127.0.0.1:46023/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
To further debug and diagnose cluster problems, use kubectl cluster-info dump .
Kind runs one Docker container representing one Kubernetes node in the
cluster. By default, Kind runs a single node cluster with everything needed
for a functioning Kubernetes cluster. You can see the Node container by
running this:
$ docker ps
CONTAINER ID IMAGE COMMAND CREATED ...
033de99b1e53 kindest/node:v1.18.2 "/usr/local/bin/entr..." 2 minutes...
We have a running Kubernetes cluster now—let’s run our service on it! To