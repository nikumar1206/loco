#!/bin/bash

set -e

echo "Setting up Loco for local development..."

# Check if minikube is installed
if ! command -v minikube &> /dev/null; then
    echo "Installing minikube..."
    # For macOS ARM64
    curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-darwin-arm64
    sudo install minikube-darwin-arm64 /usr/local/bin/minikube
    rm minikube-darwin-arm64
fi

# Start minikube if not running
if ! minikube status | grep -q "Running"; then
    echo "Starting minikube..."
    minikube start
fi

# Check if helm is installed
if ! command -v helm &> /dev/null; then
    echo "Installing helm..."
    # For macOS ARM64
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
    chmod 700 get_helm.sh
    ./get_helm.sh
    rm get_helm.sh
fi


# Install Envoy Gateway, todo: bump to 1.4 once supported
echo "Installing Envoy Gateway + Gateway Crds..."
kubectl apply --server-side=true -f https://github.com/envoyproxy/gateway/releases/download/v1.5.2/envoy-gateway-crds.yaml
helm upgrade eg oci://docker.io/envoyproxy/gateway-helm -n envoy-gateway-system --create-namespace -i

# Install cert-manager
echo "Installing cert-manager..."
helm repo add jetstack https://charts.jetstack.io --force-update
helm repo update
helm upgrade --install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set installCRDs=true -i


# create loco namespace
kubectl apply -f kube/loco_namespace.yml
# Create self-signed issuer

echo "Creating self-signed issuer..."
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned-issuer
spec:
  selfSigned: {}
EOF

# Create certificate for localhost
echo "Creating certificate for localhost..."
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: loco-tls
  namespace: loco-system
spec:
  secretName: loco-tls
  issuerRef:
    name: selfsigned-issuer
    kind: ClusterIssuer
  commonName: localhost
  dnsNames:
  - localhost
EOF

# Apply base resources
echo "Applying base resources..."
kubectl apply -f kube/gateway.yml
kubectl apply -f kube/loco_routes.yml
kubectl apply -f kube/loco_rbac.yml

# Build and load image
echo "Building loco image..."
docker build -t loco-api:latest -f api/Dockerfile .
minikube image load loco-api:latest

# Create secrets with dummy values for local dev
echo "Creating secrets..."
kubectl create secret generic env-config \
  --from-literal=GITLAB_PAT=dummy \
  --from-literal=GITLAB_PROJECT_ID=dummy \
  --from-literal=GITLAB_TOKEN_NAME=dummy \
  --from-literal=GH_OAUTH_CLIENT_ID=dummy \
  --from-literal=GITLAB_URL=dummy \
  --from-literal=GITLAB_REGISTRY_URL=dummy \
  --from-literal=GITLAB_DEPLOY_TOKEN_NAME=dummy \
  --from-literal=APP_ENV=DEVELOPMENT \
  --from-literal=LOG_LEVEL=-4 \
  --from-literal=PORT=:8000 \
  --from-literal=GH_OAUTH_CLIENT_SECRET=dummy \
  --from-literal=GH_OAUTH_REDIRECT_URL=http://localhost:8000/api/v1/oauth/github/callback \
  --from-literal=GH_OAUTH_STATE=dummy \
  -n loco-system --dry-run=client -o yaml | kubectl apply -f -

# Update deployment to use local image
kubectl apply -f kube/loco_service.yml
echo "Updating deployment image..."
kubectl set image -n loco-system deployment/loco-api loco-api=loco-api:latest
kubectl patch deployment loco-api -n loco-system --type='json' -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}]'


echo "\n\nSetup complete!"
echo "Note: the secrets installed are just dummy."
echo "To access the application:"
echo "1. Run 'minikube tunnel' in a separate terminal to expose the LoadBalancer."
echo "2. Access the app at http://localhost"
echo "3. To check status: kubectl get pods -n loco-system"
echo "4. To view logs: kubectl logs -n loco-system deployment/loco-api"
