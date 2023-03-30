#!/usr/bin/env bash
set -e
TARGET_NAMESPACE=$(~/.bin/SPI/getnamespace.sh)
TOKEN=$(~/.bin/SPI/gettoken.sh)
echo 'cleaning '$TARGET_NAMESPACE
delete_all_spi_cr.sh $TARGET_NAMESPACE

echo "Going to create SPIAccessTokenBinding in "$TARGET_NAMESPACE

cat <<EOF | kubectl apply -n $TARGET_NAMESPACE -f -
apiVersion: appstudio.redhat.com/v1beta1
kind: SPIAccessTokenBinding
metadata:
  name: test-access-token-binding
spec:
  permissions:
    required:
    - area: repository
      type: r
  repoUrl: https://github.com/spi-test-org-1/spi-org-test-repo-1
  secret:
    type: kubernetes.io/basic-auth
EOF
echo 'Binding created'
sleep 2
kubectl wait  --for=jsonpath='{.status.phase}'=AwaitingTokenData  Spiaccesstokenbinding/test-access-token-binding -n $TARGET_NAMESPACE  --timeout=60s
kubectl get spiaccesstokenbinding/test-access-token-binding -n $TARGET_NAMESPACE -o  json | jq .

BASEURL=$(kubectl get spiaccesstokenbinding/test-access-token-binding -n $TARGET_NAMESPACE -o  json | jq -r  .status.oAuthUrl)
open $BASEURL'&k8s_token='$TOKEN

kubectl wait  --for=jsonpath='{.status.phase}'=Injected  Spiaccesstokenbinding/test-access-token-binding -n $TARGET_NAMESPACE  --timeout=60s

LINKED_SECRET_NAME=$(kubectl get spiaccesstokenbinding/test-access-token-binding -n  ${TARGET_NAMESPACE}   -o  json | jq -r  .status.syncedObjectRef.name)
echo 'Linked secret: '${LINKED_SECRET_NAME}