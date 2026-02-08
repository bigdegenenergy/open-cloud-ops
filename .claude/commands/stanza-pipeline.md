---
description: Build and optimize Stanza NLP pipelines for specific tasks
model: haiku
allowed-tools: Write(*), Read(*), Bash(*)
---

# Stanza Pipeline Optimization

Design efficient, language-aware Stanza pipelines.

## Quick Pipelines

```python
import stanza

# Sentiment
nlp = stanza.Pipeline('en', processors='tokenize,pos,lemma')

# NER
nlp = stanza.Pipeline('en', processors='tokenize,pos,lemma,ner')

# Parsing
nlp = stanza.Pipeline('en', processors='tokenize,pos,lemma,depparse')

# Full
nlp = stanza.Pipeline('en', processors='tokenize,mwt,pos,lemma,depparse,ner,coref')
```

## Multi-lingual Processing

```python
def process_multilingual(texts_dict):
    """Process different languages efficiently."""
    pipelines = {}
    results = {}
    
    for lang in set(texts_dict.values()):
        if lang not in pipelines:
            pipelines[lang] = stanza.Pipeline(lang)
    
    for text, lang in texts_dict.items():
        doc = pipelines[lang](text)
        results[text] = {
            'language': lang,
            'sentences': [s.text for s in doc.sentences],
            'entities': [e.text for e in doc.entities]
        }
    
    return results
```

## Language Detection

```python
from stanza.server import CoreNLPClient

def detect_language(text):
    """Detect language automatically."""
    client = CoreNLPClient(annotators=['tokenize', 'ssplit'])
    doc = client.annotate(text, lang='auto')
    return doc.language
```

## Custom Processor

```python
import stanza
from stanza.models.common import Doc

class CustomProcessor:
    def __init__(self, nlp_pipeline):
        self.nlp = nlp_pipeline
    
    def extract_entities_by_type(self, text):
        doc = self.nlp(text)
        entities_by_type = {}
        
        for ent in doc.entities:
            ent_type = ent.type
            if ent_type not in entities_by_type:
                entities_by_type[ent_type] = []
            entities_by_type[ent_type].append(ent.text)
        
        return entities_by_type
    
    def extract_dependencies(self, text):
        doc = self.nlp(text)
        deps = []
        
        for sent in doc.sentences:
            for word in sent.words:
                if word.head > 0:
                    deps.append({
                        'dependent': word.text,
                        'relation': word.deprel,
                        'head': sent.words[word.head - 1].text
                    })
        
        return deps
```

## Batch Processing

```python
import stanza
from multiprocessing import Pool

def process_batch(texts, lang='en', batch_size=32):
    """Process large text collections efficiently."""
    nlp = stanza.Pipeline(lang)
    
    results = []
    for i in range(0, len(texts), batch_size):
        batch = texts[i:i+batch_size]
        for text in batch:
            doc = nlp(text)
            results.append(doc)
    
    return results

# With multiprocessing
def worker_init_pipeline():
    """Initializer function for worker processes."""
    global nlp
    nlp = stanza.Pipeline('en')

def process_doc(text):
    """Worker function that uses global pipeline."""
    return nlp(text)

def process_parallel(texts, num_processes=4):
    """Process documents in parallel with efficient pipeline reuse."""
    with Pool(num_processes, initializer=worker_init_pipeline) as pool:
        results = pool.map(process_doc, texts)
    return results
```

## Performance Optimization

```python
# 1. Reuse pipeline
nlp = stanza.Pipeline('en', processors='tokenize,pos,lemma')

texts = load_texts()
for text in texts:
    doc = nlp(text)  # Efficient: reuses same pipeline

# 2. Disable unused processors
nlp = stanza.Pipeline('en', processors='tokenize,pos')  # Skip lemma, ner, depparse

# 3. Use GPU if available
nlp = stanza.Pipeline('en', use_gpu=True)

# 4. Cache results
cache = {}
def cached_process(text):
    if text not in cache:
        cache[text] = nlp(text)
    return cache[text]
```

