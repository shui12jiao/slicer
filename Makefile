mongo:
	docker run -d --name mongo --network=host -e MONGODB_ROOT_USER=admin -e MONGODB_ROOT_PASSWORD=admin -e MONGODB_DATABASE=slicer bitnami/mongodb

minikube:
	minikube start --cni flannel --driver=none --container-runtime='containerd' && kubectl create namespace open5gs
