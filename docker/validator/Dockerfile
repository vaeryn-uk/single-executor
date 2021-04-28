FROM debian

COPY built/binary /bin/binary
COPY built/watchdog /bin/watchdog

CMD /bin/watchdog -f /etc/watchdog/watchdog.instance.yaml -c /etc/watchdog/watchdog.cluster.yaml