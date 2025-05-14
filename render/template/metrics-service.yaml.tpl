apiVersion: v1
kind: Service
metadata:
  name: amf{{ .SliceID }}-metrics-service
  namespace: open5gs
  labels:
    nf: amf
    {{- if ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
    app: monarch
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io.scheme: "http"
    prometheus.io/path: "/metrics"
    prometheus.io/port: "9090"
spec:
  ports:
    - name: metrics # expose metrics port
      port: 9090 # defined in amf chart
  selector:
    nf: amf # target amf pods
---
apiVersion: v1
kind: Service
metadata:
  name: smf{{ .SliceID }}-metrics-service
  namespace: open5gs
  labels:
    nf: smf
    {{- if ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
    app: monarch
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io.scheme: "http"
    prometheus.io/path: "/metrics"
    prometheus.io/port: "9090"
spec: # smf和upf是切片独有的, 对应有多个deployment, 所以指定clusterIP为None 即创建的是一个 Headless Service，该服务不会为用户分配一个虚拟的 Cluster IP。相反，Kubernetes 会直接将请求路由到所有符合标签选择器（nf: upf）的 Pod。
  clusterIP: None
  ports:
    - name: metrics # expose metrics port
      port: 9090 # defined in smf chart
  selector:
    nf: smf # target smf pods
    {{- if ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
---
apiVersion: v1
kind: Service
metadata:
  name: upf{{ .SliceID }}-metrics-service
  namespace: open5gs
  labels:
    nf: upf
    {{- if ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
    app: monarch
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io.scheme: "http"
    prometheus.io/path: "/metrics"
    prometheus.io/port: "9090"
spec:
  clusterIP: None
  ports:
    - name: metrics # expose metrics port
      port: 9090 # defined in upf chart
  selector:
    nf: upf # target upf pods
    {{- if ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
