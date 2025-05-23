apiVersion: v1
kind: Service
metadata:
  name: smf{{.ID}}-nsmf
  labels:
    app: open5gs
    nf: smf
    slice: {{.ID}}
    name: smf{{.ID}}
spec:
  ports:
    - name: sbi
      port: 80
    - name: gtpc
      port: 2123
      protocol: UDP
    - name: gtpu
      port: 2152
      protocol: UDP
    - name: diameter-base
      port: 3868
    - name: diameter-over
      port: 5868
  selector:
    app: open5gs
    nf: smf
    slice: {{.ID}}
    name: smf{{.ID}}
