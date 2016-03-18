FROM scratch
ADD cacert.pem /etc/ssl/ca-bundle.pem
ADD dist/ec2cluster.Linux.x86_64 /ec2cluster
ENV PATH=/
CMD ["/ec2cluster"]