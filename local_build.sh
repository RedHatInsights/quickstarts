APP=quickstarts
NS=quickstarts

TAG=`cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 7 | head -n 1`
IMAGE="127.0.0.1:5000/$APP"

podman build -t $IMAGE:$TAG -f Dockerfile && \
podman push $IMAGE:$TAG `minikube ip`:5000/$APP:$TAG --tls-verify=false && \
bonfire deploy $APP --get-dependencies --namespace $NS --set-parameter $APP/$APP/IMAGE_TAG=$TAG

echo $TAG
