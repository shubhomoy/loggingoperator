#!/bin/bash

kubectl delete -f 5FluentD.yaml
kubectl delete -f 4Kibana.yaml
kubectl delete -f 3Elasticsearch.yaml
kubectl delete -f 2Fluentbit.yaml
kubectl delete -f 1CreateRoleAndAccounts.yaml