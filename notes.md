# JMS

## ARTEMIS

Fix OK

## IBM

Is there any pertinance https://developer.ibm.com/tutorials/mq-running-ibm-mq-apps-on-quarkus-and-graalvm-using-qpid-amqp-jms-classes/ ?
https://github.com/ibm-messaging/mq-dev-patterns/blob/master/amqp-qpid/qpid-quarkus/README.md

## AMQP

https://issues.redhat.com/browse/QUARKUS-2593

https://github.com/apache/camel-quarkus/blame/main/integration-tests/sjms2-qpid-amqp-client/pom.xml#L46
        <!-- The JMS client library to test with -->
        <dependency>
            <groupId>org.amqphub.quarkus</groupId>
            <artifactId>quarkus-qpid-jms</artifactId>
            <exclusions>
                <exclusion>
                    <groupId>org.apache.geronimo.specs</groupId>
                    <artifactId>geronimo-jms_2.0_spec</artifactId>
                </exclusion>
            </exclusions>
        </dependency>
        <dependency>
            <groupId>jakarta.jms</groupId>
            <artifactId>jakarta.jms-api</artifactId>
        </dependency>


> error 

{"level":"info","ts":1674232089.4608047,"logger":"camel-k.maven.build","msg":"Caused by: com.oracle.graal.pointsto.constraints.UnresolvedElementException: Discovered unresolved method during parsing: io.netty.handler.proxy.Socks5ProxyHandler.<init>(java.net.SocketAddress, java.lang.String, java.lang.String). This error is reported at image build time because class io.vertx.core.net.impl.ChannelProvider is registered for linking at image build time by command line"}

{"level":"info","ts":1674231977.277233,"logger":"camel-k.maven.build","msg":" - org.graalvm.home.HomeFinderFeature: Finds GraalVM paths and its version number"}
{"level":"info","ts":1674232089.4591255,"logger":"camel-k.maven.build","msg":"Fatal error: com.oracle.graal.pointsto.util.AnalysisError$ParsingError: Error encountered while parsing io.vertx.core.net.impl.ChannelProvider$$Lambda$f8015ee0e13e9e13cbf29cfa1aa9ec685375b9ca.handle(java.lang.Object) "}

