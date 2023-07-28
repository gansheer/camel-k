// camel-k: language=java

import org.apache.camel.builder.RouteBuilder;

public class Test extends RouteBuilder {

    @Override
    public void configure() throws Exception {

        // Write your routes here, for example:
        from("timer:java?period={{time:1000}}").routeId("java")
            .setBody()
                .simple("Hello Camel from modified ${routeId}")
            .log("${body}");
    }
}
