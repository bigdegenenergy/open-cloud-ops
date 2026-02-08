---
description: Generate a CoreNLP Java project scaffold with example pipelines
model: haiku
allowed-tools: Write(*), Bash(*), Read(*)
---

# CoreNLP Project Scaffold

You are the **CoreNLP Scaffold Generator**. Create a complete Java NLP project.

## Context

Before scaffolding, interview the user:

1. **NLP Task**: What are you building?
   - Sentiment analysis, NER, Parsing, Tokenization, Coreference?
   
2. **Language(s)**: Which language(s)?
   - English (default), Chinese, German, Arabic, French, Spanish?
   
3. **Deployment**: How will it run?
   - Embedded in Java app, REST server, Batch processing?
   
4. **Scale**: How much text?
   - Real-time (< 100ms), batch (seconds), offline?

## Generation Checklist

- [ ] Create `pom.xml` with CoreNLP dependencies
- [ ] Create `NLPPipeline.java` — Main pipeline module
- [ ] Create `AnnotationProcessor.java` — Process annotations
- [ ] Create `NLPConfig.java` — Configuration
- [ ] Create example task (Sentiment, NER, etc.)
- [ ] Create `README.md` with quick start
- [ ] Create example data

## Project Structure

```
my-corenlp-project/
├── pom.xml                      # Maven config
├── src/main/java/
│   └── nlp/
│       ├── NLPPipeline.java     # Main pipeline
│       ├── NLPConfig.java       # Configuration
│       └── tasks/
│           ├── SentimentTask.java
│           ├── NERTask.java
│           └── ParsingTask.java
├── src/test/java/
│   └── nlp/
│       └── NLPPipelineTest.java
├── README.md
└── example_data.txt
```

## Template: pom.xml

```xml
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.nlp</groupId>
  <artifactId>corenlp-pipeline</artifactId>
  <version>1.0</version>

  <dependencies>
    <!-- CoreNLP -->
    <dependency>
      <groupId>edu.stanford.nlp</groupId>
      <artifactId>stanford-corenlp</artifactId>
      <version>4.5.7</version>
    </dependency>
    
    <!-- Language models (choose needed ones) -->
    <dependency>
      <groupId>edu.stanford.nlp</groupId>
      <artifactId>stanford-corenlp</artifactId>
      <version>4.5.7</version>
      <classifier>models</classifier>
    </dependency>

    <!-- Testing -->
    <dependency>
      <groupId>junit</groupId>
      <artifactId>junit</artifactId>
      <version>4.13</version>
      <scope>test</scope>
    </dependency>
  </dependencies>

  <build>
    <plugins>
      <plugin>
        <groupId>org.apache.maven.plugins</groupId>
        <artifactId>maven-compiler-plugin</artifactId>
        <version>3.8.1</version>
        <configuration>
          <source>11</source>
          <target>11</target>
        </configuration>
      </plugin>
    </plugins>
  </build>
</project>
```

## Template: NLPPipeline.java

```java
import edu.stanford.nlp.pipeline.*;
import edu.stanford.nlp.ling.*;
import java.util.*;

public class NLPPipeline {
    private StanfordCoreNLP pipeline;
    
    public NLPPipeline(String... annotators) {
        // Initialize pipeline with requested annotators
        Properties props = new Properties();
        props.setProperty("annotators", String.join(",", annotators));
        props.setProperty("coref.algorithm", "neural");
        this.pipeline = new StanfordCoreNLP(props);
    }
    
    public Document process(String text) {
        CoreDocument doc = new CoreDocument(text);
        pipeline.annotate(doc);
        return new Document(doc);
    }
    
    // Convenience constructors
    public static NLPPipeline forSentiment() {
        return new NLPPipeline(
            "tokenize", "ssplit", "pos", "lemma", "sentiment"
        );
    }
    
    public static NLPPipeline forNER() {
        return new NLPPipeline(
            "tokenize", "ssplit", "pos", "lemma", "ner"
        );
    }
    
    public static NLPPipeline forParsing() {
        return new NLPPipeline(
            "tokenize", "ssplit", "pos", "lemma", "depparse"
        );
    }
}

// Wrapper class (non-public to avoid multiple public classes in one file)
class Document {
    private CoreDocument doc;
    
    public Document(CoreDocument doc) {
        this.doc = doc;
    }
    
    // Return CoreLabel tokens (not List<String>)
    public List<CoreLabel> tokens() {
        List<CoreLabel> result = new ArrayList<>();
        for (CoreSentence sent : doc.sentences()) {
            result.addAll(sent.tokens());
        }
        return result;
    }
    
    public List<String> sentences() {
        List<String> result = new ArrayList<>();
        for (CoreSentence sent : doc.sentences()) {
            result.add(sent.text());
        }
        return result;
    }
    
    // Return CoreEntityMention (not CoreLabel)
    public List<CoreEntityMention> namedEntities() {
        return doc.entityMentions();
    }
}
```

## Tasks to Generate

```bash
# Sentiment analysis
/corenlp-scaffold

# Answer:
# - Task: Sentiment Analysis
# - Language: English
# - Deployment: Java app
# - Scale: Real-time
```

## Tips

1. **Start simple**: Tokenization + POS tagging
2. **Add gradually**: Add parsers/NER as needed
3. **Models take time**: First run downloads models (patience!)
4. **Thread safety**: Share pipeline across threads carefully
5. **Memory**: Models can be 500MB-2GB
6. **Caching**: Cache annotations when possible

## Output

Generate production-ready project:
- ✅ Proper Maven structure
- ✅ Multiple example tasks
- ✅ README with quick start
- ✅ Unit tests
- ✅ Performance notes

Make it immediately compilable and runnable.
