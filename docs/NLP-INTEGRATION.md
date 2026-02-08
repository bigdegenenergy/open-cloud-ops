# NLP Integration Guide

This toolkit now includes comprehensive support for **Stanford NLP frameworks** for both Java and Python.

## What's Included

### 1. CoreNLP (Java) 🔧
- **Best for**: Production Java applications, robust pipelines, coreference resolution
- **Tasks**: Tokenization, POS tagging, NER, dependency parsing, sentiment analysis, coreference
- **Languages**: 6 languages (English, Chinese, German, Spanish, French, Arabic)
- **Agent**: `nlp-architect` 
- **Commands**: `/corenlp-scaffold`, `/corenlp-pipeline`, `/corenlp-evaluate`, `/corenlp-deploy`

### 2. Stanza (Python) 🐍
- **Best for**: Research, quick prototyping, multi-lingual needs
- **Tasks**: Tokenization, POS tagging, NER, dependency parsing, lemmatization
- **Languages**: 70+ languages
- **Agent**: `nlp-architect`
- **Commands**: `/stanza-scaffold`, `/stanza-pipeline`, `/stanza-evaluate`, `/stanza-deploy`

### 3. Smart Sync ⚡
Automatically syncs language-specific configurations:
```
Java project (pom.xml/build.gradle) → Syncs CoreNLP agents + commands + templates
Python project (setup.py/pyproject.toml) → Syncs Stanza agents + commands + templates
Both → Syncs shared NLP documentation
```

## Quick Start

### For Java Projects

```bash
/corenlp-scaffold

# Answer interview questions
# → Full Maven project generated

# Build
mvn clean compile

# Run
java -cp target/classes:... YourNLPTask
```

### For Python Projects

```bash
/stanza-scaffold

# Answer interview questions
# → Complete Python project generated

# Install
pip install -r requirements.txt

# Run
python pipeline.py
```

## Agents

### NLP Architect
Expert in building production NLP systems with both CoreNLP and Stanza.

**Use when:**
- Designing NLP task flows
- Choosing between CoreNLP and Stanza
- Optimizing for language support
- Building multi-lingual systems
- Tuning performance vs accuracy

## Commands

### CoreNLP Commands (Java)

- **`/corenlp-scaffold`** — Generate Maven project with example tasks
- **`/corenlp-pipeline`** — Design and optimize annotation pipelines
- **`/corenlp-evaluate`** — Evaluate NLP task performance
- **`/corenlp-deploy`** — Package for production deployment

### Stanza Commands (Python)

- **`/stanza-scaffold`** — Generate Python project with example tasks
- **`/stanza-pipeline`** — Design multi-lingual pipelines
- **`/stanza-evaluate`** — Evaluate NLP task performance  
- **`/stanza-deploy`** — Deploy as REST service or PyPI package

## Comparison

| Feature | CoreNLP | Stanza |
|---|---|---|
| **Language** | Java | Python |
| **Languages Supported** | 6 | 70+ |
| **Coreference** | Mature | Emerging |
| **Dependency Parsing** | Very strong | Strong |
| **NER** | Excellent | Good |
| **Deployment** | JAR, REST | Python, REST, PyPI |
| **Real-time** | Good | Good (with GPU) |
| **Research** | Established | Modern |
| **Learning Curve** | Moderate | Easy |

## Common Tasks

### Sentiment Analysis

**CoreNLP (Java)**:
```java
NLPPipeline pipeline = NLPPipeline.forSentiment();
Document doc = pipeline.process("This movie is great!");
// Sentiment score: 0-4 scale
```

**Stanza (Python)**:
```python
from transformers import pipeline
sentiment = pipeline('sentiment-analysis')
result = sentiment("This movie is great!")
```

### Named Entity Recognition

**CoreNLP**:
```java
NLPPipeline pipeline = NLPPipeline.forNER();
Document doc = pipeline.process("John Smith works at Google");
List<CoreLabel> entities = doc.namedEntities();
```

**Stanza**:
```python
nlp = stanza.Pipeline('en', processors='tokenize,ner')
doc = nlp("John Smith works at Google")
for ent in doc.entities:
    print(f"{ent.text} ({ent.type})")
```

