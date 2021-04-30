FROM debian

COPY built/binary /bin/binary
COPY built/watchdog /bin/watchdog

RUN chmod +x /bin/binary /bin/watchdog

COPY config/watchdog /etc/watchdog

CMD /bin/watchdog -f /etc/watchdog/watchdog.instance.yaml -c /etc/watchdog/watchdog.cluster.yaml
