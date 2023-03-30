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
     linkedTo:
     - serviceAccount:
         managed:
           generateName: test-sa-
EOF
echo 'Binding created'
sleep 2
kubectl wait  --for=jsonpath='{.status.phase}'=AwaitingTokenData  Spiaccesstokenbinding/test-access-token-binding -n $TARGET_NAMESPACE  --timeout=60s
#kubectl get spiaccesstokenbinding/test-access-token-binding -n $TARGET_NAMESPACE -o  json | jq .

UPLOAD_URL=$(kubectl get spiaccesstokenbinding/test-access-token-binding -n $TARGET_NAMESPACE -o  json | jq -r .status.uploadUrl)
echo 'Upload  token to: '$UPLOAD_URL
#echo $GITHUB_SPI
curl --insecure -s \
  -H 'Content-Type: application/json' \
  -H "Authorization: bearer "$TOKEN \
  -d "{ \"access_token\": \"$GITHUB_SPI\" }" \
  $UPLOAD_URL


kubectl wait  --for=jsonpath='{.status.phase}'=Injected  Spiaccesstokenbinding/test-access-token-binding -n $TARGET_NAMESPACE  --timeout=60s

LINKED_SECRET_NAME=$(kubectl get spiaccesstokenbinding/test-access-token-binding -n  ${TARGET_NAMESPACE}   -o  json | jq -r  .status.syncedObjectRef.name)
#
echo 'Linked secret: '${LINKED_SECRET_NAME}
kubectl get secret ${LINKED_SECRET_NAME} -n  ${TARGET_NAMESPACE}   -o  json | jq .

SA_NAME=$(kubectl get spiaccesstokenbinding/test-access-token-binding -n  ${TARGET_NAMESPACE}   -o  json | jq -r  '.status.serviceAccountNames[0]')
echo 'SA name: '${SA_NAME}
kubectl get serviceAccount ${SA_NAME} -n  ${TARGET_NAMESPACE}   -o  json | jq .
#kubectl delete  spiaccesstokenbinding/test-access-token-binding -n  ${TARGET_NAMESPACE}
#
#

#
#
#
#echo 'After'
#echo 'SA count '$(kubectl get sa  -l=spi.appstudio.redhat.com/managed-by-binding -n ${TARGET_NAMESPACE} -o go-template='{{printf "%d\n" (len  .items)}}')
#echo 'secret count '$(kubectl get secret  -l=spi.appstudio.redhat.com/managed-by-binding -n ${TARGET_NAMESPACE} -o go-template='{{printf "%d\n" (len  .items)}}')
#sleep 10
#echo 'After 10 sec'
#echo 'SA count '$(kubectl get sa  -l=spi.appstudio.redhat.com/managed-by-binding -n ${TARGET_NAMESPACE} -o go-template='{{printf "%d\n" (len  .items)}}')
#echo 'secret count '$(kubectl get secret  -l=spi.appstudio.redhat.com/managed-by-binding -n ${TARGET_NAMESPACE} -o go-template='{{printf "%d\n" (len  .items)}}')