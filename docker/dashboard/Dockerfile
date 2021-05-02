FROM debian

COPY /built/dashboard /bin/dashboard

# Copy watchdog config as this is used in our dashboard.
COPY /config/watchdog /www/watchdog-config

RUN chmod +x /bin/dashboard

CMD /bin/dashboard
