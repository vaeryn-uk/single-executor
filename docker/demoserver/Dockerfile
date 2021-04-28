FROM debian

COPY /built/demoserver /bin/demoserver

# Copy watchdog config as this is used in our dashboard.
COPY /config/watchdog /www/watchdog-config

CMD /bin/demoserver