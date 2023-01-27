FROM scratch

COPY kube-webhook-certgen /kube-webhook-certgen

ENTRYPOINT ["/kube-webhook-certgen"]
