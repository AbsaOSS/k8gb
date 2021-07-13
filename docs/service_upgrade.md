## Changning CoreDNS service type
Since Kubernetes API doesn't allow to change service type, following steps should be taken.

Option 1:
* deploy new service with changed name by setting `coredns.service.name=migrate-svc`
  ```helm upgrade -n <k8gb namespace> --install <release name> <chart repo>/k8gb --set coredns.service.name=migrate-svc```
* deploy back with original service name
  ```helm upgrade -n <k8gb namespace> --install <release name> <chart repo>/k8gb```

Option 2:
Simply delete CoreDNS service prior running helm upgrade:
* ```kubectl delete svc -n <k8gb namespace> <release name>-coredns```
* ```helm upgrade -n <k8gb namespace> --install <release name> <chart repo>/k8gb```
