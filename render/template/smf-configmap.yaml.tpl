apiVersion: v1
kind: ConfigMap
metadata:
  name: smf{{.ID}}-configmap
  labels:
    app: open5gs
    nf: smf
    slice: {{.ID}}
    name: smf{{.ID}}
data:
  smfcfg.yaml: |
    logger:
      file: /open5gs/install/var/log/open5gs/smf.log

    global:
      max:
        ue: 1024

    smf:
      sbi:
        server:
          - dev: eth0
            advertise: smf{{.ID}}-nsmf
            port: 80
        client:
          scp:
            - uri: http://scp-nscp:80
      pfcp:
        server:
          - dev: n4
        client:
          upf:
            - address: {{.UPFN4Addr}}
      gtpc:
        server:
          - dev: eth0
      gtpu:
        server:
          - dev: n3
      metrics:
        server:
          - address: 0.0.0.0
            port: 9090
      session:
      {{- range .SessionValues }}
        - subnet: {{.Subnet}}
          gateway: {{.Gateway}}
      {{- end }}
      dns:
        - 8.8.8.8
        - 8.8.4.4
      mtu: 1400
      ctf:
        enabled: auto
      freeDiameter: /open5gs/install/etc/freeDiameter/smf.conf

      info:
        - s_nssai:
          - sst: {{.SST}}
            sd: {{.SD}}
            dnn:
            {{- range .SessionValues }}
             - {{.DNN}}
            {{- end}}

