---
description: Evaluate Stanza NLP task performance with metrics
model: haiku
allowed-tools: Write(*), Read(*), Bash(*)
---

# Stanza Evaluation

Measure quality of Stanza pipelines.

## Metrics

```python
def evaluate_ner(gold_docs, pred_docs):
    """Evaluate NER using precision, recall, F1."""
    tp = fp = fn = 0
    
    for gold, pred in zip(gold_docs, pred_docs):
        gold_ents = {(e.start_char, e.end_char, e.type) for e in gold.entities}
        pred_ents = {(e.start_char, e.end_char, e.type) for e in pred.entities}
        
        tp += len(gold_ents & pred_ents)
        fp += len(pred_ents - gold_ents)
        fn += len(gold_ents - pred_ents)
    
    # Guard against division by zero
    pred_count = tp + fp
    gold_count = tp + fn
    
    precision = tp / pred_count if pred_count > 0 else 0.0
    recall = tp / gold_count if gold_count > 0 else 0.0
    f1 = 2*precision*recall / (precision + recall) if (precision + recall) > 0 else 0.0
    
    return {'precision': precision, 'recall': recall, 'f1': f1}
```

## Testing

```python
import stanza

def test_ner():
    nlp = stanza.Pipeline('en', processors='tokenize,ner')
    doc = nlp("John Smith works at Google")
    
    assert len(doc.entities) == 2
    assert doc.entities[0].type == 'PERSON'
```

## pytest Integration

```bash
pytest tests/ -v

# Runs all tests with coverage
```

Goal: Validate NLP quality before production.
