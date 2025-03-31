apiVersion: apps/v1
kind: Deployment
metadata:
  name: kpi{{ .SliceID }}-calculator
  namespace: monarch
  labels:
    {{- if .ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
    app: monarch
    component: kpi-calculator
spec:
  selector:
    matchLabels:
      {{- if .ne .SliceID "" }}
      slice: {{ .SliceID }}
      {{- end }}
      app: monarch
      component: kpi-calculator
  replicas: 1
  template:
    metadata:
      labels:
        {{- if .ne .SliceID "" }}
        slice: {{ .SliceID }}
        {{- end }}
        app: monarch
        component: kpi-calculator
    spec:
      containers:
        - image: ghcr.io/niloysh/kpi-calculator-open5gs:v1.0.0-standard
          name: kpi-calculator
          imagePullPolicy: Always
          ports:
            - name: metrics
              containerPort: 9000
          env:
            - name: UPDATE_PERIOD
              value: "1"
            - name: MONARCH_THANOS_URL
              value: "${MONARCH_THANOS_URL}"
            - name: TIME_RANGE
              value: "30s"
          command: ["/bin/bash", "-c", "--"]
          args: ["python -u kpi_calculator.py --log debug"]
          resources:
            requests:
              memory: "100Mi"
              cpu: "100m"
            limits:
              memory: "200Mi"
              cpu: "200m"
      restartPolicy: Always
---
apiVersion: v1
kind: Service
metadata:
  name: kpi{{ .SliceID }}-calculator-service
  namespace: monarch
  labels:
    {{- if .ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
    app: monarch
    component: kpi-calculator
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io.scheme: "http"
    prometheus.io/path: "/metrics"
    prometheus.io/port: "9000"
spec:
  ports:
    - name: metrics # expose metrics port
      port: 9000 # defined in chart
      targetPort: metrics # port name in pod
  selector:
    {{- if .ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
    app: monarch # target pods
    component: kpi-calculator
---

