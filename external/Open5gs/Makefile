install:
	kubectl create namespace open5gs && kubectl apply -k mongodb -n open5gs && \
	kubectl apply -k networks5g -n open5gs && kubectl apply -k msd/overlays/open5gs-metrics -n open5gs && \
	kubectl apply -k open5gs-webui -n open5gs