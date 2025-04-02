apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: amf{{ .SliceID }}-servicemonitor # 不同的切片使用不同的servicemonitor，否则删除时会删除所有的servicemonitor
  namespace: monarch
  labels:
    nf: amf
    {{- if ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
    app: monarch
spec:
  namespaceSelector:
    any: true # important otherwise this is not picked up
  selector:
    matchLabels:
      nf: amf # target amf service
      {{- if ne .SliceID "" }}
      slice: {{ .SliceID }}
      {{- end }}
  endpoints:
    - port: metrics
      interval: "{{ .Interval }}s"
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: smf{{ .SliceID }}-servicemonitor
  namespace: monarch
  labels:
    nf: smf
    {{- if ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
    app: monarch
spec:
  namespaceSelector:
    any: true # important otherwise this is not picked up
  selector:
    matchLabels:
      nf: smf # target smf service
      {{- if ne .SliceID "" }}
      slice: {{ .SliceID }}
      {{- end }}
  endpoints:
    - port: metrics
      interval: "{{ .Interval }}s"
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: upf{{ .SliceID }}-servicemonitor
  namespace: monarch
  labels:
    nf: upf
    {{- if ne .SliceID "" }}
    slice: {{ .SliceID }}
    {{- end }}
    app: monarch
spec:
  namespaceSelector:
    any: true # important otherwise this is not picked up
  selector:
    matchLabels:
      nf: upf
      {{- if ne .SliceID "" }}
      slice: {{ .SliceID }}
      {{- end }}
  endpoints:
    - port: metrics
      interval: "{{ .Interval }}s"
