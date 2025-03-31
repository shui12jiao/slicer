apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: amf-servicemonitor
  namespace: monarch
  labels:
    nf: amf
    app: monarch
spec:
  namespaceSelector:
    any: true # important otherwise this is not picked up
  selector:
    matchLabels:
      nf: amf # target amf service
  endpoints:
    - port: metrics
      interval: "${MONARCH_MONITORING_INTERVAL}"
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: smf{{ join .SliceIDs "," }}-servicemonitor
  namespace: monarch
  labels:
    nf: smf{{ join .SliceIDs "," }}
    app: monarch
spec:
  namespaceSelector:
    any: true # important otherwise this is not picked up
  selector:
    matchExpressions:
      - key: nf
        operator: In
        values:
          {{- range .SliceIDs }}
          - smf{{ . | quote }} # target smf service
          {{- end }}
  endpoints:
    - port: metrics
      interval: "${MONARCH_MONITORING_INTERVAL}"
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: upf{{ join .SliceIDs "," }}-servicemonitor
  namespace: monarch
  labels:
    nf: upf{{ join .SliceIDs "," }}
    app: monarch
spec:
  namespaceSelector:
    any: true # important otherwise this is not picked up
  selector:
    matchExpressions:
      - key: nf
        operator: In
        values:
          {{- range .SliceIDs }}
          - upf{{ . | quote }} # target upf service
          {{- end }}
  endpoints:
    - port: metrics
      interval: "${MONARCH_MONITORING_INTERVAL}"
