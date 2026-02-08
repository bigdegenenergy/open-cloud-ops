---
description: Evaluate CoreNLP NLP task performance with metrics and analysis
model: haiku
allowed-tools: Write(*), Read(*), Bash(*)
---

# CoreNLP Evaluation

Measure quality of CoreNLP pipelines.

## Evaluation Metrics

### NER Evaluation
```java
public double evaluateNER(List<Document> goldDocs, List<Document> predDocs) {
    int tp = 0, fp = 0, fn = 0;
    
    for (int i = 0; i < goldDocs.size(); i++) {
        // Use character offsets to track each mention uniquely
        // (not just text, to handle repeated entities correctly)
        Set<String> gold = extractEntitiesWithOffsets(goldDocs.get(i));
        Set<String> pred = extractEntitiesWithOffsets(predDocs.get(i));
        
        tp += gold.stream().filter(pred::contains).count();
        fp += pred.stream().filter(e -> !gold.contains(e)).count();
        fn += gold.stream().filter(e -> !pred.contains(e)).count();
    }
    
    // Handle division by zero
    if (tp + fp == 0 || tp + fn == 0) {
        return 0.0;  // No entities found or no ground truth
    }
    
    double precision = tp / (double)(tp + fp);
    double recall = tp / (double)(tp + fn);
    
    if (precision + recall == 0) {
        return 0.0;
    }
    
    double f1 = 2 * precision * recall / (precision + recall);
    
    return f1;
}

// Helper: Extract entities as (start_offset, end_offset, type) tuples
// This correctly handles multi-token entities as single spans (not individual tokens)
private Set<String> extractEntitiesWithOffsets(Document doc) {
    Set<String> entities = new HashSet<>();
    
    for (CoreMap sentence : doc.get(SentencesAnnotation.class)) {
        List<CoreLabel> tokens = sentence.get(TokensAnnotation.class);
        
        // Group consecutive tokens with same NER tag into entity spans
        int i = 0;
        while (i < tokens.size()) {
            CoreLabel token = tokens.get(i);
            String ner = token.get(NamedEntityTagAnnotation.class);
            
            if (ner != null && !ner.equals("O")) {
                // Start of entity span
                int startOffset = token.get(CharacterOffsetBeginAnnotation.class);
                int endOffset = token.get(CharacterOffsetEndAnnotation.class);
                
                // Extend span for consecutive tokens with same NER tag
                i++;
                while (i < tokens.size()) {
                    CoreLabel nextToken = tokens.get(i);
                    String nextNer = nextToken.get(NamedEntityTagAnnotation.class);
                    
                    if (nextNer != null && nextNer.equals(ner)) {
                        endOffset = nextToken.get(CharacterOffsetEndAnnotation.class);
                        i++;
                    } else {
                        break;
                    }
                }
                
                // Create entity key as (start, end, type) tuple
                String entitySpan = startOffset + "-" + endOffset + ":" + ner;
                entities.add(entitySpan);
            } else {
                i++;
            }
        }
    }
    return entities;
}
```

### Parsing Evaluation
```java
// Note: Document wrapper must expose CoreDocument or CoreSentence for dependency parsing
// This example assumes you can access the underlying CoreDocument from Document class
public double evaluateParsing(CoreDocument goldDoc, CoreDocument predDoc) {
    // UAS (Unlabeled Attachment Score) - percentage of words with correct head
    // LAS (Labeled Attachment Score) - percentage of words with correct head AND dependency relation
    
    if (goldDoc.sentences().size() != predDoc.sentences().size()) {
        throw new IllegalArgumentException("Gold and predicted sentence counts must match");
    }
    
    int correctHeads = 0;  // UAS: correct head attachment
    int correctLabels = 0; // LAS: correct head + dependency label
    int total = 0;
    
    // Compare dependency structures using SemanticGraph from each sentence
    for (int s = 0; s < goldDoc.sentences().size(); s++) {
        CoreSentence goldSent = goldDoc.sentences().get(s);
        CoreSentence predSent = predDoc.sentences().get(s);
        
        // Get dependency graphs
        SemanticGraph goldDeps = goldSent.dependencyParse();
        SemanticGraph predDeps = predSent.dependencyParse();
        
        // Compare heads for each word
        for (IndexedWord word : goldDeps.vertexSet()) {
            IndexedWord goldHead = goldDeps.getParent(word);
            IndexedWord predHead = predDeps.getParent(word);
            
            int goldHeadIdx = goldHead != null ? goldHead.index() : 0;  // 0 = root
            int predHeadIdx = predHead != null ? predHead.index() : 0;
            
            if (goldHeadIdx == predHeadIdx) correctHeads++;
            
            // Check dependency labels (LAS)
            GrammaticalRelation goldRel = goldDeps.getEdge(goldHead, word) != null ? 
                goldDeps.getEdge(goldHead, word).getRelation() : null;
            GrammaticalRelation predRel = predDeps.getEdge(predHead, word) != null ? 
                predDeps.getEdge(predHead, word).getRelation() : null;
            
            if (goldHeadIdx == predHeadIdx && 
                goldRel != null && goldRel.equals(predRel)) {
                correctLabels++;
            }
            total++;
        }
    }
    
    if (total == 0) return 0.0;
    
    // Return UAS (Unlabeled Attachment Score)
    // LAS would be: correctLabels / (double) total
    return correctHeads / (double) total;
}
```

## Testing

```java
@Test
public void testNERQuality() {
    NLPPipeline pipeline = NLPPipeline.forNER();
    Document doc = pipeline.process("John Smith works at Google");
    
    List<CoreEntityMention> entities = doc.namedEntities();
    assertEquals(2, entities.size());
    // CoreEntityMention.getEntityType() returns the NER tag
    assertEquals("PERSON", entities.get(0).getEntityType());
}
```

## CI/CD Integration

```xml
<!-- pom.xml -->
<plugin>
    <groupId>org.apache.maven.plugins</groupId>
    <artifactId>maven-surefire-plugin</artifactId>
    <version>2.22.2</version>
    <configuration>
        <includes>
            <include>**/*Test.java</include>
        </includes>
    </configuration>
</plugin>
```

Goal: Validate NLP task quality before deployment.
