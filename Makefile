.PHONY: mongo minikube k3d cloc build push

mongo:
	docker run -d --name mongo --network=host -e MONGODB_ROOT_USER=admin -e MONGODB_ROOT_PASSWORD=admin -e MONGODB_DATABASE=slicer bitnami/mongodb

minikube:
	minikube start --cni cilium --driver=docker --nodes=3 && kubectl create namespace open5gs

minikube-local:
	minikube start --cni flannel --driver=none --container-runtime='containerd' && kubectl create namespace open5gs

k3d:
	k3d cluster create 5g --servers 1 --agents 2 --k3s-arg "--flannel-backend=none@server:0" --k3s-arg "--disable-network-policy@server:0" --k3s-arg "--disable=traefik@server:0" \
	--volume /sys/kernel/debug:/sys/kernel/debug:rw --volume /sys/kernel/tracing:/sys/kernel/tracing:rw && kubectl create namespace open5gs && kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

cloc:
	cloc --exclude-dir=external,docs,charts --exclude-ext=csv,py .

charts:
	git submodule update --recursive --remote

CHARTS := open5gs-amf open5gs-ausf open5gs-bsf open5gs-nrf open5gs-nssf \
          open5gs-pcf open5gs-scp open5gs-smf open5gs-udm open5gs-udr \
          open5gs-upf open5gs-webui

jsonschema2go:
	@for chart in $(CHARTS); do \
		base=$$(echo $$chart | sed 's/open5gs-//'); \
		echo "Generating $$base.go from ./charts/$$chart/values.schema.json..."; \
		go-jsonschema -e -p value -t ./charts/$$chart/values.schema.json -o ./kube/value/$$base.go; \
	done

# docker
# IMAGE_PREFIX := shui12jiao/slicer
IMAGE_PREFIX := crpi-sut5dyyu9y5gqtfq.cn-shanghai.personal.cr.aliyuncs.com/sminggg/slicer
VERSION := $(VERSION)

ifeq ($(VERSION),) # 如果未提供版本号，则使用默认值 'latest'
    VERSION := latest
endif

build:
	docker build -t $(IMAGE_PREFIX):$(VERSION) .
push:
	docker push $(IMAGE_PREFIX):$(VERSION)

run:
	docker run -d --name slicer --network=host --env-file .env shui12jiao/slicer:latest
	
# kubernetes

prepare:
	cd external/Open5gs && kubectl create namespace open5gs || kubectl apply -f multus-daemonset-thick.yml &&
	kubectl apply -k mongodb -n open5gs && kubectl apply -k networks5g -n open5gs && kubectl apply -k msd/overlays/open5gs-metrics -n open5gs
	
deploy:
	kubectl apply -k kubernetes

# helm
package:
	cd charts/common && helm dependency build && helm lint && helm package . --destination ../ && \
	cd ../slice && helm dependency build && helm lint && helm package . --destination ../