## Testing

```python
import pytest
import stanza

@pytest.fixture
def pipeline():
    return stanza.Pipeline('en', processors='tokenize,ner')

def test_ner(pipeline):
    doc = pipeline("John Smith works at Google.")
    entities = doc.entities
    
    assert len(entities) == 2
    assert entities[0].type == 'PERSON'
    assert entities[1].type == 'ORG'

def test_multilingual():
    en_nlp = stanza.Pipeline('en')
    es_nlp = stanza.Pipeline('es')
    
    en_doc = en_nlp("Hello world")
    es_doc = es_nlp("Hola mundo")
    
    assert len(en_doc.sentences) == 1
    assert len(es_doc.sentences) == 1
```

## REST API (Thread-Safe with GIL Optimization)

```python
from flask import Flask, request, jsonify
import stanza
import threading
import torch

# ⚠️ IMPORTANT: Reduce PyTorch threading to 1 to avoid GIL contention in Flask (multi-threaded)
torch.set_num_threads(1)

app = Flask(__name__)

# Pre-load pipelines for supported languages only
SUPPORTED_LANGS = {'en', 'es', 'fr', 'zh'}

# Thread-safe pipeline storage
pipelines = {}
pipelines_lock = threading.RLock()

def init_pipelines():
    """Initialize pipelines once at startup (called from app factory)."""
    for lang in SUPPORTED_LANGS:
        with pipelines_lock:
            pipelines[lang] = stanza.Pipeline(
                lang, processors='tokenize,pos,lemma,ner'
            )
        print(f"✅ Initialized {lang} pipeline")

def create_app():
    """Application factory pattern for Flask."""
    app = Flask(__name__)
    
    # Initialize pipelines once at startup (replaces deprecated @app.before_first_request)
    with app.app_context():
        init_pipelines()
    
    @app.route('/analyze', methods=['POST'])
    def analyze():
        text = request.json['text']
        lang = request.json.get('lang', 'en')
        
        # Only use pre-loaded pipelines (no dynamic initialization)
        if lang not in pipelines:
            return {'error': f'Language {lang} not supported. Supported: {list(SUPPORTED_LANGS)}'}, 400
        
        # Use lock to ensure thread-safe access (PyTorch + GIL)
        with pipelines_lock:
            doc = pipelines[lang](text)
        
        return jsonify({
            'sentences': [s.text for s in doc.sentences],
            'entities': [{
                'text': e.text,
                'type': e.type
            } for e in doc.entities]
        })
    
    return app

if __name__ == '__main__':
    # Use WSGI server (e.g., gunicorn) in production instead of Flask's dev server
    # gunicorn -w 4 -b 0.0.0.0:5000 app:create_app()
    app = create_app()
    app.run(port=5000, threaded=False)  # Single-threaded for simplicity; use gunicorn for production
```

**⚠️ Production Notes**:
- Do NOT initialize Stanza pipelines inside request handlers. Pipelines are slow (~30-60s) and memory-intensive.
- Pre-load all needed languages at startup.
- **GIL Optimization**: Call `torch.set_num_threads(1)` to prevent PyTorch/GIL contention in multi-threaded Flask environments.
- **Thread Safety**: Use locks when accessing pipelines from multiple threads.
- **Serialization Bottleneck**: The global lock (`pipelines_lock`) serializes all NLP requests, even on multi-core machines. For high-throughput production:
  - Use **Gunicorn with sync workers**: Each worker has its own pipeline instance (no lock needed)
    ```bash
    gunicorn -w 4 -b 0.0.0.0:5000 app:create_app()
    ```
  - Or use **Celery task queue** for async processing (offload heavy NLP to worker pool)
  - Or use **TorchServe/BentoML** for specialized model serving with batching
- **Multi-process WSGI**: For production, use `gunicorn` with multiple worker processes instead of Flask's threaded mode.

**Goal: Design efficient, scalable NLP pipelines for your task.**
