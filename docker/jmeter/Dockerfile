FROM justb4/jmeter@sha256:088ac52b759a198a5afa5ae13d0a6306e9f2017d71ad140ff57427f6930406f7

# Install the WebSocket Samplers plugin by Peter Doornbosch (v1.3.2)
# https://github.com/ptrd/jmeter-websocket-samplers
ARG WS_PLUGIN_VERSION=1.3.2
ADD https://repo1.maven.org/maven2/net/luminis/jmeter/jmeter-websocket-samplers/${WS_PLUGIN_VERSION}/jmeter-websocket-samplers-${WS_PLUGIN_VERSION}.jar \
    /opt/apache-jmeter-5.5/lib/ext/jmeter-websocket-samplers-${WS_PLUGIN_VERSION}.jar
