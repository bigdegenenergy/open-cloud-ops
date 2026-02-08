---
description: Generate a Stanza Python project scaffold with example pipelines
model: haiku
allowed-tools: Write(*), Bash(*), Read(*)
---

# Stanza Project Scaffold

You are the **Stanza Scaffold Generator**. Create a complete Python NLP project.

## Context

Before scaffolding, interview the user:

1. **NLP Task**: What are you building?
   - Sentiment analysis, NER, Parsing, Tokenization, POS tagging?
   
2. **Language(s)**: Which language(s)?
   - English (default), Spanish, Chinese, Arabic, 70+ supported?
   - Single or multi-lingual?
   
3. **Deployment**: How will it run?
   - Python app, REST server, Batch processing, Real-time?
   
4. **Scale**: How much text?
   - Real-time (< 100ms), batch (seconds), offline?

## Generation Checklist

- [ ] Create `requirements.txt` with Stanza + dependencies
- [ ] Create `pipeline.py` — Main pipeline module
- [ ] Create `processors.py` — Task-specific processors
- [ ] Create `config.py` — Configuration
- [ ] Create example task (Sentiment, NER, etc.)
- [ ] Create `README.md` with quick start
- [ ] Create unit tests

## Project Structure

```
my-stanza-project/
├── requirements.txt             # Python dependencies
├── config.py                    # Configuration
├── pipeline.py                  # Main NLP pipeline
├── processors/
│   ├── sentiment.py
│   ├── ner.py
│   └── parsing.py
├── tests/
│   └── test_pipeline.py
├── example_data.txt
└── README.md
```

## Template: requirements.txt

```
stanza==1.8.0
transformers==4.38.0
torch==2.2.0
numpy==1.24.0
flask==3.0.0
pytest==7.4.0
```

## Template: pipeline.py

```python
import stanza

class NLPPipeline:
    def __init__(self, lang='en', processors='tokenize,pos,lemma,depparse'):
        """
        Initialize Stanza pipeline.
        
        Args:
            lang: Language code (en, zh, es, ar, etc.)
            processors: Comma-separated processor list
        """
        self.lang = lang
        self.nlp = stanza.Pipeline(
            lang=lang,
            processors=processors,
            use_gpu=False,  # Set to True if GPU available
            download_method=None  # Set lang model path if offline
        )
    
    def process(self, text):
        """Process text and return annotated document."""
        doc = self.nlp(text)
        return Document(doc)

    @staticmethod
    def for_sentiment(lang='en'):
        """Pipeline for sentiment analysis using Stanza's native sentiment processor."""
        return NLPPipeline(lang, 'tokenize,pos,lemma,sentiment')
    
    @staticmethod
    def for_ner(lang='en'):
        """Pipeline for NER."""
        return NLPPipeline(lang, 'tokenize,pos,lemma,ner')
    
    @staticmethod
    def for_parsing(lang='en'):
        """Pipeline for dependency parsing."""
        return NLPPipeline(lang, 'tokenize,pos,lemma,depparse')

class Document:
    """Wrapper for processed document."""
    
    def __init__(self, doc):
        self.doc = doc
    
    def sentences(self):
        """Get sentences."""
        return [sent.text for sent in self.doc.sentences]
    
    def tokens(self):
        """Get all tokens."""
        tokens = []
        for sent in self.doc.sentences:
            tokens.extend([word.text for word in sent.words])
        return tokens
    
    def entities(self):
        """Get named entities."""
        entities = []
        for ent in self.doc.entities:
            entities.append({
                'text': ent.text,
                'type': ent.type,
                'start_char': ent.start_char,
                'end_char': ent.end_char
            })
        return entities
    
    def parse_tree(self, sent_idx=0):
        """Get dependency parse for sentence."""
        sent = self.doc.sentences[sent_idx]
        return {
            'words': [w.text for w in sent.words],
            'heads': [w.head for w in sent.words],
            'deprels': [w.deprel for w in sent.words]
        }
```

## Template: processors/sentiment.py

```python
from pipeline import NLPPipeline

class SentimentAnalyzer:
    """Sentiment analysis using Stanza's native sentiment processor."""
    
    def __init__(self, lang='en'):
        self.nlp = NLPPipeline.for_sentiment(lang)
    
    def analyze(self, text):
        """Analyze sentiment of text using Stanza."""
        doc = self.nlp(text)
        
        results = []
        for sentence in doc.doc.sentences:
            # Stanza sentiment: 0=very negative, 1=negative, 2=neutral, 3=positive, 4=very positive
            sentiment_labels = ['very negative', 'negative', 'neutral', 'positive', 'very positive']
            score = sentence.sentiment
            
            results.append({
                'text': sentence.text,
                'score': score,
                'label': sentiment_labels[score] if 0 <= score < len(sentiment_labels) else 'unknown'
            })
        
        return results

class MultilingualSentimentAnalyzer:
    def __init__(self):
        self.nlp_en = NLPPipeline.for_sentiment('en')
        self.nlp_es = NLPPipeline.for_sentiment('es')
        self.nlp_zh = NLPPipeline.for_sentiment('zh')
    
    def analyze(self, text, lang='en'):
        if lang == 'es':
            nlp = self.nlp_es
        elif lang == 'zh':
            nlp = self.nlp_zh
        else:
            nlp = self.nlp_en
        
        doc = nlp.process(text)
        return {'sentences': doc.sentences()}
```

## Tasks to Generate

```bash
# Sentiment analysis
/stanza-scaffold

# Answer:
# - Task: Sentiment Analysis
# - Language: English
# - Deployment: Python app
# - Scale: Batch
```

## Language Support

Stanza supports 70+ languages! Notable ones:

```python
languages = [
    'af', 'ar', 'bg', 'ca', 'zh', 'zh_hant', 'cu', 'da',
    'nl', 'en', 'et', 'fi', 'fr', 'de', 'got', 'grc',
    'hu', 'hy', 'id', 'it', 'ja', 'kk', 'ko', 'la', 'lv',
    'lt', 'nb', 'pl', 'pt', 'ro', 'ru', 'sk', 'sl', 'es',
    'sv', 'tr', 'uk', 'vi', ...
]
```

## Multi-lingual Example

```python
def process_multilingual(texts):
    """Process texts in different languages."""
    from stanza.server import CoreNLPClient
    
    # Start server with multiple language models
    client = CoreNLPClient(
        annotators=['tokenize', 'ssplit', 'pos', 'lemma'],
        properties={'ssplit.language': 'en'},
        memory='4G'
    )
    
    results = {}
    for lang, text in texts.items():
        doc = client.annotate(text, lang)
        results[lang] = doc
    
    return results
```

## Tips

1. **First run**: Models download automatically (~500MB per language)
2. **Offline**: Pre-download with `stanza.download('en')`
3. **GPU**: Set `use_gpu=True` if CUDA available
4. **Batch processing**: Process multiple documents efficiently
5. **Memory**: Models cache in memory (1-2GB common)

## Output

Generate production-ready project:
- ✅ Proper Python structure
- ✅ Multiple example tasks
- ✅ README with quick start
- ✅ Unit tests
- ✅ Multi-lingual support

Make it immediately runnable: `python pipeline.py`
