FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-beeline"]
COPY baton-beeline /