// camel-k: language=kotlin
// camel-k: trait=quarkus.package-type=fast-jar
// camel-k: trait=quarkus.package-type=native

// Write your routes here, for example:
from("timer:kotlin?period=1000")
  .routeId("kotlin")
  .setBody()
    .simple("Hello Camel K from \${routeId}!")
  .to("log:info")
