apiVersion: v1
kind: ConfigMap
metadata:
  name: upf{{.ID}}-configmap
  labels:
    app: open5gs
    nf: upf
    slice: {{.ID}}
    name: upf{{.ID}}
data:
  upfcfg.yaml: |
    logger:
        file: /open5gs/install/var/log/open5gs/upf.log
        level: info

    global:
      max:
        ue: 1024

    upf:
      pfcp:
        server:
          - dev: n4
      gtpu:
        server:
          - dev: n3
      session:
      {{- range .SessionValues }}
        - subnet: {{.Subnet}}
          gateway: {{.Gateway}}
          dnn: {{.DNN}}
      {{- end }}
      metrics:
        server:
          - address: 0.0.0.0
            port: 9090

  wrapper.sh: |
    #!/bin/bash   

    sysctl -w net.ipv6.conf.all.disable_ipv6=1;
    sh -c "echo 1 > /proc/sys/net/ipv4/ip_forward";
    ip tuntap add name ogstun mode tun;
    {{- range .SessionValues }}
    ip addr add {{.GatewayWithCIDR}} dev ogstun;
    iptables -t nat -A POSTROUTING -s {{.Subnet}} ! -o ogstun -j MASQUERADE;
    {{- end}}
    ip link set ogstun up;

    /open5gs/install/bin/open5gs-upfd -c /open5gs/config/upfcfg.yaml