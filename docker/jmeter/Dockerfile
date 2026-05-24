FROM justb4/jmeter@sha256:088ac52b759a198a5afa5ae13d0a6306e9f2017d71ad140ff57427f6930406f7

# Install the WebSocket Samplers plugin pinned in websocket-plugin.lock.
# Plugin by Peter Doornbosch (v1.3.2)
# https://github.com/ptrd/jmeter-websocket-samplers
ARG WS_PLUGIN_VERSION=1.3.2
ARG WS_PLUGIN_SHA256=f9818f27fbbfe871bc2fb191215970928a475a586c640c7bb29bf76a9d7ced3f
ADD --checksum=sha256:${WS_PLUGIN_SHA256} https://repo1.maven.org/maven2/net/luminis/jmeter/jmeter-websocket-samplers/${WS_PLUGIN_VERSION}/jmeter-websocket-samplers-${WS_PLUGIN_VERSION}.jar \
    /opt/apache-jmeter-5.5/lib/ext/jmeter-websocket-samplers-${WS_PLUGIN_VERSION}.jar
