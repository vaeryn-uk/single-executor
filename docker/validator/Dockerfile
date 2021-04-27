FROM debian

COPY built/binary /bin/binary
COPY built/watchdog /bin/watchdog

CMD /bin/watchdog -c /etc/watchdog.yaml