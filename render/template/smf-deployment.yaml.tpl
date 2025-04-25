apiVersion: apps/v1
kind: Deployment
metadata:
  name: open5gs-smf{{.ID}}
  labels:
    app: open5gs
    nf: smf
    slice: {{.ID}}
    name: smf{{.ID}}
spec:
  selector:
    matchLabels:
      app: open5gs
      nf: smf
      slice: {{.ID}}
      name: smf{{.ID}}
  replicas: 1
  template:
    metadata:
      labels:
        app: open5gs
        nf: smf
        slice: {{.ID}}
        name: smf{{.ID}}
      annotations:
        k8s.v1.cni.cncf.io/networks: '[
          { "name": "n4network", "interface": "n4", "ips": [ "{{.N4Addr}}" ] },
          { "name": "n3network", "interface": "n3", "ips": [ "{{.N3Addr}}" ] }
          ]'
    spec:
      # nodeSelector:
      #   kubernetes.io/hostname: cn231
      initContainers:
        - name: wait-ausf
          image: busybox:1.32.0
          env:
            - name: DEPENDENCIES
              value: ausf-nausf:80
          command:
            [
              "sh",
              "-c",
              "until nc -z $DEPENDENCIES; do echo waiting for the AUSF; sleep 2; done;",
            ]
      containers:
        - image: docker.io/shui12jiao/open5gs:v2.7.2
          name: smf{{.ID}}
          imagePullPolicy: IfNotPresent
          ports:
            - name: nsmf
              containerPort: 80
            - name: pfcp
              containerPort: 8805
              protocol: UDP
          command: ["./open5gs-smfd"]
          args: ["-c", "/open5gs/config/smfcfg.yaml"]
          env:
            - name: GIN_MODE
              value: release
          volumeMounts:
            - mountPath: /open5gs/config/
              name: smf{{.ID}}-volume
          securityContext:
            capabilities:
              add: ["NET_ADMIN"]
          resources:
            requests:
              memory: "256Mi"
              cpu: "200m"
            limits:
              memory: "512Mi"
              cpu: "500m"
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      volumes:
        - name: smf{{.ID}}-volume
          projected:
            sources:
              - configMap:
                  name: smf{{.ID}}-configmap
                  items:
                    - key: smfcfg.yaml
                      path: smfcfg.yaml
