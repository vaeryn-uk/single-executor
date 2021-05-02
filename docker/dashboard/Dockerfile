FROM debian

RUN apt update
RUN apt install docker docker-compose -y

COPY /built/dashboard /bin/dashboard

# Copy watchdog config as this is used in our dashboard.
COPY /config/watchdog /www/watchdog-config

RUN chmod +x /bin/dashboard

CMD /bin/dashboard
