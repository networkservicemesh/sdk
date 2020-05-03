---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "icmp-responder"
      networkservicemesh.io/impl: "icmp-responder"
  replicas: 2
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "icmp-responder"
        networkservicemesh.io/impl: "icmp-responder"
    spec:
      serviceAccount: nse-acc
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: networkservicemesh.io/app
                    operator: In
                    values:
                      - icmp-responder
                  - key: networkservicemesh.io/impl
                    operator: In
                    values:
                      - icmp-responder
              topologyKey: "kubernetes.io/hostname"
      containers:
        - name: icmp-responder-nse
          image: fkautz/icmp-responder:0.0.8
          command: ["/bin/sh"]
          imagePullPolicy: {{ .Values.pullPolicy }}
          stdin: true
          stdinOnce: true
          tty: true
          env:
            - name: ADVERTISE_NSE_NAME
              value: "icmp-responder"
            - name: ADVERTISE_NSE_LABELS
              value: "app=icmp-responder"
            - name: TRACER_ENABLED
              value: "true"
            - name: IP_ADDRESS
              value: "172.16.1.0/24"
            - name: NSM_LISTEN_ON_URL
              value: unix:///run/networkservicemesh/nsm.client.io.sock
            - name: NSM_CONNECT_TO_URL
              value: unix:///run/networkservicemesh/nsm.server.io.sock
            - name: NSM_REGISTRY_URL
              value: unix:///run/networkservicemesh/registry.io.sock
            - name: GRPC_GO_LOG_VERBOSITY_LEVEL
              value: "99"
            - name: GRPC_GO_LOG_SEVERITY_LEVEL
              value: "info"
{{- if .Values.global.JaegerTracing }}
            - name: TRACER_ENABLED
              value: "true"
            - name: JAEGER_AGENT_HOST
              value: jaeger.nsm-system
            - name: JAEGER_AGENT_PORT
              value: "6831"
{{- end }}
          resources:
            limits:
              networkservicemesh.io/socket: 1
metadata:
  name: icmp-responder-nse
  namespace: {{ .Release.Namespace }}
