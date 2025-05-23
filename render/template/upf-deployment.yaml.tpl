apiVersion: apps/v1
kind: Deployment
metadata:
  name: open5gs-upf{{.ID}}
  labels:
    app: open5gs
    nf: upf
    slice: {{.ID}}
    name: upf{{.ID}}
spec:
  selector:
    matchLabels:
      app: open5gs
      nf: upf
      slice: {{.ID}}
      name: upf{{.ID}}
  replicas: 1
  template:
    metadata:
      labels:
        app: open5gs
        nf: upf
        slice: {{.ID}}
        name: upf{{.ID}}
      annotations:
        k8s.v1.cni.cncf.io/networks: '[
          { "name": "n3network", "interface": "n3", "ips": [ "{{.N3Addr}}" ] },
          { "name": "n4network", "interface": "n4", "ips": [ "{{.N4Addr}}" ] }
          ]'
    spec:
      # nodeSelector:
      #   kubernetes.io/hostname: cn231
      initContainers:
        - name: wait-smf
          image: busybox:1.32.0
          env:
            - name: DEPENDENCIES
              value: smf-nsmf:80
          command:
            [
              "sh",
              "-c",
              "until nc -z $DEPENDENCIES; do echo waiting for the SMF; sleep 2; done;",
            ]
      containers:
        - name: upf
          image: ghcr.io/niloysh/open5gs:v2.7.0-upf-metrics-v2
          imagePullPolicy: IfNotPresent
          command: ["/open5gs/config/wrapper.sh"]
          volumeMounts:
            - mountPath: /open5gs/config/
              name: upf{{.ID}}-volume
          ports:
            - containerPort: 8805
              name: n4
              protocol: UDP
          resources:
            requests:
              memory: "256Mi"
              cpu: "200m"
            limits:
              memory: "512Mi"
              cpu: "500m"
          securityContext:
            privileged: true
      restartPolicy: Always
      volumes:
        - name: upf{{.ID}}-volume
          configMap:
            name: upf{{.ID}}-configmap
            items:
              - key: upfcfg.yaml
                path: upfcfg.yaml
              - key: wrapper.sh
                path: wrapper.sh
                mode: 0777