### Multi-lingual Processing

**CoreNLP** (limited):
```java
NLPPipeline pipeline = new NLPPipeline("zh"); // Chinese
Document doc = pipeline.process("我是约翰");
```

**Stanza** (extensive):
```python
nlp_zh = stanza.Pipeline('zh')
nlp_ar = stanza.Pipeline('ar')
nlp_vi = stanza.Pipeline('vi')

doc_zh = nlp_zh("我是约翰")
doc_ar = nlp_ar("أنا جون")
doc_vi = nlp_vi("Tôi là John")
```

## Workflow Example

### Build a Multi-lingual Sentiment Analysis System

**Step 1: Choose Framework**
- English only? CoreNLP or Stanza
- Multiple languages (20+)? Stanza
- Decision: Stanza (multi-lingual)

**Step 2: Scaffold**
```bash
/stanza-scaffold

# Task: Sentiment Analysis
# Languages: English, Spanish, French, Arabic
# Deployment: REST server
```

**Step 3: Implement**
```python
class MultilingualSentiment:
    def __init__(self):
        self.pipelines = {
            'en': stanza.Pipeline('en'),
            'es': stanza.Pipeline('es'),
            'fr': stanza.Pipeline('fr'),
            'ar': stanza.Pipeline('ar')
        }
    
    def analyze(self, text, lang):
        doc = self.pipelines[lang](text)
        return self.score_sentences(doc)
```

**Step 4: Deploy**
```bash
/stanza-deploy

# Creates Docker image
# Flask REST API
# Ready for production
```

## GitHub Actions

### Automatic Testing & Deployment

**For Java projects:**
- `mvn test` on PR
- Build compilation check
- Maven assembly creation

**For Python projects:**
- `pytest` on PR
- PyPI publish on merge

## Resources

### CoreNLP
- **Official Docs**: https://stanfordnlp.github.io/CoreNLP/
- **GitHub**: https://github.com/stanfordnlp/CoreNLP
- **Models**: Stanford NLP models homepage

### Stanza
- **Official Docs**: https://stanfordnlp.github.io/stanza/
- **GitHub**: https://github.com/stanfordnlp/stanza
- **Languages**: 70+ supported

### Shared
- **NLP Patterns**: `docs/NLP-PATTERNS.md`
- **Framework Comparison**: `docs/NLP-FRAMEWORKS-COMPARISON.md`

## Performance Tips

### CoreNLP (Java)
- Pre-load models once, reuse pipeline
- Use thread pools for batch processing
- Monitor heap size (Xmx flag)
- Cache results when possible

### Stanza (Python)
- Download models offline: `stanza.download('en')`
- Use GPU if available: `use_gpu=True`
- Batch process documents
- Implement result caching

## Language Detection

### Automatic Detection

```python
# Stanza can auto-detect language
import stanza
doc = stanza.Pipeline('auto')(text)
```

```java
// CoreNLP language detection
Properties props = new Properties();
props.setProperty("annotators", "tokenize,ssplit,langid");
StanfordCoreNLP pipeline = new StanfordCoreNLP(props);
```

## Next Steps

1. ✅ Understand CoreNLP vs Stanza
2. ✅ Pick framework for your project
3. ✅ Run `/corenlp-scaffold` or `/stanza-scaffold`
4. ✅ Implement your NLP task
5. ✅ Deploy with `/corenlp-deploy` or `/stanza-deploy`
6. ✅ Monitor quality in production

## Troubleshooting

**Q: Which should I use?**
- A: Java project + want proven solution? CoreNLP
- A: Python project + need multi-lingual? Stanza
- A: Research + flexibility? Stanza
- A: Production + robust? CoreNLP

**Q: Slow on first run?**
- A: Models loading from disk (normal)
- A: Subsequent runs much faster

**Q: OutOfMemory?**
- A: Reduce annotators (tokenize, pos only)
- A: Increase Java heap: `-Xmx4g`
- A: Process in smaller batches

---

**Questions?** Check the framework docs or ask the NLP Architect agent.

**Happy NLP building! 🚀**
