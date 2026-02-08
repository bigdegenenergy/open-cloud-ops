---
description: Deploy Stanza NLP pipelines to production
model: haiku
allowed-tools: Write(*), Read(*), Bash(*)
---

# Stanza Deployment

Package and deploy Stanza systems.

## REST API (Thread-Safe)

```python
from flask import Flask, request, jsonify
import stanza
import threading
import torch

# ⚠️ IMPORTANT: Reduce PyTorch threading to avoid GIL contention
torch.set_num_threads(1)

app = Flask(__name__)
nlp = stanza.Pipeline('en')
nlp_lock = threading.RLock()  # Protect pipeline from concurrent access

@app.route('/analyze', methods=['POST'])
def analyze():
    text = request.json['text']
    
    # Use lock for thread-safe pipeline access
    with nlp_lock:
        doc = nlp(text)
    
    return jsonify({
        'entities': [
            {'text': e.text, 'type': e.type}
            for e in doc.entities
        ]
    })

@app.route('/health')
def health():
    return {'status': 'ok'}

if __name__ == '__main__':
    # For production, use gunicorn with sync workers instead of Flask dev server:
    # gunicorn -w 4 -b 0.0.0.0:5000 app:app
    app.run(host='127.0.0.1', port=5000, threaded=False)
```

**⚠️ Production Deployment Notes**:
- Use **Gunicorn with sync workers** (each worker has own pipeline, no lock needed)
- Set `torch.set_num_threads(1)` to prevent PyTorch/GIL issues
- Use `host='127.0.0.1'` in production (not '0.0.0.0' unless behind reverse proxy)

**Performance Note**: Since requests are locked per-process when using threading mode, horizontal scaling via multiple Gunicorn worker processes is the intended path for handling concurrent traffic:
```bash
gunicorn -w 8 -b 0.0.0.0:5000 --worker-class sync app:app
```
Each worker runs independently with its own Stanza pipeline instance, eliminating lock contention. Use `--workers` (typically 2-4× CPU cores) based on your traffic patterns.

## PyPI Package

```bash
python setup.py sdist bdist_wheel
twine upload dist/*
```

Goal: Production deployment with REST API and PyPI publishing.
