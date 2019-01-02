#!/bin/bash

kubectl delete -f /Users/shubhomoy.biswas/go/src/github.com/log_management/logging-operator/deploy/crds/logging_v1alpha1_logmanagement_cr.yaml
kubectl delete -f 2SetupOperator.yaml
kubectl delete -f 1CreateRoleAndAccounts.yaml