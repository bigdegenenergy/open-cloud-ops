---
description: Build and optimize CoreNLP annotation pipelines for specific NLP tasks
model: haiku
allowed-tools: Write(*), Read(*), Bash(*)
---

# CoreNLP Pipeline Optimization

Design efficient annotation pipelines for your NLP task.

## Pipeline Builders

### Sentiment Analysis
```java
NLPPipeline pipeline = NLPPipeline.forSentiment();
// Annotators: tokenize, ssplit, pos, lemma, sentiment
// Output: Sentiment scores per sentence
```

### Named Entity Recognition
```java
NLPPipeline pipeline = NLPPipeline.forNER();
// Annotators: tokenize, ssplit, pos, lemma, ner
// Output: Entity mentions with types (PERSON, ORG, LOCATION, etc.)
```

### Dependency Parsing
```java
NLPPipeline pipeline = NLPPipeline.forParsing();
// Annotators: tokenize, ssplit, pos, lemma, depparse
// Output: Dependency tree structures
```

### Coreference Resolution
```java
NLPPipeline pipeline = new NLPPipeline(
    "tokenize", "ssplit", "pos", "lemma", "ner", "depparse", "coref"
);
// Output: Coreference chains (who refers to whom)
```

## Custom Pipeline

```java
public class CustomPipeline extends NLPPipeline {
    public CustomPipeline() {
        super(
            "tokenize",      // Split into tokens
            "ssplit",        // Split into sentences
            "pos",           // Part-of-speech tagging
            "lemma",         // Lemmatization
            "ner",           // Named entity recognition
            "depparse"       // Dependency parsing
            // Optional: "coref", "sentiment", "kbp"
        );
    }
    
    // Custom post-processing
    public void extractNamedEntities(Document doc) {
        for (CoreLabel entity : doc.namedEntities()) {
            String text = entity.value();
            String type = entity.ner();
            System.out.println(text + " (" + type + ")");
        }
    }
}
```

## Performance Optimization

### 1. Lazy Loading
```java
Properties props = new Properties();
props.setProperty("annotators", "tokenize,ssplit,pos");
props.setProperty("enforceRequirements", "false"); // Skip missing annotators
StanfordCoreNLP pipeline = new StanfordCoreNLP(props);
```

### 2. Batch Processing
```java
List<String> documents = loadDocuments();
ExecutorService executor = Executors.newFixedThreadPool(4);

for (String text : documents) {
    executor.submit(() -> {
        Document doc = pipeline.process(text);
        processResults(doc);
    });
}

executor.shutdown();
executor.awaitTermination(1, TimeUnit.HOURS);
```

### 3. Caching Annotations
```java
private Map<String, Document> cache = new LinkedHashMap<String, Document>(16, 0.75f, true) {
    protected boolean removeEldestEntry(Map.Entry eldest) {
        return size() > 1000; // LRU cache of 1000 docs
    }
};

public Document getOrProcess(String text) {
    return cache.computeIfAbsent(text, pipeline::process);
}
```

## Language Support

| Language | Code | Availability |
|---|---|---|
| English | en | Full support |
| Chinese | zh | Full support |
| German | de | Full support |
| Spanish | es | Good support |
| French | fr | Good support |
| Arabic | ar | Limited |

```java
// Language-specific configuration
public NLPPipeline forLanguage(String lang) {
    if (lang.equals("zh")) {
        // Chinese needs different tokenization
        props.setProperty("tokenize.language", "Chinese");
    }
    return new NLPPipeline(...);
}
```

## Troubleshooting

**Issue**: Pipeline slow on first run
- **Cause**: Models loading from disk
- **Fix**: Models cache in memory after first use

**Issue**: OutOfMemory
- **Cause**: Too many heavy annotators + documents
- **Fix**: Process in smaller batches, increase heap (`-Xmx4g`)

**Issue**: Annotator not working
- **Cause**: Missing model files
- **Fix**: Download full models JAR: `stanford-corenlp-*.jar-models.jar`

## Testing

```java
@Test
public void testSentimentAnalysis() {
    NLPPipeline pipeline = NLPPipeline.forSentiment();
    Document doc = pipeline.process("This movie is great!");
    
    assertEquals(1, doc.sentences().size());
    // Sentiment: 0-4 scale (most negative to most positive)
}

@Test
public void testNER() {
    NLPPipeline pipeline = NLPPipeline.forNER();
    Document doc = pipeline.process("John Smith works at Google.");
    
    List<CoreLabel> entities = doc.namedEntities();
    assertEquals(2, entities.size());
    assertEquals("PERSON", entities.get(0).ner());
    assertEquals("ORG", entities.get(1).ner());
}
```

## REST API Wrapper

```java
// Simple REST endpoint for your pipeline
@RestController
public class NLPController {
    private NLPPipeline pipeline = NLPPipeline.forNER();
    
    @PostMapping("/analyze")
    public AnalysisResult analyze(@RequestBody String text) {
        Document doc = pipeline.process(text);
        return new AnalysisResult(doc);
    }
}
```

**Goal: Design efficient, language-appropriate NLP pipelines.**
