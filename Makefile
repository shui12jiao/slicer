.PHONY: mongo minikube k3d cloc build push

mongo:
	docker run -d --name mongo --network=host -e MONGODB_ROOT_USER=admin -e MONGODB_ROOT_PASSWORD=admin -e MONGODB_DATABASE=slicer bitnami/mongodb

minikube:
	minikube start --cni flannel --driver=none --container-runtime='containerd' && kubectl create namespace open5gs

k3d:
	k3d cluster create 5g --servers 1 --agents 2 --k3s-arg "--flannel-backend=none@server:0" --k3s-arg "--disable-network-policy@server:0" --k3s-arg "--disable=traefik@server:0" \
	--volume /sys/kernel/debug:/sys/kernel/debug:rw --volume /sys/kernel/tracing:/sys/kernel/tracing:rw && kubectl create namespace open5gs && kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

cloc:
	cloc --exclude-dir=kubernetes,Monarch --exclude-ext=csv,py .


# docker相关
IMAGE_PREFIX := shui12jiao/slicer
VERSION := $(VERSION)

ifeq ($(VERSION),) # 如果未提供版本号，则使用默认值 'latest'
    VERSION := latest
endif

build:
	docker build -t $(IMAGE_PREFIX):$(VERSION) .
push:
	docker push $(IMAGE_PREFIX):$(VERSION)


