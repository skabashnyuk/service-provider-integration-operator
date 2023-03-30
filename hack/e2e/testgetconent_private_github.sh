#!/usr/bin/env bash
set -e
TARGET_NAMESPACE=$(~/.bin/SPI/getnamespace.sh)
TOKEN=$(~/.bin/SPI/gettoken.sh)
echo 'cleaning '$TARGET_NAMESPACE
delete_all_spi_cr.sh $TARGET_NAMESPACE

echo 'Going to create SPIFileContentRequest'
cat <<EOF | kubectl apply -n $TARGET_NAMESPACE -f -
apiVersion: appstudio.redhat.com/v1beta1
kind: SPIFileContentRequest
metadata:
  name: test-file-content-request
spec:
  repoUrl: https://github.com/skabashnyuk/some-private-repo
  filePath: devfile.yaml
EOF
echo 'Binding created'

kubectl wait  --for=jsonpath='{.status.phase}'=AwaitingTokenData SPIFileContentRequest/test-file-content-request -n $TARGET_NAMESPACE  --timeout=60s

#kubectl get SPIFileContentRequest/test-file-content-request -n $TARGET_NAMESPACE -o  json | jq .


UPLOAD_URL=$(kubectl get SPIFileContentRequest/test-file-content-request -n $TARGET_NAMESPACE -o  json | jq -r .status.tokenUploadUrl)
LINKING_BINDING_NAME=$(kubectl get SPIFileContentRequest/test-file-content-request -n $TARGET_NAMESPACE -o  json | jq -r .status.linkedBindingName)
echo 'Upload url: '$UPLOAD_URL

curl --insecure -s \
  -H 'Content-Type: application/json' \
  -H "Authorization: bearer "$TOKEN \
  -d "{ \"access_token\": \"$GITHUB_SPI\" }" \
  $UPLOAD_URL
kubectl wait  --for=jsonpath='{.status.phase}'=Delivered SPIFileContentRequest/test-file-content-request -n $TARGET_NAMESPACE  --timeout=60s
echo
echo 'Content:'
kubectl get SPIFileContentRequest/test-file-content-request -n $TARGET_NAMESPACE -o  json | jq -r .status.content | base64 -d