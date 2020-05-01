set shell := ["powershell.exe", "-c"]

pod := `kubectl get pods -l networkservicemesh.io/app=icmp-responder --field-selector status.phase=Running -o jsonpath="{.items[0].metadata.name}"`

build:
    docker build -t fkautz/icmp-responder:0.0.8 .

load:
    kind load docker-image fkautz/icmp-responder:0.0.8

kill:
    kubectl delete pod {{pod}}

install:
    helm install ./icmp-responder --generate-name

term:
    kubectl exec -it {{pod}} -- /bin/sh

list:
    kubectl get pods --all-namespaces

register:
    kubectl exec -n spire spire-server-0 -- \
        /opt/spire/bin/spire-server entry create \
        -spiffeID spiffe://test.com/icmp-responder \
        -parentID spiffe://test.com/spire-agent \
        -selector k8s:ns:default \
        -selector k8s:sa:default
