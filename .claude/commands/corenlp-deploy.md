---
description: Deploy CoreNLP pipelines to production
model: haiku
allowed-tools: Write(*), Read(*), Bash(*)
---

# CoreNLP Deployment

Package and deploy CoreNLP systems.

## REST Server

```java
@RestController
@RequestMapping("/nlp")
public class NLPController {
    private NLPPipeline pipeline = NLPPipeline.forNER();
    
    @PostMapping("/analyze")
    public AnalysisResult analyze(@RequestBody String text) {
        Document doc = pipeline.process(text);
        return new AnalysisResult(doc.namedEntities());
    }
}
```

## Maven Assembly

```bash
mvn clean package assembly:single

# Creates: target/corenlp-*-jar-with-dependencies.jar
```

## Running the Fat JAR

```bash
# ⚠️ CRITICAL: CoreNLP models require significant memory
# Minimum: 2GB, Recommended: 4GB+, Large pipelines: 8GB+

# Basic execution (4GB heap)
java -Xmx4g -jar target/corenlp-*-jar-with-dependencies.jar

# Production with GC tuning
java -Xmx4g -Xms2g -XX:+UseG1GC -XX:MaxGCPauseMillis=200 \
     -jar target/corenlp-*-jar-with-dependencies.jar

# Custom JVM options via environment variable
export JAVA_OPTS="-Xmx8g -Xms4g -XX:+UseG1GC"
java $JAVA_OPTS -jar target/corenlp-*-jar-with-dependencies.jar
```

### Memory Requirements by Pipeline

| Pipeline | Minimum | Recommended | Notes |
|----------|---------|------------|-------|
| Tokenize only | 512MB | 1GB | Fast, minimal memory |
| NER + POS | 2GB | 3GB | Standard NLP tasks |
| Full (+ Parsing) | 4GB | 6GB | Includes dependency parsing |
| Multiple languages | 6GB | 8GB+ | Keeps all models in memory |

**⚠️ OutOfMemoryError Prevention**: If you see `java.lang.OutOfMemoryError: Java heap space`, increase `-Xmx` (e.g., `-Xmx8g`). Test locally with your expected input size.

## Health Check

```java
@GetMapping("/health")
public Map<String, String> health() {
    return Map.of("status", "ok", "pipeline", "ready");
}
```

Goal: Production-ready deployment with Docker, K8s, REST API.
