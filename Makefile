mongo:
	docker run -d --name mongo --network=host -e MONGO_INITDB_ROOT_USERNAME=admin -e MONGO_INITDB_ROOT_PASSWORD=admin mongo:latest

minikube:
	minikube start --cni flannel --driver=none --container-runtime='containerd' && kubectl create namespace open5gs
