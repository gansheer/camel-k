// camel-k: language=java

import org.apache.camel.builder.RouteBuilder;

public class XmlToJson extends RouteBuilder {

    @Override
    public void configure() throws Exception {
        // Write your routes here, for example:
        from("timer:java?period={{time:1000}}").routeId("java")
            .setBody()
                .simple("<hello>world!</hello>")
            .to("xj:identity?transformDirection=XML2JSON")
            .log("${body}");
    }
}
